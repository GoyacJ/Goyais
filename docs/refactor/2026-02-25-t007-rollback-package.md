# T-007 Rollback Package (Worker + Kode-cli Cutover)

- Date: 2026-02-25
- Task ID: T-007
- Scope: provide a one-command rollback path to restore pre-cutover assets (`services/worker` and related release/dev paths) and verify critical runtime contracts.

## One-Command Rollback

```bash
scripts/release/rollback/rollback-to-stable.sh <stable-tag-or-sha>
```

Notes:

- If no ref is passed, default is `v0.4.0` (or `GOYAIS_ROLLBACK_REF`).
- The restore script requires a clean working tree.

## Package Contents

- `scripts/release/rollback/restore-pre-t007.sh`
  - Restores pre-T007 files/directories from a stable ref via `git checkout <ref> -- <paths...>`.
- `scripts/release/rollback/verify-rollback.sh`
  - Verifies:
    - Hub `/health`, auth, conversation create + message submit path (via integration tests)
    - Worker health/internal-token tests when `uv` is available
- `scripts/release/rollback/rollback-to-stable.sh`
  - Wrapper that runs restore then verify.

## Validation Targets

- Hub checks:
  - `TestHealth`
  - `TestLoginValidationAndAuthErrors`
  - `TestProjectConversationFlowWithCursorPagination`
- Worker checks (conditional on `uv`):
  - `tests/test_health.py`
  - `tests/test_internal_tokens.py`

## Drill Evidence

- Drill execution record: `docs/refactor/2026-02-25-t007-rollback-drill.md`

## Failure Handling

- Unknown ref: script exits non-zero with guidance to pass explicit tag/SHA.
- Dirty worktree: script exits non-zero and requires commit/stash first.
- Missing `uv`: worker checks are skipped but Hub checks still enforced.
