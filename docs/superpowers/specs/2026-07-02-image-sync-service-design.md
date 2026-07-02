---
comet_change: image-sync-service
role: technical-design
canonical_spec: openspec
---

# Image Sync Service — Technical Design

## 概述

HTTP 服务，用户通过接口触发容器镜像在两个 Registry 之间的同步。异步执行，支持优先级，多实例部署不丢任务。

## 架构

```
Client → Gin HTTP Server → Asynq Client → Redis (Asynq internal)
                                          ↑
                              Asynq Worker (same process)
                                    │
                              crane.Copy (pull + push)
                                    │
                              Source Registry ↔ Target Registry
```

单进程部署：HTTP Server 和 Asynq Worker 在同一进程内，通过 Asynq 共享 Redis 连接。

## API 设计

### POST /api/v1/sync

提交镜像同步任务。

请求体：
```json
{
  "src": "harbor-prod",
  "dst": "acr-cn",
  "image": "library/nginx",
  "tag": "1.25",
  "priority": "high"
}
```

- src/dst：配置文件中的 registry 名称
- image：完整镜像路径，以用户传入为准，不自动补全
- tag：镜像标签
- priority：high/normal/low，不传默认 normal

响应 202：
```json
{
  "task_id": "asynq-task-uuid"
}
```

错误响应：
- 400：src/dst 在配置中不存在、请求参数缺失
- 500：Asynq 入队失败

### GET /api/v1/tasks/:id

查询任务状态。

响应 200：
```json
{
  "task_id": "xxx",
  "status": "completed",
  "completed_at": "2026-07-02T10:30:00Z"
}
```

status 枚举：pending / processing / completed / failed

错误响应：
- 404：任务不存在

## 配置

YAML 格式，启动时通过 `--config` flag 指定路径，默认 `./config.yaml`。

```yaml
server:
  port: 8080

redis:
  addr: localhost:6379
  password: ""

asynq:
  concurrency: 10

registries:
  harbor-prod:
    url: registry.example.com
    username: admin
    password: secret
```

密码明文存储，依赖部署环境文件权限保护。

## 组件设计

### 1. config/config.go

负责加载 YAML 配置文件，提供全局 Config 结构体。

```go
type Config struct {
    Server    ServerConfig
    Redis     RedisConfig
    Asynq     AsynqConfig
    Registries map[string]RegistryConfig
}

type RegistryConfig struct {
    URL      string
    Username string
    Password string
}
```

### 2. handler/handler.go

Gin 路由注册和请求处理。

- POST /api/v1/sync → 从配置获取凭据 → 构建 Asynq task → Enqueue → 返回 task_id
- GET /api/v1/tasks/:id → asynq.Inspector.GetTaskInfo → 返回状态

### 3. worker/worker.go

Asynq Worker 注册和任务处理。

- 注册 `image:sync` 任务类型
- 从 payload 解析 src/dst/image/tag
- 从配置获取凭据
- 调用 syncer 执行拷贝

### 4. syncer/syncer.go

封装 go-containerregistry 的 crane.Copy。

```go
func Copy(srcImage, dstImage string, srcAuth, dstAuth authn.Authenticator) error
```

- 构建完整镜像引用（registry + image + tag）
- 通过 authn.FromConfig 注入凭据
- 调用 crane.Copy 执行并行 layer 传输

### 5. main.go

入口函数：

1. 解析 --config flag
2. 加载配置
3. 创建 Asynq server + client
4. 注册 worker handler
5. 启动 Asynq worker（goroutine）
6. 注册 Gin 路由
7. 启动 HTTP server

## 任务队列

Asynq 配置：
- 队列优先级：critical=6, default=3, low=1
- 并发数：配置文件 asynq.concurrency
- 重试：3 次，指数退避
- 任务类型标识：`image:sync`

## 错误处理

- 配置中 registry 不存在：HTTP 400
- Registry 不可达：Worker 重试 3 次后标记 failed
- 镜像不存在：Worker 重试 3 次后标记 failed
- 目标 registry 写入失败：Worker 重试 3 次后标记 failed

## 依赖

| 依赖 | 版本 | 用途 |
|------|------|------|
| gin | v1.12.0 | HTTP 框架 |
| go-containerregistry | v0.21.7 | 镜像拷贝 |
| asynq | v0.26.0 | 任务队列 |
| go-redis | v9.21.0 | Asynq 底层依赖 |

## 测试策略

- 单元测试：config 加载、syncer 封装（mock registry）
- 集成测试：Asynq 任务分发 + 执行（需要 Redis）
- 端到端：启动服务，POST 同步任务，轮询状态至完成
