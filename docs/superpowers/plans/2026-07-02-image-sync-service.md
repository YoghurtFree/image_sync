---
change: image-sync-service
design-doc: docs/superpowers/specs/2026-07-02-image-sync-service-design.md
base-ref: none
---

# Image Sync Service — Implementation Plan

## Overview

实现一个 HTTP 服务，用户通过接口触发容器镜像在两个 Registry 之间的同步。单进程架构，Gin HTTP Server + Asynq Worker。

## Task Breakdown

### Task 1: 项目初始化

Go module 依赖、目录结构、配置加载。

**Files:**
- `go.mod` — 更新依赖
- `config/config.go` — Config 结构体 + YAML 加载
- `config.yaml` — 示例配置文件

**Steps:**
1. `go get` 安装依赖：gin、go-containerregistry、asynq、go-redis
2. 创建 `config/config.go`：
   - Config 结构体（Server, Redis, Asynq, Registries）
   - `Load(path string) (*Config, error)` 函数
3. 创建 `config.yaml` 示例文件

**Acceptance:**
- `go build ./...` 通过
- 能加载示例配置文件并打印到 stdout

### Task 2: 镜像同步引擎

封装 go-containerregistry 的 crane.Copy。

**Files:**
- `syncer/syncer.go` — Copy 函数

**Steps:**
1. 创建 `syncer/syncer.go`：
   - `Copy(srcImage, dstImage string, srcAuth, dstAuth authn.Authenticator) error`
   - 构建完整镜像引用（registry/image:tag）
   - 调用 `crane.Copy` 执行拷贝
2. 创建 `syncer/syncer_test.go`：单元测试（mock 或集成测试）

**Acceptance:**
- 能对两个 registry 执行 crane.Copy
- 错误情况返回有意义的 error

### Task 3: Asynq 任务队列集成

Worker 注册、任务分发、优先级队列。

**Files:**
- `worker/worker.go` — Worker 注册 + 任务处理函数

**Steps:**
1. 创建 `worker/worker.go`：
   - `NewServer(cfg config.Config) *asynq.Server`
   - `NewClient(cfg config.Config) *asynq.Client`
   - `HandleImageSync(ctx context.Context, t *asynq.Task) error` — 任务处理函数
   - 从 payload 解析 src/dst/image/tag
   - 从配置获取凭据
   - 调用 syncer.Copy
2. 优先级队列配置：critical=6, default=3, low=1

**Acceptance:**
- Worker 能注册并消费 `image:sync` 任务
- 高优先级任务先于低优先级执行

### Task 4: 同步 API 端点

Gin 路由处理。

**Files:**
- `handler/handler.go` — 路由注册 + 请求处理

**Steps:**
1. 创建 `handler/handler.go`：
   - `RegisterRoutes(r *gin.Engine, client *asynq.Client, cfg config.Config)`
   - POST /api/v1/sync：校验参数 → 获取凭据 → Enqueue → 返回 task_id
   - GET /api/v1/tasks/:id：Inspector.GetTaskInfo → 返回状态
2. 请求/响应结构体定义

**Acceptance:**
- POST /api/v1/sync 返回 202 + task_id
- GET /api/v1/tasks/:id 返回任务状态
- 参数错误返回 400

### Task 5: main.go 入口

组装所有组件。

**Files:**
- `main.go` — 入口函数

**Steps:**
1. 解析 `--config` flag（默认 `./config.yaml`）
2. 加载配置
3. 创建 Asynq server + client
4. 注册 worker handler
5. 启动 Asynq worker（goroutine）
6. 注册 Gin 路由
7. 启动 HTTP server

**Acceptance:**
- `go run main.go --config config.yaml` 启动成功
- HTTP 端点可访问

### Task 6: 错误处理与重试

完善错误处理和重试策略。

**Files:**
- `worker/worker.go` — 重试配置
- `handler/handler.go` — 错误响应

**Steps:**
1. Asynq 重试配置：3 次指数退避
2. Worker 错误处理：registry 不可达、镜像不存在
3. HTTP 错误响应标准化

**Acceptance:**
- 同步失败自动重试 3 次
- 最终失败返回有意义的错误信息

## Dependency Order

```
Task 1 (config)
  → Task 2 (syncer)
    → Task 3 (worker)
      → Task 4 (handler)
        → Task 5 (main.go)
          → Task 6 (error handling)
```

## Build & Test

```bash
go build ./...
go test ./...
```
