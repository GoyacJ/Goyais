# T-007 Rollback Drill Record

- Date: 2026-02-25
- Scope: verify rollback package can restore pre-T007 assets and pass key checks.

## Drill Method

To avoid mutating the active dirty workspace, the drill ran in a temporary clone:

1. Clone repository to temp path.
2. Copy rollback scripts from current workspace into clone (`scripts/release/rollback/*.sh`).
3. Simulate post-T007 state in clone by removing:
   - `services/worker/`
   - `scripts/release/build-worker-sidecar.sh`
4. Commit simulated state.
5. Execute rollback:

```bash
./scripts/release/rollback/rollback-to-stable.sh HEAD~1
```

## Observed Result

- `restore-pre-t007.sh` restored pre-cutover assets from `HEAD~1`.
- `verify-rollback.sh` passed Hub contract checks:
  - `TestHealth`
  - `TestLoginValidationAndAuthErrors`
  - `TestProjectConversationFlowWithCursorPagination`
- `verify-rollback.sh` also passed worker checks (`uv` present):
  - `tests/test_health.py`
  - `tests/test_internal_tokens.py`
- Post-rollback confirmations:
  - `services/worker` restored.
  - `scripts/release/build-worker-sidecar.sh` restored.

## Drill Log Excerpt

```text
[rollback] restoring pre-T007 assets from HEAD~1
[rollback] restore complete
[rollback-verify] hub contract checks: health/auth/conversation/message
ok   goyais/services/hub/internal/httpapi  1.672s
[rollback-verify] worker health/internal token checks
6 passed in 0.89s
[rollback-verify] rollback verification passed
[rollback] done: pre-T007 stack restored and verified from HEAD~1
```
