# python_server AGENTS (Design Phase)

当前模块处于设计阶段（design phase）。编码前必须先通过设计门禁。

## Design Gate (MUST)

- 输出服务接口与模块边界（含依赖隔离策略）。
- 输出数据模型、状态机、错误模型。
- 输出 DoD、测试矩阵、回滚与兼容策略。
- 与 `docs/prd.md` 及 Go 契约文档保持语义一致。

## Ready for Coding 条件

- 契约已冻结并审阅通过。
- 关键风险项有监控与降级策略。
