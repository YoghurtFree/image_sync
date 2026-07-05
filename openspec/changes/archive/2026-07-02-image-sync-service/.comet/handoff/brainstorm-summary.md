# Brainstorm Summary

- Change: image-sync-service
- Date: 2026-07-02

## 确认的技术方案

单进程架构：Gin HTTP Server + Asynq Worker 在同一进程。

API 端点：
- POST /api/v1/sync — 提交同步任务，202 返回 task_id
- GET /api/v1/tasks/:id — 查询任务状态

请求格式：{src, dst, image, tag, priority}，image 以用户传入为准不补全，priority 默认 normal。

配置：YAML 文件，flag 指定路径 + 默认 ./config.yaml。包含 server、redis、asynq、registries 四个 section。

镜像拷贝：go-containerregistry crane.Copy，凭据通过 authn.FromConfig 注入。
任务队列：Asynq，三优先级 critical/default/low，重试 3 次指数退避。
状态查询：asynq.Inspector，不直接操作 Redis。

目录结构：main.go / config/ / handler/ / worker/ / syncer/

## 关键取舍与风险

- 单进程部署简单，但 HTTP 和 Worker 绑定，扩缩容粒度粗
- 配置文件明文密码，依赖部署环境文件权限保护
- image 字段不补全，用户自行传完整路径

## 测试策略

- 单元测试：config 加载、syncer 封装
- 集成测试：Asynq 任务分发 + 执行（需要 Redis）
- 端到端：启动服务，POST 同步任务，查询状态

## Spec Patch

无
