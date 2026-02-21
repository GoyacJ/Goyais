# Contributing to Goyais

Thanks for your interest in contributing to Goyais.

## Ways to Contribute

- Report bugs
- Improve documentation
- Propose or implement features
- Refactor for readability and maintainability
- Add or improve tests

## Development Setup

1. Fork and clone the repository.
2. Install dependencies from the repository root:

```bash
pnpm install
pnpm protocol:generate
```

3. Run the core checks before opening a pull request:

```bash
pnpm version:check
pnpm protocol:generate
pnpm typecheck
pnpm test
cd server/hub-server-go && go test ./...
```

## Pull Request Expectations

Please include the following in every PR:

- **Problem statement**: what issue is being solved
- **Approach**: what changed and why
- **Verification evidence**: test/check commands and outcomes
- **Documentation updates**: if behavior, architecture, or usage changed

If your change affects APIs, protocol, execution flow, or security behavior, include a clear migration or compatibility note.

## Branch and Commit Guidelines

- Use a dedicated feature/fix branch from `master`.
- Keep commits focused and reviewable.
- Use clear commit messages that describe intent.
- Avoid unrelated changes in the same PR.

## Code Style

- Follow existing project conventions in each module.
- Keep changes minimal and scoped.
- Add tests for behavior changes and bug fixes whenever feasible.

## Communication

- Use GitHub Issues for bug reports and feature requests.
- For security issues, do **not** open public issues. Use private reporting as described in `SECURITY.md`.

Thanks for helping improve Goyais.
