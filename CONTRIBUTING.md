# Contributing to Goyais

[简体中文](CONTRIBUTING.zh-CN.md) | English

## Prerequisites

1. Read `docs/12-dev-kickoff-package.md`.
2. Read `docs/13-development-plan.md`.
3. Read `docs/15-development-standards.md`.
4. Read `docs/16-open-source-governance.md`.

## Workflow

1. Create a branch: `codex/<task-id>-<topic>`.
2. Move target task to `IN_PROGRESS` in `docs/14-development-progress.md`.
3. Implement and test the change.
4. Update impacted docs and progress status.
5. Open a PR with required template fields.

## Design-First Rule

1. Implementation must strictly follow design docs (`docs/00-17` where applicable).
2. If you find a document error or conflict, fix docs first (or in the same PR) before merging implementation.
3. Do not treat undocumented behavior as final contract.

## Pull Request Requirements

Your PR description must include:

- Task ID / issue link
- Scope and rationale
- Test evidence
- Compatibility impact (API/SSE/events/errors)
- Risks and rollback notes

## Quality Gates

At minimum, PR must pass:

- lint
- unit tests
- integration tests when affected
- build

## Contract Change Rules

If change affects API, events, errors, policy, or domain model:

1. Update `docs/12-dev-kickoff-package.md` if frozen contract changed.
2. Update relevant source docs (`docs/00-11`, `docs/15`, `docs/16`, `docs/17`).
3. Mention compatibility and migration impact in PR.
