# Next Plan Template (Evidence-Driven)

## Baseline Snapshot

- status: confirmed|partial|unknown
- path: /absolute/path
- command: `...`
- findings:

## Acceptance Progress

- total:
- done:
- todo:
- ratio:
- deferred domains:

## Implementation Scan Matrix

| Domain | Status | Evidence Path | Evidence Command | Notes |
|---|---|---|---|---|

## Contract Drift Findings

- [ ] drift item
  - status:
  - path:
  - command:
  - impact:

## Risk Register

| Risk | Level | Trigger | Impact | Mitigation | Rollback | Evidence |
|---|---|---|---|---|---|---|

## Next Slices (DoD + Tests + Rollback)

### Slice 1

- Goal:
- Scope:
- Affected Files:
- DoD:
- Tests:
- Rollback:

### Slice 2

- Goal:
- Scope:
- Affected Files:
- DoD:
- Tests:
- Rollback:

## Thread/Worktree Execution Plan

- Proposed thread:
- Branch naming: `codex/<thread-id>-<topic>`
- Worktree path:
- Integration order:
- Pre-commit guard commands:
  - `git diff --cached --name-only`
  - `git diff --cached --name-only | rg '^(data/objects/|.*\.db$|build/|web/dist/|web/node_modules/|\.agents/)' && exit 1 || true`

## Evidence Appendix

- Commands executed:
- Key outputs:
- Unknowns and why:
