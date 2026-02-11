# Rule 06: Java Implementation + Python/Flutter Design Gates

## Trigger Conditions

- `java_server`、`python_server`、`flutter_mobile` 需求推进。

## Hard Constraints (MUST)

- Java：
  - 当前阶段为实现期，默认单应用拓扑（`single`）并支持 `resource-only` 扩展模式。
  - Auth + Resource 可同进程运行，但模块边界必须保留（运行时合并，模块不变）。
  - 动态权限必须基于 `policyVersion`，并具备 Redis 失效广播能力。
  - 数据权限首期固定为 SQL 行级过滤。
- Python/Flutter：
  - 继续执行设计门禁：接口草案、数据模型、DoD、验收矩阵、回滚策略。
- 三端统一约束：
  - 与 `docs/prd.md` 与 Go 契约语义对齐。
  - 未满足门禁要求不得进入合并流程。

## Counterexamples

- Java 回退为双应用强耦合运行（必须先启动独立 auth 才能跑 API）。
- Java 未接入动态权限却声明支持 policyVersion 实时生效。
- Python/Flutter 无契约草案直接搭框架。

## Validation Commands

- `rg -n 'topology-mode|policyVersion|invalidation' java_server -S`
- `bash java_server/scripts/ci/java_javadoc_check.sh`
- `rg -n 'Design Gate|DoD|验收' python_server flutter_mobile -S`
