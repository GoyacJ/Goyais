# Implementation Scan Map

按域执行：`API route -> handler -> service -> repo -> migration -> integration test`。

## Domain Map

| Domain | API Route Prefix | Handler | Service | Repository | Migration Hint | Test Hint |
|---|---|---|---|---|---|---|
| commands | `/api/v1/commands*` | `/Users/goya/Repo/Git/Goyais/internal/access/http/commands.go` | `/Users/goya/Repo/Git/Goyais/internal/command/service.go` | `/Users/goya/Repo/Git/Goyais/internal/command/` | `migrations/*/*commands*.sql` | `router_integration_test.go`, `postgres_contract_test.go` |
| shares | `/api/v1/shares*` | `/Users/goya/Repo/Git/Goyais/internal/access/http/shares.go` | `/Users/goya/Repo/Git/Goyais/internal/command/service.go` | `/Users/goya/Repo/Git/Goyais/internal/command/` | `migrations/*/*acl*.sql` | `router_integration_test.go` |
| assets | `/api/v1/assets*` | `/Users/goya/Repo/Git/Goyais/internal/access/http/assets.go` | `/Users/goya/Repo/Git/Goyais/internal/asset/service.go` | `/Users/goya/Repo/Git/Goyais/internal/asset/` | `migrations/*/*asset*.sql` | `router_integration_test.go`, `postgres_contract_test.go` |
| workflow | `/api/v1/workflow-*` | `/Users/goya/Repo/Git/Goyais/internal/access/http/workflows.go` | `/Users/goya/Repo/Git/Goyais/internal/workflow/service.go` | `/Users/goya/Repo/Git/Goyais/internal/workflow/` | `migrations/*/*workflow*.sql` | `router_integration_test.go`, `postgres_contract_test.go` |
| registry | `/api/v1/registry/*` | `/Users/goya/Repo/Git/Goyais/internal/access/http/registry.go` | `/Users/goya/Repo/Git/Goyais/internal/registry/service.go` | `/Users/goya/Repo/Git/Goyais/internal/registry/` | `migrations/*/*registry*.sql` | `router_integration_test.go`, `postgres_contract_test.go` |
| plugin-market | `/api/v1/plugin-market/*` | `/Users/goya/Repo/Git/Goyais/internal/access/http/plugins.go` | `/Users/goya/Repo/Git/Goyais/internal/plugin/service.go` | `/Users/goya/Repo/Git/Goyais/internal/plugin/` | `migrations/*/*plugin*.sql` | `router_integration_test.go`, `postgres_contract_test.go` |
| streams | `/api/v1/streams*` | `/Users/goya/Repo/Git/Goyais/internal/access/http/streams.go` | `/Users/goya/Repo/Git/Goyais/internal/stream/service.go` | `/Users/goya/Repo/Git/Goyais/internal/stream/` | `migrations/*/*stream*.sql` | `router_integration_test.go`, `postgres_contract_test.go` |
| algorithms (MVP) | `/api/v1/algorithms/*` | `/Users/goya/Repo/Git/Goyais/internal/access/http/algorithms.go` | `/Users/goya/Repo/Git/Goyais/internal/algorithm/service.go` | `/Users/goya/Repo/Git/Goyais/internal/algorithm/` | `migrations/*/*algorithm*.sql` | `router_integration_test.go` |

> 说明：当前基线下 `algorithms (MVP)` 已实现；仅当 `deps.CommandService` 为空时才应视为 `placeholder`。

## Status Rules

- `implemented`: route 非 placeholder 且 handler/service/repo/migration/test 证据完整。
- `partial`: route 已挂载但证据缺项。
- `placeholder`: route 仍走 `NewNotImplementedHandler(...)`。
- `unknown`: 扫描信号不足，需人工复核。
