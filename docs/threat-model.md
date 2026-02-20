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
- Desktop bypassing Hub and calling remote runtime directly
- Runtime/workspace misbinding (runtime serves wrong workspace)
- Secret plaintext exposure during remote provider calls
- Confirmation race in multi-user workspace

## Mitigations

- Sensitive tools require explicit approval
- Path guard rejects writes outside workspace
- Command guard allowlist + denylist
- Audit logs for allow/deny with parameters and outcomes
- Sync server bearer token authentication
- Unified error model with trace_id for incident correlation
- Diagnostics export endpoint requires runtime token
- Diagnostics payload is recursively redacted (`Authorization`, `token`, `apiKey`, `secret_ref`, path values)
- Remote runtime enforces `X-Hub-Auth` + `X-User-Id` + `X-Trace-Id` when `GOYAIS_RUNTIME_REQUIRE_HUB_AUTH=true`
- Hub gateway enforces workspace membership + RBAC and probes runtime workspace binding before forwarding
- Hub rejects misbound runtime with `E_RUNTIME_MISCONFIGURED`
- `secret:*` values are AES-GCM encrypted at rest in Hub and resolved only via internal endpoint for runtime use
- Confirmation write path is atomic (`pending -> approved|denied` once) and conflicting decisions return `E_CONFIRMATION_ALREADY_DECIDED`

## Residual risk (P1)

- Hub-managed runtime process lifecycle (spawn/heartbeat self-heal)
- Stronger internal trust channel (mTLS / workload identity)
- Secret transport hardening end-to-end (KMS/HSM + key rotation)
- Artifact large binary sync
- Overly broad diagnostics access if runtime token is leaked
