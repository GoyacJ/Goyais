# Overview

Goyais is split into three runtime surfaces:

- Desktop: Vue + Tauri app for local and remote workspace orchestration.
- Hub: Go HTTP API for workspace, resource, execution, and admin capabilities.
- Worker: Python runtime for execution, policy, and safety-aware command orchestration.

## Build and Quality Commands

- `pnpm lint`
- `pnpm test`
- `pnpm test:strict`
- `pnpm coverage:gate`
- `pnpm e2e:smoke`
- `pnpm docs:build`
- `pnpm slides:build`

## Where to Look

- Refactor plans: [docs/refactor](https://github.com/GoyacJ/Goyais/tree/main/docs/refactor)
- Release checklist: [docs/release-checklist.md](https://github.com/GoyacJ/Goyais/blob/main/docs/release-checklist.md)
- Review artifacts: [docs/reviews](https://github.com/GoyacJ/Goyais/tree/main/docs/reviews)
