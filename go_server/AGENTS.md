# go_server AGENTS

本文件定义 Go 后端模块的实现级约束，优先级低于根 `AGENTS.md`。

## 1. API 与契约

- API 前缀固定 `/api/v1`。
- 写路径遵循 Command-first。
- 错误模型统一：`error: { code, messageKey, details }`。
- 变更 API/状态机/数据模型必须同步 `go_server/docs/*` 契约文档。

## 2. 配置与 Provider

- ENV 前缀：`GOYAIS_`。
- 优先级：`ENV > YAML > default`。
- provider 支持矩阵按 `go_server/docs/arch/overview.md` 与 `go_server/docs/spec/v0.1.md`。

## 3. Single Binary

- 发布命令：`make -C go_server build`。
- embed 目标：`go_server/internal/access/webstatic/dist`。
- 关键验收：`bash .agents/skills/goyais-release-regression/scripts/verify_single_binary.sh`。

## 4. Quality Gates

- `go test ./...`
- `make -C go_server build`
- `bash go_server/scripts/ci/contract_regression.sh`

## 5. 实现边界

- 优先保证兼容性与可回滚性。
- 禁止在 handler 里堆叠业务规则，复杂逻辑应下沉到 service/domain。
