## Tasks

- [x] 1. 项目初始化：Go module 依赖、目录结构、配置加载（YAML 配置文件含 Registry 凭据）
- [x] 2. 镜像同步引擎：go-containerregistry 封装，执行 crane.Copy，凭据从配置注入
- [x] 3. Asynq 任务队列集成：Worker 注册、任务分发、优先级队列、Inspector 查询
- [x] 4. 同步 API 端点：POST /sync 提交任务，GET /task/:id 查询状态
- [ ] 5. 错误处理与重试：失败任务记录、重试策略
- [ ] 6. 多实例部署验证：Asynq 持久化、任务不丢失
