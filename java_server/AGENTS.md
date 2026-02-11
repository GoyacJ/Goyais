# java_server AGENTS (Design Phase)

当前模块处于设计阶段（design phase）。编码前必须先通过设计门禁。

## Design Gate (MUST)

- 输出接口草案：API、DTO、错误码、分页语义。
- 输出数据模型草案：实体、状态机、迁移策略。
- 输出 DoD、测试矩阵、回滚策略。
- 与 `docs/prd.md`、`go_server/docs/*` 契约口径对齐，避免跨语言漂移。

## Ready for Coding 条件

- 需求边界冻结。
- 契约评审通过。
- 风险清单与 fallback 方案已确认。
