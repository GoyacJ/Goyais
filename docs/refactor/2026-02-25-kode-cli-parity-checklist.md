# T-001 Contract/Parity Checklist (Kode-cli Baseline)

- Date: 2026-02-25
- Task ID: T-001
- Scope: Freeze the executable wrapper-level behavior contract of original `Kode-cli` as golden assets (stdout/stderr/exit code) for later Hub+Go runtime parity checks.
- Golden artifact: `docs/refactor/parity/kode-cli-wrapper-golden.json`

## Baseline Evidence Sources

- Wrapper contract source: `Kode-cli/cli.js`, `Kode-cli/cli-acp.js`
- CLI option and output expectation evidence: `Kode-cli/tests/e2e/cli-smoke.test.ts`
- Version source: `Kode-cli/package.json`
- Runtime fallback wording evidence: `Kode-cli/README.md` (native binary + Node fallback notes)

## Captured Contract Matrix (Wrapper Bootstrap Layer)

| Case ID | Command | Contract Surface | Expected Channel |
| --- | --- | --- | --- |
| `cli_help_lite` | `node Kode-cli/cli.js --help-lite` | Usage/help-lite text and success exit | stdout + exit code 0 |
| `cli_version` | `node Kode-cli/cli.js --version` | Version text and success exit | stdout + exit code 0 |
| `cli_help_without_dist` | `node Kode-cli/cli.js --help` | Fallback error when runtime payload missing | stderr + exit code 1 |
| `acp_without_dist` | `node Kode-cli/cli-acp.js` | ACP fallback error when runtime payload missing | stderr + exit code 1 |

## Runbook

1. Capture/refresh golden asset:

```bash
node scripts/refactor/t001-capture-kode-cli-baseline.mjs
```

2. Validate current outputs against frozen golden:

```bash
node scripts/refactor/t001-capture-kode-cli-baseline.mjs --check
```

3. Validate script behavior:

```bash
node --test scripts/refactor/t001-kode-cli-baseline.test.mjs
```

## Pending Unknowns (Carried to Next Tasks)

- Full runtime behavior (`--print`, interaction loop, plan mode transitions, command subtree help matrix) is not executable in current baseline run because `Kode-cli/dist` is unavailable in this workspace snapshot.
- Probe expansion is required in T-002/T-003 to include runtime-level transcripts after reproducible dist bootstrap is established.
