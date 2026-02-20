# ADR-0002: 事件模型（Event Envelope）与 SQLite Schema

## Status
Accepted

## Decision
- Unified event envelope fields:
  - `protocol_version`, `event_id`, `run_id`, `seq`, `ts`, `type`, `payload`
- Runtime persists:
  - `projects`, `sessions`, `runs`, `events`, `artifacts`, `model_configs`, `audit_logs`, `tool_confirmations`, `system_events`

## Notes
- `events` table keeps append-only ordered records using `global_seq` and `(run_id, seq)` unique.
- `tool_confirmations.status` is tri-state: `pending | approved | denied`.
- Restart recovery rule: pending confirmations are marked denied by system and emit `error` + `done` events.
- Replay API reads persisted events by run.
