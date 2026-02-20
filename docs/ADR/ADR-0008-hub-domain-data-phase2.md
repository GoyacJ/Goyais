# ADR-0008: Hub Domain Data Phase 2（Remote Projects + Model Configs + RBAC）

## Status
Accepted

## Context
Phase 1 已完成 Remote Workspace 控制面（bootstrap/auth/workspaces/navigation）。Phase 2 需要让 Desktop 在 remote workspace 下读写远端 Projects 与 Model Configs，并保持：

- 不改 `runtime/python-agent` 协议与行为
- local workspace 继续使用 runtime 本地 API
- remote workspace 的业务数据以 hub-server 为权威
- 权限必须服务端强制，前端仅做 UI 级 gating

## Decision
- 在 `hub-server` 新增 domain 表：
  - `projects`
  - `model_configs`
  - `secrets`
- 所有 domain endpoint 强制 `workspace_id` query，并执行两层校验：
  - active membership（workspace scope）
  - permission（RBAC）
- 权限点：
  - Projects：`project:read` / `project:write`
  - Model Configs：`modelconfig:read` / `modelconfig:manage`
- Model Config secret 策略：
  - create/update 接受 `api_key`
  - `api_key` 不可读回，GET 仅返回 `secret_ref`
  - update 若携带新 `api_key` 则轮换 `secret_ref`
- 加密策略（P0）：
  - 使用 `GOYAIS_HUB_SECRET_KEY`（base64 编码 32-byte key）
  - `secrets.value_encrypted` 存 AES-256-GCM 密文（`enc:v1:...`）
  - 缺失/非法密钥时拒绝写入 secret（安全默认）

## Consequences
- Desktop 切到 remote workspace 后，Projects/Model Configs 走 hub 数据源；切回 local 仍走 runtime。
- 写权限不足时，前端按钮隐藏/禁用；即使绕过 UI，服务端仍返回 403。
- `GOYAIS_HUB_SECRET_KEY` 成为 model-config 写路径的必要配置项，部署需要额外管理密钥。

## Caveats and Follow-ups
- Phase 2 使用应用层密钥加密（非 KMS/HSM）；后续可替换为托管密钥系统，保持 `secretCrypto` 接口不变。
- 尚未引入审计日志与 secret rotation policy 细粒度策略（计划在后续阶段完善）。
