# ADR-0004: 单人同步语义与冲突策略

## Status
Accepted

## Decision
- Sync server is self-hosted and single-user token-authenticated.
- Sync object scope in MVP-1:
  - events
  - artifact metadata (text/patch only)
- Conflict strategy:
  - server-authoritative append-only sequence
  - idempotency via `event_id` uniqueness

## API
- `POST /v1/sync/push` request uses `since_global_seq`
- `GET /v1/sync/pull?since_server_seq=...`
- `server_seq` is assigned by sync server and is the only pull cursor
