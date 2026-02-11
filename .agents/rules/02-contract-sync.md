# Rule 02: Contract Sync

## Trigger Conditions

- API、数据模型、状态机、配置键、静态路由缓存策略发生变化时。

## Hard Constraints (MUST)

- 同一变更必须同步更新契约文档：
  - `go_server/docs/api/openapi.yaml`
  - `go_server/docs/arch/overview.md`
  - `go_server/docs/arch/data-model.md`
  - `go_server/docs/arch/state-machines.md`
  - `go_server/docs/acceptance.md`

## Counterexamples

- 代码先改，文档后补。
- 仅更新 OpenAPI，不更新状态机/验收。

## Validation Commands

- `bash go_server/scripts/ci/path_migration_audit.sh`
- `rg -n 'openapi|data-model|state-machines|acceptance' go_server/docs -S`
