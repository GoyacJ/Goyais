# Frontend Refactor Scope (2026-02-25)

This phase enforces a strict debt-cleanup strategy:

- remove runtime fallback/mock branches from desktop frontend
- migrate state to Pinia
- migrate UI styling to UnoCSS with token alignment
- remove Hub legacy compatibility paths while keeping `/v1`
- keep verification evidence for each task and checkpoint

## Canonical Plan Documents

- [Master plan](https://github.com/GoyacJ/Goyais/blob/main/docs/refactor/2026-02-25-frontend-refactor-master-plan.md)
- [Task plan](https://github.com/GoyacJ/Goyais/blob/main/docs/refactor/2026-02-25-frontend-refactor-task-plan.md)
- [Spec baseline](https://github.com/GoyacJ/Goyais/blob/main/docs/refactor/2026-02-25-frontend-spec-plan.md)

## Current Acceptance Gates

- desktop lint/test/strict/coverage/token/quality gates
- hub go test suite
- worker pytest + lint
- smoke E2E
- docs and slides build
