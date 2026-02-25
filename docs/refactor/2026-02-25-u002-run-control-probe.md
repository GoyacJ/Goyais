# Probe Report: U-002 Run Control Semantics

- Date: 2026-02-25
- Scope: Resolve `approve/deny/resume` state/control semantics before `T-003` integration.

### Probe: U-002

- Goal: Determine authoritative behavior for legacy confirmation decisions and control queue delivery, then map to new `Run` state machine actions.
- Original project path: `services/hub/internal/httpapi` (legacy execution domain behavior preserved in current baseline).
- Command(s) to run:
  - `cd services/hub && go test -count=1 ./internal/httpapi -run TestProbe`
  - `cd services/hub && go test -count=1 ./internal/httpapi -run TestHydrateExecutionDomainFromStoreKeepsLegacyExecutionStateAndCommands`
  - `cd services/hub && go test -count=1 ./internal/httpapi -run TestExecutionConfirmEndpointRemoved`
- Input fixture:
  - `normalizeLegacyExecutionEventType("confirmation_resolved", {"decision":"approve|deny"})`
  - Control queue containing `confirm`, `resume`, `stop` command types.
  - Legacy snapshot with execution state `confirming` and control command `confirm`.
- Expected observable points:
  - stdout: test output with probe assertions.
  - stderr: empty.
  - exit code: `0` for passing assertions.
  - side effect: none (test-only).
- Actual result:
  - `decision=deny` maps to `execution_stopped`.
  - `decision=approve` maps to `execution_started`.
  - Control polling returns only `stop` commands even if queue contains `confirm/resume`.
  - `/v1/executions/{execution_id}/confirm` route is removed (`404` contract).
- Conclusion:
  - `deny` should align with stop/cancel semantics in run state machine.
  - `approve` should align with continue/start (`running`) semantics.
  - `resume` has no direct legacy control endpoint behavior; inferred mapping is "resume execution to running" for queued/waiting states.
- Follow-up action:
  - Implemented in `agentcore/state`:
    - `deny -> cancelled`
    - `approve/resume -> running` (queued/waiting_approval)
  - Added probe tests in `internal/httpapi/control_semantics_probe_test.go`.
