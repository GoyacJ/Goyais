# Risk Matrix

用于将“下一步计划”中的风险显式结构化。

| Risk ID | Level (P0/P1/P2) | Trigger | Impact | Mitigation | Rollback | Owner | Evidence Path | Evidence Command |
|---|---|---|---|---|---|---|---|---|
| R-001 | P0 | 契约未同步 | 回归失败、接口漂移 | 同提交更新五份契约文档 | 回退路由到上一稳定实现 | thread owner | /absolute/path | `rg ...` |
| R-002 | P1 | provider 环境不稳定 | 集成测试波动 | 增加 env gate + 重试策略 | 切回 minimal/sqlite 路径 | thread owner | /absolute/path | `go test ...` |
| R-003 | P1 | 占位域误判为已实现 | 计划失真 | 强制 implementation scan matrix | 标记 deferred 并拆新 slice | thread owner | /absolute/path | `bash .../implementation_scan.sh` |

## 使用规则

- 每个风险至少一条证据（绝对路径 + 命令）。
- 若风险无法量化，标记 `unknown` 并说明原因。
- 计划交付前必须存在可执行回滚步骤。
