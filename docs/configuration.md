# Configuration Reference

This document defines runtime configuration for the v0.2.0 Hub-First stack.

## Layers

1. Local desktop-managed process config (editable in Settings, local profile only)
2. Service deployment config (`hub-server-go`, `python-agent`)
3. Remote connection config (desktop client endpoints only)

## Local Desktop Config Center

Desktop stores a `LocalProcessConfigV1` model and can persist it through Tauri commands:

- `local_config_read`
- `local_config_write`
- `service_start`
- `service_status`
- `service_stop`

Sensitive values are expected to be stored in system keychain and injected at process start:

- `GOYAIS_RUNTIME_SHARED_SECRET`
- `GOYAIS_HUB_INTERNAL_SECRET`
- `GOYAIS_SYNC_TOKEN`
- `GOYAIS_RUNTIME_SECRET_TOKEN`

## Hub (`server/hub-server-go`)

### Port and log level aliases

- `GOYAIS_HUB_PORT` overrides `PORT`
- `GOYAIS_HUB_LOG_LEVEL` overrides `LOG_LEVEL`

### Main variables

- `GOYAIS_DB_DRIVER` (`sqlite` or `postgres`)
- `GOYAIS_DB_PATH`
- `GOYAIS_DATABASE_URL`
- `GOYAIS_AUTH_MODE` (`local_open` or `remote_auth`)
- `GOYAIS_WORKER_BASE_URL`
- `GOYAIS_RUNTIME_SHARED_SECRET`
- `GOYAIS_MAX_CONCURRENT_EXECUTIONS`

See `/Users/goya/Repo/Git/Goyais/server/hub-server-go/.env.example` for a complete template.

## Runtime (`runtime/python-agent`)

Main variables:

- `GOYAIS_RUNTIME_HOST`
- `GOYAIS_RUNTIME_PORT`
- `GOYAIS_RUNTIME_REQUIRE_HUB_AUTH`
- `GOYAIS_RUNTIME_SHARED_SECRET`
- `GOYAIS_RUNTIME_WORKSPACE_ID`
- `GOYAIS_RUNTIME_WORKSPACE_ROOT`
- `GOYAIS_HUB_BASE_URL`
- `GOYAIS_AGENT_MODE`

The development command now respects `GOYAIS_RUNTIME_HOST` and `GOYAIS_RUNTIME_PORT`.

See `/Users/goya/Repo/Git/Goyais/runtime/python-agent/.env.example`.

## Remote Profile Behavior

In remote profile mode, Settings can edit only client-side connection defaults (for example server URL prefill).  
Remote Hub/Runtime deployment environment variables must be managed in deployment infrastructure, not by desktop.
