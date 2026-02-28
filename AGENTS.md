# AGENTS.md

## Purpose and Non-Goals

This file defines AI-first engineering standards for this repository.
It exists to keep architecture, design quality, and code quality high during rapid AI-assisted development.

This file does not define business requirements, product scope, roadmap priority, or feature policy.

## AI Development Contract

All AI contributors MUST follow these rules:

1. Reuse first. Search existing implementations before adding new ones.
2. Prefer design-improving changes over narrow patches when the current structure is clearly weak.
3. Keep changes explicit and reviewable, with clear rationale and verification evidence.
4. Avoid hidden side effects and unrelated edits.
5. Produce a completion report using the template in this file.

## Architecture Invariants (Must Not Break)

1. Control authority stays hub-centric: Desktop/Mobile clients must not bypass Hub authority for execution control.
2. Execution semantics stay consistent: one active execution per conversation, FIFO behavior within a conversation.
3. Workspace boundaries remain enforced for authorization and data isolation.
4. Shared contracts remain aligned when changed: Hub APIs, OpenAPI spec, and shared TypeScript API models must stay consistent.

## Change Surface Lock (Existing Files Only)

To keep AI edits controllable, change surface lock is mandatory.

1. Before implementation, AI MUST declare `Locked existing files` for this task.
2. AI MAY only edit existing files in that lock list.
3. New files may be added without pre-listing.
4. If a non-locked existing file becomes necessary, AI MUST update the lock list and explain why before editing it.
5. Deleting or renaming an existing file counts as modifying an existing file and MUST be pre-locked.
6. Final output MUST include `Locked existing files`.
7. Final output MUST include `Actually modified existing files`.
8. Final output MUST include `Lock delta` with explicit statement whether scope was exceeded.

## Frontend Engineering Standards

1. Reuse existing components, stores, tokens, and interaction patterns before creating new ones.
2. If new UI primitives are introduced, AI MUST explain why existing components are not suitable.
3. Use Vue 3 Composition API with `<script setup lang="ts">` for new or refactored Vue code unless a file is intentionally legacy.
4. Keep view logic, state logic, and API integration concerns separated.
5. Prefer existing design tokens and shared styles over one-off CSS values.
6. Add or update tests when behavior changes.
7. Avoid copy-paste variants of components; extract shared logic instead.

## Backend Engineering Standards

1. Reuse existing helpers, validators, adapters, and error handling patterns before adding new utilities.
2. Keep HTTP handlers thin; push business/domain behavior into reusable internal modules.
3. Keep permission checks and safety checks centralized and consistent.
4. Preserve stable error semantics and response models when refactoring.
5. Prefer table-driven tests for behavior-heavy logic and boundary cases.
6. Avoid duplicated parsing, validation, storage, or transport logic across packages.

## Refactor Policy (Development Phase)

This project is in active development. Benefit-driven breaking refactors are allowed and encouraged when they improve the system.

AI SHOULD trigger refactor-first behavior when it detects:

1. Repeated logic that should be shared.
2. Layer violations or tight coupling.
3. Poor naming, unclear module boundaries, or mixed responsibilities.
4. Temporary patches that would increase technical debt.

Any breaking or structure-changing refactor MUST provide:

1. Design debt being removed.
2. Why the new structure is better.
3. Affected files and compatibility impact.
4. Verification evidence proving no regression in required behavior.

## Anti-Laziness Rules for AI

AI MUST NOT:

1. Default to minimal edits when those edits worsen architecture.
2. Duplicate existing logic instead of extracting/reusing.
3. Leave placeholder TODO implementations as final output.
4. Skip tests or verification for behavior changes.
5. Introduce silent coupling across modules without documenting rationale.

AI MUST:

1. Prefer coherent design over patch accumulation.
2. Remove dead paths when a refactor supersedes them.
3. Keep naming precise and consistent with module responsibility.
4. Make tradeoffs explicit in the completion report.

## Verification Policy (Layered Gates)

Default policy is minimum-sufficient verification for the change scope.
Escalate verification for higher-risk or cross-stack changes.

1. Desktop changes: run `pnpm lint` and `pnpm test`.
2. Desktop high-risk changes (execution flow/state orchestration/runtime-critical UI): also run `pnpm test:strict` and `pnpm e2e:smoke`.
3. Mobile changes: run `pnpm lint:mobile` and `pnpm test:mobile`.
4. Mobile high-risk changes: also run `pnpm build:mobile` and `pnpm --filter @goyais/mobile e2e:smoke`.
5. Hub changes: run `cd services/hub && go test ./... && go vet ./...`.
6. Docs-only changes: run `pnpm docs:build`; if slides changed, also run `pnpm slides:build`.
7. Cross-stack or release-sensitive changes: also run `make health`.

Every delivery must list exactly which commands were run and their outcomes.

## Change Report Template (Mandatory)

Use this template in every final delivery:

```text
Summary:
- <what changed>

Reuse scan:
- <existing candidates reviewed>
- <what was reused and why>

Design/architecture decisions:
- <key decisions and rationale>

Refactor justification:
- <required when refactor happened; otherwise "None">

Locked existing files:
- <path>

Actually modified existing files:
- <path>

Lock delta:
- <"None (scope respected)" OR explicit differences with reason>

Added files:
- <path>

Verification evidence:
- <command> -> <pass/fail + key result>

Follow-up debt:
- <item or "None">
```

## Instruction Precedence in This Repo

Instruction discovery follows the implemented projectdocs behavior:

1. Walk from git root to current working directory.
2. For each directory, load at most one instruction file with precedence:
`AGENTS.override.md` -> `AGENTS.md` -> `CLAUDE.md`.
3. Concatenate selected files in root-to-leaf order.
4. Total instruction size is bounded by `GOYAIS_PROJECT_DOC_MAX_BYTES` (default 32768 bytes), with truncation when exceeded.

## Maintenance Rules for This File

1. Keep rules specific, technical, and executable.
2. Keep this file concise enough to avoid truncation in instruction loading.
3. Update command examples whenever scripts or toolchains change.
4. Prefer `MUST`/`SHOULD` language for enforceable behavior.
5. Do not duplicate product/business policy here.
