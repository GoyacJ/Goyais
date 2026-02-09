# Goyais

[简体中文](README.zh-CN.md) | English

Goyais is an intent-driven, AI-native orchestration platform for multimodal assets and governed execution.

## Project Status

`Pre-implementation (design-complete, code-in-progress)`

The repository currently uses a design-first workflow. Core contracts and implementation plans are defined under `docs/`.

## Why Goyais

- Intent-driven operation: users can trigger platform actions through text/voice conversation.
- Unified execution model: intent -> plan -> approval/policy -> workflow/agent run.
- Governed AI operations: RBAC, policy checks, approval gates, and audit trails.
- Multimodal asset lifecycle: upload/import/process/derive/reuse with lineage.
- Enterprise observability: trace/run events, replay, streaming updates, and diagnostics.
- Internationalization-ready platform: locale-aware API, UI, and error/message delivery.

## Core Capabilities

- Asset system (file/stream/structured/text) with immutable lineage.
- Tool/model/algorithm registry and resolution.
- Workflow engine (DAG + CAS context patching).
- Agent runtime (plan-act-observe-recover loop).
- Intent orchestrator for full-platform actions (not only AI inference tasks).
- Policy and approval engine for high-risk operations.

## Documentation Map

Design documents are currently maintained in Chinese under `docs/`.

- Overview: `docs/00-overview.md`
- Architecture: `docs/01-architecture.md`
- Domain Model: `docs/02-domain-model.md`
- API Design: `docs/10-api-design.md`
- Frontend Design: `docs/11-frontend-design.md`
- Dev Kickoff Package: `docs/12-dev-kickoff-package.md`
- Development Plan: `docs/13-development-plan.md`
- Development Progress: `docs/14-development-progress.md`
- Development Standards: `docs/15-development-standards.md`
- Open-source Governance: `docs/16-open-source-governance.md`
- I18n & Localization Design: `docs/17-internationalization-design.md`

## Development Workflow

Before starting any task:

1. Read `docs/12-dev-kickoff-package.md`.
2. Read `docs/13-development-plan.md`.
3. Read `docs/15-development-standards.md`.
4. Read `docs/16-open-source-governance.md`.
5. Mark task as `IN_PROGRESS` in `docs/14-development-progress.md`.

Rules:

- Follow design docs strictly.
- If docs are wrong or conflicting, fix docs first (or in the same PR) before implementation is accepted.

## Internationalization (Product)

Goyais targets product-level i18n support (not documentation-only):

- Locale negotiation via request headers and user preference.
- Locale-aware API messages and error text.
- Frontend language switching and translation key management.
- Policy/approval/notification content rendering by locale.

See `docs/17-internationalization-design.md` for details.

## Open Source

- License: Apache-2.0 (`LICENSE`)
- Contributing: `CONTRIBUTING.md` / `CONTRIBUTING.zh-CN.md`
- Security: `SECURITY.md` / `SECURITY.zh-CN.md`
- Governance: `GOVERNANCE.md` / `GOVERNANCE.zh-CN.md`
- Code of Conduct: `CODE_OF_CONDUCT.md` / `CODE_OF_CONDUCT.zh-CN.md`
- Maintainers: `MAINTAINERS.md` / `MAINTAINERS.zh-CN.md`

## Roadmap Snapshot

- S0: API/SSE skeleton, baseline middleware, event store foundation.
- S1: Intent MVP + RBAC + approval core.
- S2: Asset-workflow closed loop.
- S3: Full AI interaction (text + voice) UX hardening.
- S4: Stability, performance, security audit, and release readiness.

## License

Licensed under Apache License 2.0. See `LICENSE`.
