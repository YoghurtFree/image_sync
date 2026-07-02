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
