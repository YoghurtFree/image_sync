# image-sync

容器镜像同步 HTTP 服务，支持在不同 Registry 之间异步同步镜像。

## 技术栈

- Go 1.25
- [Gin](https://github.com/gin-gonic/gin) v1.12 — HTTP 框架
- [go-containerregistry](https://github.com/google/go-containerregistry) v0.21.7 — crane 镜像操作
- [Asynq](https://github.com/hibiken/asynq) v0.26 — Redis-backed 异步任务队列

## 快速开始

### 前置依赖

- Go 1.25+
- Redis

### 构建与运行

```bash
go build -o image-sync .
./image-sync --config ./config.yaml
```

服务启动后同时运行 HTTP Server（默认 :8080）和 Asynq Worker。

## 配置

`config.yaml` 示例：

```yaml
server:
  port: 8080
redis:
  addr: localhost:6379
  password: ""
asynq:
  concurrency: 10
registries:
  harbor:
    url: harbor.example.com
    username: admin
    password: secret
  acr:
    url: registry.cn-hangzhou.aliyuncs.com
    username: ""
    password: ""
```

- `registries` 是一个 map，key 为 alias（如 `harbor`），API 请求中通过 alias 引用
- `asynq.concurrency` 控制 Worker 并发数

## API

### 提交同步任务

```
POST /api/v1/sync
```

请求体：

```json
{
  "src": "harbor",
  "dst": "acr",
  "image": "library/nginx",
  "tag": "1.25",
  "priority": "high"
}
```

| 字段 | 必填 | 说明 |
|------|------|------|
| `src` | 是 | 源 Registry alias |
| `dst` | 是 | 目标 Registry alias |
| `image` | 是 | 镜像名（如 `library/nginx`） |
| `tag` | 是 | 镜像 tag |
| `priority` | 否 | `high` / `default` / `low`，默认 `default` |

响应 `202 Accepted`：

```json
{
  "task_id": "abc123"
}
```

### 查询任务状态

```
GET /api/v1/tasks/:id
```

响应 `200 OK`：

```json
{
  "task_id": "abc123",
  "status": "completed",
  "completed_at": "2026-07-05T12:00:00Z"
}
```

`status` 取值：`pending` / `processing` / `completed` / `failed`。任务不存在时返回 `404`。

## 架构

```
POST /api/v1/sync → Gin 校验 → Asynq 入队 → Redis
                                              ↓
                    GET /api/v1/tasks/:id ← Asynq Worker
                                              ↓
                                        crane.Pull(src)
                                              ↓
                                        crane.Push(dst)
```

- 异步处理：接口立即返回 task_id，后台执行镜像拷贝
- Asynq 优先级队列：`high` → critical(6) / `default` → default(3) / `low` → low(1)，权重比决定 Worker 消费速率
- 任务最大重试 3 次（Asynq MaxRetry）
- src/dst 使用独立认证（Pull + Push 而非 crane.Copy）
