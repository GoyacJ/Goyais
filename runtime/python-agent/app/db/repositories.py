from __future__ import annotations

import json
import uuid
from typing import Any

import aiosqlite


class Repository:
    def __init__(self, conn: aiosqlite.Connection):
        self.conn = conn

    async def ensure_project(self, project_id: str, workspace_path: str) -> None:
        await self.conn.execute(
            """
            INSERT INTO projects(project_id, name, workspace_path, created_at, updated_at)
            VALUES(?, ?, ?, strftime('%Y-%m-%dT%H:%M:%fZ','now'), strftime('%Y-%m-%dT%H:%M:%fZ','now'))
            ON CONFLICT(project_id) DO UPDATE SET
              workspace_path=excluded.workspace_path,
              updated_at=strftime('%Y-%m-%dT%H:%M:%fZ','now')
            """,
            (project_id, project_id, workspace_path),
        )
        await self.conn.commit()

    async def ensure_session(self, session_id: str, project_id: str) -> None:
        await self.conn.execute(
            """
            INSERT INTO sessions(session_id, project_id, title, created_at, updated_at)
            VALUES(?, ?, ?, strftime('%Y-%m-%dT%H:%M:%fZ','now'), strftime('%Y-%m-%dT%H:%M:%fZ','now'))
            ON CONFLICT(session_id) DO UPDATE SET
              updated_at=strftime('%Y-%m-%dT%H:%M:%fZ','now')
            """,
            (session_id, project_id, session_id),
        )
        await self.conn.commit()

    async def create_run(self, payload: dict[str, Any], run_id: str, trace_id: str) -> None:
        model_config_id = payload.get("model_config_id") or None
        if model_config_id:
            cursor = await self.conn.execute(
                "SELECT 1 FROM model_configs WHERE model_config_id=?",
                (model_config_id,),
            )
            if await cursor.fetchone() is None:
                model_config_id = None

        await self.conn.execute(
            """
            INSERT INTO runs(run_id, project_id, session_id, model_config_id, input, workspace_path, trace_id, status, created_at, started_at)
            VALUES(?, ?, ?, ?, ?, ?, ?, 'running', strftime('%Y-%m-%dT%H:%M:%fZ','now'), strftime('%Y-%m-%dT%H:%M:%fZ','now'))
            """,
            (
                run_id,
                payload["project_id"],
                payload["session_id"],
                model_config_id,
                payload["input"],
                payload["workspace_path"],
                trace_id,
            ),
        )
        await self.conn.commit()

    async def update_run_status(self, run_id: str, status: str) -> None:
        await self.conn.execute(
            """
            UPDATE runs
            SET status=?, completed_at=CASE WHEN ? IN ('completed', 'failed', 'cancelled') THEN strftime('%Y-%m-%dT%H:%M:%fZ','now') ELSE completed_at END
            WHERE run_id=?
            """,
            (status, status, run_id),
        )
        await self.conn.commit()

    async def get_run_status(self, run_id: str) -> str | None:
        cursor = await self.conn.execute("SELECT status FROM runs WHERE run_id=?", (run_id,))
        row = await cursor.fetchone()
        if row is None:
            return None
        return str(row["status"])

    async def get_run_trace_id(self, run_id: str) -> str | None:
        cursor = await self.conn.execute("SELECT trace_id FROM runs WHERE run_id=?", (run_id,))
        row = await cursor.fetchone()
        if row is None:
            return None
        return str(row["trace_id"])

    async def next_seq(self, run_id: str) -> int:
        cursor = await self.conn.execute("SELECT COALESCE(MAX(seq), 0) + 1 AS seq FROM events WHERE run_id=?", (run_id,))
        row = await cursor.fetchone()
        return int(row["seq"])

    async def insert_event(self, event: dict[str, Any]) -> None:
        protocol_version = str(event.get("protocol_version", "2.0.0"))
        await self.conn.execute(
            """
            INSERT INTO events(protocol_version, trace_id, event_id, run_id, seq, ts, type, payload_json, created_at)
            VALUES(?, ?, ?, ?, ?, ?, ?, ?, strftime('%Y-%m-%dT%H:%M:%fZ','now'))
            """,
            (
                protocol_version,
                event["trace_id"],
                event["event_id"],
                event["run_id"],
                event["seq"],
                event["ts"],
                event["type"],
                json.dumps(event["payload"]),
            ),
        )
        await self.conn.commit()

    async def list_events_by_run(self, run_id: str) -> list[dict[str, Any]]:
        cursor = await self.conn.execute(
            "SELECT protocol_version, trace_id, event_id, run_id, seq, ts, type, payload_json FROM events WHERE run_id=? ORDER BY seq ASC",
            (run_id,),
        )
        rows = await cursor.fetchall()
        return [
            {
                "protocol_version": row["protocol_version"],
                "trace_id": row["trace_id"],
                "event_id": row["event_id"],
                "run_id": row["run_id"],
                "seq": row["seq"],
                "ts": row["ts"],
                "type": row["type"],
                "payload": json.loads(row["payload_json"]),
            }
            for row in rows
        ]

    async def list_runs_by_session(self, session_id: str) -> list[dict[str, Any]]:
        cursor = await self.conn.execute(
            "SELECT run_id, trace_id, status, created_at, input FROM runs WHERE session_id=? ORDER BY created_at DESC",
            (session_id,),
        )
        rows = await cursor.fetchall()
        return [dict(row) for row in rows]

    async def get_run(self, run_id: str) -> dict[str, Any] | None:
        cursor = await self.conn.execute(
            """
            SELECT run_id, project_id, session_id, model_config_id, input, workspace_path, trace_id, status,
                   created_at, started_at, completed_at
            FROM runs
            WHERE run_id=?
            """,
            (run_id,),
        )
        row = await cursor.fetchone()
        if row is None:
            return None
        return dict(row)

    async def upsert_model_config(self, data: dict[str, Any]) -> None:
        await self.conn.execute(
            """
            INSERT INTO model_configs(model_config_id, provider, model, base_url, temperature, max_tokens, secret_ref, created_at, updated_at)
            VALUES(?, ?, ?, ?, ?, ?, ?, strftime('%Y-%m-%dT%H:%M:%fZ','now'), strftime('%Y-%m-%dT%H:%M:%fZ','now'))
            ON CONFLICT(model_config_id) DO UPDATE SET
              provider=excluded.provider,
              model=excluded.model,
              base_url=excluded.base_url,
              temperature=excluded.temperature,
              max_tokens=excluded.max_tokens,
              secret_ref=excluded.secret_ref,
              updated_at=strftime('%Y-%m-%dT%H:%M:%fZ','now')
            """,
            (
                data["model_config_id"],
                data["provider"],
                data["model"],
                data.get("base_url"),
                float(data.get("temperature", 0)),
                data.get("max_tokens"),
                data["secret_ref"],
            ),
        )
        await self.conn.commit()

    async def list_model_configs(self) -> list[dict[str, Any]]:
        cursor = await self.conn.execute(
            "SELECT model_config_id, provider, model, base_url, temperature, max_tokens, secret_ref, created_at, updated_at FROM model_configs ORDER BY created_at DESC"
        )
        rows = await cursor.fetchall()
        return [dict(row) for row in rows]

    async def get_model_config(self, model_config_id: str) -> dict[str, Any] | None:
        cursor = await self.conn.execute(
            """
            SELECT model_config_id, provider, model, base_url, temperature, max_tokens, secret_ref
            FROM model_configs
            WHERE model_config_id=?
            """,
            (model_config_id,),
        )
        row = await cursor.fetchone()
        if row is None:
            return None
        return dict(row)

    async def insert_audit(
        self,
        *,
        audit_id: str,
        trace_id: str,
        run_id: str | None,
        event_id: str | None,
        call_id: str | None,
        action: str,
        tool_name: str | None,
        args: dict[str, Any] | None,
        result: Any,
        requires_confirmation: bool,
        user_decision: str,
        outcome: str,
    ) -> None:
        await self.conn.execute(
            """
            INSERT INTO audit_logs(
              audit_id, trace_id, run_id, event_id, call_id, action, tool_name, args_json, result_json,
              requires_confirmation, user_decision, decision_ts, outcome, created_at
            ) VALUES(
              ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?,
              CASE WHEN ? != 'n/a' THEN strftime('%Y-%m-%dT%H:%M:%fZ','now') ELSE NULL END,
              ?, strftime('%Y-%m-%dT%H:%M:%fZ','now')
            )
            """,
            (
                audit_id,
                trace_id,
                run_id,
                event_id,
                call_id,
                action,
                tool_name,
                json.dumps(args or {}),
                json.dumps(result),
                1 if requires_confirmation else 0,
                user_decision,
                user_decision,
                outcome,
            ),
        )
        await self.conn.commit()

    async def upsert_tool_confirmation_status(
        self,
        run_id: str,
        call_id: str,
        status: str,
        *,
        decided_by: str = "user",
    ) -> None:
        if status not in {"pending", "approved", "denied"}:
            raise ValueError(f"Invalid confirmation status: {status}")

        await self.conn.execute(
            """
            INSERT INTO tool_confirmations(run_id, call_id, status, decided_at, decided_by, created_at, updated_at)
            VALUES(
              ?, ?, ?,
              CASE WHEN ? = 'pending' THEN NULL ELSE strftime('%Y-%m-%dT%H:%M:%fZ','now') END,
              ?,
              strftime('%Y-%m-%dT%H:%M:%fZ','now'),
              strftime('%Y-%m-%dT%H:%M:%fZ','now')
            )
            ON CONFLICT(run_id, call_id) DO UPDATE SET
              status=excluded.status,
              decided_at=excluded.decided_at,
              decided_by=excluded.decided_by,
              updated_at=excluded.updated_at
            """,
            (run_id, call_id, status, status, decided_by),
        )
        await self.conn.commit()

    async def get_tool_confirmation_status(self, run_id: str, call_id: str) -> str | None:
        cursor = await self.conn.execute(
            "SELECT status FROM tool_confirmations WHERE run_id=? AND call_id=?",
            (run_id, call_id),
        )
        row = await cursor.fetchone()
        if row is None:
            return None
        return str(row["status"])

    async def list_pending_confirmations(self) -> list[dict[str, Any]]:
        cursor = await self.conn.execute(
            """
            SELECT tc.run_id, tc.call_id
            FROM tool_confirmations tc
            JOIN runs r ON r.run_id = tc.run_id
            WHERE tc.status='pending' AND r.status IN ('running', 'waiting_confirmation')
            ORDER BY r.created_at ASC
            """
        )
        rows = await cursor.fetchall()
        return [dict(row) for row in rows]

    async def list_projects(self) -> list[dict[str, Any]]:
        cursor = await self.conn.execute(
            "SELECT project_id, name, workspace_path, created_at, updated_at FROM projects ORDER BY created_at DESC"
        )
        rows = await cursor.fetchall()
        return [dict(row) for row in rows]

    async def create_project(self, project_id: str, name: str, workspace_path: str) -> None:
        await self.conn.execute(
            """
            INSERT INTO projects(project_id, name, workspace_path, created_at, updated_at)
            VALUES(?, ?, ?, strftime('%Y-%m-%dT%H:%M:%fZ','now'), strftime('%Y-%m-%dT%H:%M:%fZ','now'))
            """,
            (project_id, name, workspace_path),
        )
        await self.conn.commit()

    async def insert_system_event(self, event_type: str, payload: dict[str, Any]) -> str:
        event_id = payload.get("event_id")
        if not isinstance(event_id, str) or not event_id:
            event_id = payload["event_id"] = f"sys_{uuid.uuid4().hex[:12]}"

        await self.conn.execute(
            """
            INSERT INTO system_events(event_id, ts, type, payload_json, created_at)
            VALUES(?, strftime('%Y-%m-%dT%H:%M:%fZ','now'), ?, ?, strftime('%Y-%m-%dT%H:%M:%fZ','now'))
            """,
            (event_id, event_type, json.dumps(payload)),
        )
        await self.conn.commit()
        return event_id

    async def list_system_events(self, since_global_seq: int = 0) -> list[dict[str, Any]]:
        cursor = await self.conn.execute(
            """
            SELECT global_seq, event_id, ts, type, payload_json
            FROM system_events
            WHERE global_seq > ?
            ORDER BY global_seq ASC
            """,
            (since_global_seq,),
        )
        rows = await cursor.fetchall()
        return [
            {
                "global_seq": row["global_seq"],
                "event_id": row["event_id"],
                "ts": row["ts"],
                "type": row["type"],
                "payload": json.loads(row["payload_json"]),
            }
            for row in rows
        ]

    async def insert_or_update_sync_state(self, *, last_pushed_global_seq: int, last_pulled_server_seq: int) -> None:
        await self.conn.execute(
            """
            UPDATE sync_state
            SET last_pushed_global_seq=?, last_pulled_server_seq=?, updated_at=strftime('%Y-%m-%dT%H:%M:%fZ','now')
            WHERE singleton_id=1
            """,
            (last_pushed_global_seq, last_pulled_server_seq),
        )
        await self.conn.commit()

    async def list_audit_logs_by_run(self, run_id: str, limit: int = 200) -> list[dict[str, Any]]:
        cursor = await self.conn.execute(
            """
            SELECT trace_id, audit_id, run_id, event_id, call_id, action, tool_name, args_json, result_json,
                   requires_confirmation, user_decision, decision_ts, outcome, created_at
            FROM audit_logs
            WHERE run_id=?
            ORDER BY created_at DESC
            LIMIT ?
            """,
            (run_id, limit),
        )
        rows = await cursor.fetchall()
        result: list[dict[str, Any]] = []
        for row in rows:
            item = dict(row)
            item["args"] = json.loads(item.pop("args_json") or "{}")
            item["result"] = json.loads(item.pop("result_json") or "{}")
            result.append(item)
        return result

    async def get_sync_state(self) -> dict[str, Any]:
        cursor = await self.conn.execute(
            "SELECT device_id, last_pushed_global_seq, last_pulled_server_seq FROM sync_state WHERE singleton_id=1"
        )
        row = await cursor.fetchone()
        return dict(row)

    async def list_unsynced_events(self, since_global_seq: int) -> list[dict[str, Any]]:
        cursor = await self.conn.execute(
            """
            SELECT global_seq, protocol_version, trace_id, event_id, run_id, seq, ts, type, payload_json
            FROM events
            WHERE global_seq > ?
            ORDER BY global_seq ASC
            """,
            (since_global_seq,),
        )
        rows = await cursor.fetchall()
        return [
            {
                "global_seq": row["global_seq"],
                "protocol_version": row["protocol_version"],
                "trace_id": row["trace_id"],
                "event_id": row["event_id"],
                "run_id": row["run_id"],
                "seq": row["seq"],
                "ts": row["ts"],
                "type": row["type"],
                "payload": json.loads(row["payload_json"]),
            }
            for row in rows
        ]

    async def upsert_synced_event(self, event_id: str, server_seq: int) -> None:
        await self.conn.execute(
            """
            INSERT INTO synced_event_map(event_id, server_seq, synced_at)
            VALUES(?, ?, strftime('%Y-%m-%dT%H:%M:%fZ','now'))
            ON CONFLICT(event_id) DO UPDATE SET
              server_seq=excluded.server_seq,
              synced_at=excluded.synced_at
            """,
            (event_id, server_seq),
        )
        await self.conn.commit()

    async def insert_event_if_missing(self, event: dict[str, Any]) -> bool:
        cursor = await self.conn.execute("SELECT 1 FROM events WHERE event_id=?", (event["event_id"],))
        row = await cursor.fetchone()
        if row:
            return False

        await self.insert_event(
            {
                "protocol_version": event.get("protocol_version", "2.0.0"),
                "trace_id": event["trace_id"],
                "event_id": event["event_id"],
                "run_id": event["run_id"],
                "seq": event["seq"],
                "ts": event["ts"],
                "type": event["type"],
                "payload": event["payload"],
            }
        )
        return True
