# Thread #2 执行步骤（建议顺序）

1. 对齐冻结约束：先过 `AGENTS.md` 与 `docs/arch/overview.md`。
2. 定义 profile：
   - minimal: `sqlite + memory + local + mediamtx`
   - full: `postgres + redis/redis_stack + minio + mediamtx`
3. 实现配置读取与优先级：`ENV > YAML > default`。
4. 建立 Command Gate 骨架与统一错误模型。
5. 建立授权链骨架（含 command gate + tool gate）。
6. 落地 single-binary 静态服务策略与路由优先级。
7. 执行 single-binary 验收脚本，修正阻断项。
8. 若有契约变化，同步文档并补齐验收证据。
