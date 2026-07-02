## Architecture

```
Client → Gin HTTP Server → Asynq Client → Redis (Asynq internal)
                                          ↑
                              Asynq Worker (N instances)
                                    │
                              crane.Copy (pull + push)
                                    │
                              Source Registry ↔ Target Registry
```

## Components

### 1. HTTP Layer (Gin)

API 端点：

| Method | Path | 说明 |
|--------|------|------|
| POST | `/api/v1/sync` | 提交同步任务 |
| GET | `/api/v1/tasks/:id` | 查询任务状态 |

### 2. Task Queue (Asynq)

- 队列优先级：`critical`(high) / `default`(normal) / `low`(low)
- 任务类型：`image:sync`
- 并发控制：每个 worker 最大并发数可配置
- 重试策略：指数退避，最大 3 次
- 任务状态查询：通过 `asynq.Inspector`，不直接操作 Redis

### 3. Image Copy Engine (go-containerregistry)

- 使用 `crane.Copy` 执行镜像拷贝
- 支持并行 layer 传输（go-containerregistry 内置）
- 认证通过 `authn.FromConfig` 注入

### 4. Registry Credential Store

存储方案：配置文件（YAML/TOML）

```yaml
registries:
  harbor-prod:
    url: registry.example.com
    username: admin
    password: secret
  acr-cn:
    url: xxx.azurecr.io
    username: xxx
    password: xxx
```

启动时加载到内存，运行时不直接读写配置文件。

## Data Flow

```
1. POST /sync {src: "harbor-prod", dst: "acr-cn", image: "nginx", tag: "1.25", priority: "high"}
2. 从内存 registry map 读取 src/dst 凭据
3. 构建 asynq.Task{Type: "image:sync", Payload: {src, dst, image, tag}}
4. asynq.Enqueue(task, asynq.Queue("critical"))
5. Worker 消费任务 → crane.Copy(src, dst)
6. 任务结果由 Asynq 自动管理
7. GET /task/:id → asynq.Inspector.GetTaskInfo(id)
```

## Error Handling

- 凭据不存在（配置中无对应 registry name）：400 Bad Request
- 同步失败：Asynq 重试 3 次后标记 failed
- Registry 不可达：Worker 重试 3 次后标记失败

## Multi-Instance Deployment

- Gin Server 无状态，可水平扩展
- Asynq Worker 无状态，Redis 保证任务不丢失
- 所有状态由 Asynq 管理，不直接操作 Redis
