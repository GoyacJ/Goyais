# Rule 07: Quality Gates

## Trigger Conditions

- 准备交付、合并、发布前。

## Hard Constraints (MUST)

- 回归门禁顺序：worktree audit -> merged-thread cleanup audit（warn-only） -> precommit guard -> source header check -> java javadoc check -> go test -> web typecheck/test -> build -> single binary verify。
- 验收需有可复现命令证据。

## Counterexamples

- 仅手工验证，不执行脚本回归。
- 仅跑单模块测试即宣称可发布。

## Validation Commands

- `bash go_server/scripts/ci/contract_regression.sh`
- `bash java_server/scripts/ci/java_javadoc_check.sh`
