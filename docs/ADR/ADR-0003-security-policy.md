# ADR-0003: 安全策略（工具确认、路径限制、命令白名单）

## Status
Accepted

## Decision
- Sensitive tools require user confirmation by default:
  - `write_file`, `apply_patch`, `run_command`, network operations
- Path policy:
  - writes outside workspace are denied
- Command policy:
  - denylist blocks dangerous commands
  - allowlist limits executable commands
- Confirmation state is tri-state: `pending | approved | denied`
- Runtime restart recovery:
  - pending confirmations are resolved to `denied` by system
  - emit `error` event and terminal `done(status=failed)`
- All tool calls, results, approvals, denials, and guard rejections are audit-logged.
