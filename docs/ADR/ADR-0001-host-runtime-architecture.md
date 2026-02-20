# ADR-0001: Host/Runtime 分层与通信协议

## Status
Accepted

## Context
MVP-1 requires Tauri desktop host with Python runtime using local HTTP + SSE.

## Decision
- Host: Tauri v2 + React UI
- Runtime: FastAPI service
- SQLite ownership: Runtime is the single writer. Host never writes SQLite directly.
- Protocol endpoints:
  - `POST /v1/runs`
  - `GET /v1/runs/{run_id}/events` (SSE)
  - `POST /v1/tool-confirmations`
  - `GET/POST /v1/projects`, `GET/POST /v1/model-configs` (all writes go through Runtime API)

## Consequences
- Loose coupling between UI and agent runtime
- Runtime can evolve independently
- SSE enables real-time event timeline
