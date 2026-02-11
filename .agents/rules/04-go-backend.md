# Rule 04: Go Backend Constraints

## Trigger Conditions

- `go_server` 下任意实现变更。

## Hard Constraints (MUST)

- API 前缀固定 `/api/v1`。
- 错误模型：`error: { code, messageKey, details }`。
- 配置优先级：`ENV > YAML > default`。
- provider 选择不越过 v0.1 支持矩阵。

## Counterexamples

- 引入非 `GOYAIS_` 前缀的运行时配置键。
- 返回非标准错误结构。

## Validation Commands

- `go test ./go_server/...`
- `rg -n 'messageKey|GOYAIS_|/api/v1' go_server -S`
