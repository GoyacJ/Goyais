# Threat Model (MVP-1)

## Assets

- Workspace source files
- API credentials (secret references)
- Tool execution trace and audit logs
- Run/event history

## Main risks

- Prompt injection causing destructive tool calls
- Path traversal writes outside workspace
- Command execution abuse
- Unauthorized sync data access

## Mitigations

- Sensitive tools require explicit approval
- Path guard rejects writes outside workspace
- Command guard allowlist + denylist
- Audit logs for allow/deny with parameters and outcomes
- Sync server bearer token authentication
- Unified error model with trace_id for incident correlation
- Diagnostics export endpoint requires runtime token
- Diagnostics payload is recursively redacted (`Authorization`, `token`, `apiKey`, `secret_ref`, path values)

## Residual risk (P1)

- Multi-user auth/z
- Secret transport hardening end-to-end
- Artifact large binary sync
- Overly broad diagnostics access if runtime token is leaked
