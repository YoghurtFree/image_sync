# image-sync

容器镜像同步 HTTP 服务，支持在不同 Registry 之间同步镜像。

## 技术栈

- Go 1.25, Gin v1.12, go-containerregistry v0.21.7, Asynq v0.26
- Redis（仅通过 Asynq 使用，不直接操作 Redis）

## 构建与测试

```bash
go build ./...
go test -race ./...
go vet ./...
```

## 架构

单进程：Gin HTTP 服务 + Asynq Worker 运行在同一二进制中。

```
┌─────────────────────────────────────────────────────────────────────┐
│                        image-sync-service                           │
│                                                                     │
│  ┌──────────────────────────────────────────────────────────────┐   │
│  │                      Gin HTTP Server (:8080)                 │   │
│  │                                                              │   │
│  │   POST /api/v1/sync          GET /api/v1/tasks/:id          │   │
│  │   ┌─────────────────┐        ┌─────────────────┐            │   │
│  │   │   handleSync    │        │  handleTaskStatus│            │   │
│  │   │  - 校验请求      │        │  - Inspector 查询 │            │   │
│  │   │  - 检查 registry │        │  - 遍历 3 个队列  │            │   │
│  │   │  - Enqueue 任务  │        │  - 返回状态       │            │   │
│  │   └────────┬────────┘        └─────────────────┘            │   │
│  └────────────┼─────────────────────────────────────────────────┘   │
│               │                                                     │
│               ▼                                                     │
│  ┌─────────────────────┐        ┌──────────────────────────────┐   │
│  │   asynq.Client      │───────▶│         Redis                │   │
│  │   (入队)             │        │                              │   │
│  └─────────────────────┘        │  asynq:{critical}:pending   │   │
│                                 │  asynq:{default}:pending    │   │
│  ┌─────────────────────┐        │  asynq:{low}:pending        │   │
│  │   asynq.Server      │◀───────│                              │   │
│  │   (Worker, 并发=10)  │        └──────────────────────────────┘   │
│  │                     │                                            │
│  │   Queues:           │                                            │
│  │    critical: 6      │                                            │
│  │    default:  3      │                                            │
│  │    low:      1      │                                            │
│  └────────┬────────────┘                                            │
│           │                                                         │
│           ▼                                                         │
│  ┌─────────────────────────────────────────────────────┐           │
│  │              makeHandleImageSync                     │           │
│  │                                                     │           │
│  │   payload{src, dst, image, tag}                     │           │
│  │         │                                           │           │
│  │         ▼                                           │           │
│  │   cfg.Registries[src] → srcAuth                     │           │
│  │   cfg.Registries[dst] → dstAuth                     │           │
│  │         │                                           │           │
│  │         ▼                                           │           │
│  │   syncer.Copy(srcImage, dstImage, srcAuth, dstAuth) │           │
│  └────────────────────────┬────────────────────────────┘           │
│                           │                                         │
│                           ▼                                         │
│  ┌─────────────────────────────────────────────────────┐           │
│  │                   syncer.Copy                        │           │
│  │                                                     │           │
│  │   crane.Pull(srcImage, srcAuth)                     │           │
│  │         │                                           │           │
│  │         ▼                                           │           │
│  │   crane.Push(img, dstImage, dstAuth)                │           │
│  └─────────────────────────────────────────────────────┘           │
│                                                                     │
└─────────────────────────────────────────────────────────────────────┘

        ┌───────────────┐                        ┌───────────────┐
        │  Source        │                        │  Destination   │
        │  Registry      │                        │  Registry      │
        │               │                        │               │
        │  alpine:latest │─────── pull ──────────▶│               │
        │               │        push ──────────▶│  alpine:latest │
        └───────────────┘                        └───────────────┘
```

**数据流**：

1. 用户 `POST /api/v1/sync` → Gin 校验请求 → asynq.Client 入队到 Redis
2. asynq.Server 从 Redis 消费任务 → `makeHandleImageSync` 解析 payload
3. 从 `config.yaml` 读取 src/dst registry 认证信息
4. `syncer.Copy` → `crane.Pull` 从源仓库拉镜像 → `crane.Push` 推到目标仓库
5. 用户 `GET /api/v1/tasks/:id` → asynq.Inspector 查询任务状态

**核心设计**：

- 异步处理：接口立即返回 202 + task_id，后台执行同步
- Registry 凭据存储在 `config.yaml`，API 请求通过 alias 引用
- 优先级队列：high→critical, default→default, low→low
- 使用 crane.Pull + crane.Push 而非 crane.Copy（src/dst 需要独立认证）

## 包职责

| 包 | 职责 |
|---|------|
| `config/` | YAML 配置加载，RegistryConfig 提供 BasicAuth() |
| `syncer/` | 镜像拷贝：crane.Pull + crane.Push |
| `worker/` | Asynq Server/Client/Handler，任务类型 `image:sync` |
| `handler/` | Gin 路由、请求校验、任务状态查询 |
| `main.go` | 入口，启动 Worker + HTTP Server |

## 项目约定

- 配置通过 `--config` 指定，默认 `./config.yaml`
- Asynq 队列权重：critical(6), default(3), low(1)
- MaxRetry(3) 入队重试
- 错误包装：`fmt.Errorf("context: %w", err)`
