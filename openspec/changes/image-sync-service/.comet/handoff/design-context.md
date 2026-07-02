# Comet Design Handoff

- Change: image-sync-service
- Phase: design
- Mode: compact
- Context hash: b8e17b4eb081770b4a9a40faaebdb23f6268ddd2bb08406f58f45446d84448e2

Generated-by: comet-handoff.sh

OpenSpec remains the canonical capability spec. This handoff is a deterministic, source-traceable context pack, not an agent-authored summary.

## openspec/changes/image-sync-service/proposal.md

- Source: openspec/changes/image-sync-service/proposal.md
- Lines: 1-25
- SHA256: b19aabcd7d1d04e0a9164f979ee360e433b85f9f7cdb7ce6e4e5f9307003f73f

```md
## Why

用户需要一个 HTTP 服务，通过接口触发容器镜像在两个 Registry 之间的同步。当前缺乏统一的镜像同步入口，手动操作效率低且无法多实例协作。

## What Changes

- 新增 HTTP API 服务（Gin），提供镜像同步接口
- 新增异步任务队列（Asynq + Redis），支持 high/normal/low 优先级
- 新增镜像拷贝引擎（go-containerregistry），执行实际的 pull + push 操作
- Registry 凭据通过配置文件管理，不对外暴露
- 支持多实例部署，任务状态持久化，不丢失任务

## Capabilities

### New Capabilities
- `image-sync`: 镜像同步任务的提交、执行、状态查询

### Modified Capabilities

## Impact

- 新增依赖：gin v1.12.0、go-containerregistry v0.21.7、asynq v0.26.0、go-redis v9.21.0
- 基础设施依赖：Redis 7.x（仅 Asynq 使用）
- 新增 API 端点：`/sync`、`/task/:id`
- 配置文件新增 Registry 凭据配置
```

## openspec/changes/image-sync-service/design.md

- Source: openspec/changes/image-sync-service/design.md
- Lines: 1-78
- SHA256: f32faab787755227462ebe6fbddcc06db41834018f4241ab9a3fd9c8175c74db

```md
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
```

## openspec/changes/image-sync-service/tasks.md

- Source: openspec/changes/image-sync-service/tasks.md
- Lines: 1-8
- SHA256: 21659f2740afe52c6ac5fdc4f0c1ac4e438b0c9b9ae5c32189ed4ef9eec110ef

```md
## Tasks

- [ ] 1. 项目初始化：Go module 依赖、目录结构、配置加载（YAML 配置文件含 Registry 凭据）
- [ ] 2. 镜像同步引擎：go-containerregistry 封装，执行 crane.Copy，凭据从配置注入
- [ ] 3. Asynq 任务队列集成：Worker 注册、任务分发、优先级队列、Inspector 查询
- [ ] 4. 同步 API 端点：POST /sync 提交任务，GET /task/:id 查询状态
- [ ] 5. 错误处理与重试：失败任务记录、重试策略
- [ ] 6. 多实例部署验证：Asynq 持久化、任务不丢失
```

## openspec/changes/image-sync-service/specs/image-sync/spec.md

- Source: openspec/changes/image-sync-service/specs/image-sync/spec.md
- Lines: 1-64
- SHA256: 997a8e2db320d834c06d2356d873dae61712b5722cdac1c99105237eeca4f606

```md
## ADDED Requirements

### Requirement: Submit sync task
The system SHALL allow users to submit an image sync task specifying source registry name, target registry name, image name, tag, and priority. The system MUST return a task ID immediately. Registry credentials MUST be resolved from a config file loaded at startup.

#### Scenario: Successful submission
- **WHEN** user POST /api/v1/sync with {src, dst, image, tag, priority}
- **THEN** system enqueues task to Asynq and returns 202 with {task_id}

#### Scenario: Source registry not configured
- **WHEN** user submits a sync task with a source registry name not in config
- **THEN** system returns 400 Bad Request

#### Scenario: Target registry not configured
- **WHEN** user submits a sync task with a target registry name not in config
- **THEN** system returns 400 Bad Request

### Requirement: Priority queue ordering
The system SHALL process tasks by priority: high tasks MUST be processed before normal, normal before low.

#### Scenario: High priority jumps queue
- **WHEN** a low-priority task is pending and a high-priority task is submitted
- **THEN** the high-priority task MUST be processed first

### Requirement: Query task status
The system SHALL allow users to query the status of a sync task by task ID via Asynq Inspector.

#### Scenario: Task in progress
- **WHEN** user GET /api/v1/tasks/:id and task is running
- **THEN** system returns 200 with {status: "processing"}

#### Scenario: Task completed
- **WHEN** user GET /api/v1/tasks/:id and task succeeded
- **THEN** system returns 200 with {status: "completed", completed_at}

#### Scenario: Task failed
- **WHEN** user GET /api/v1/tasks/:id and task failed
- **THEN** system returns 200 with {status: "failed", error}

#### Scenario: Task not found
- **WHEN** user queries a non-existent task
- **THEN** system returns 404 Not Found

### Requirement: Image copy execution
The system SHALL use go-containerregistry to pull the image from the source registry and push to the target registry. The system MUST support parallel layer transfer.

#### Scenario: Successful copy
- **WHEN** a sync task executes and both registries are reachable
- **THEN** system copies all layers and manifest, marks task as completed

#### Scenario: Source registry unreachable
- **WHEN** a sync task executes and source registry is unreachable
- **THEN** system retries up to 3 times, then marks task as failed with error

#### Scenario: Target registry unreachable
- **WHEN** a sync task executes and target registry is unreachable
- **THEN** system retries up to 3 times, then marks task as failed with error

### Requirement: Multi-instance deployment
The system SHALL support deploying multiple worker instances. All task state MUST be managed by Asynq so no task is lost on instance failure.

#### Scenario: Worker instance restarts
- **WHEN** a worker instance crashes while processing a task
- **THEN** Asynq MUST re-queue the task for another worker to pick up
```

