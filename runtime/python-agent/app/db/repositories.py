from __future__ import annotations

import json
import uuid
from typing import Any
import sqlite3

import aiosqlite
from app.protocol_version import load_protocol_version

PROTOCOL_VERSION = load_protocol_version()


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

    async def create_execution(self, payload: dict[str, Any], execution_id: str, trace_id: str) -> None:
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
            INSERT INTO executions(
              execution_id,
              project_id,
              session_id,
              model_config_id,
              input,
              workspace_path,
              trace_id,
              created_by,
              status,
              created_at,
              started_at
            )
            VALUES(
              ?, ?, ?, ?, ?, ?, ?, ?, 'executing',
              strftime('%Y-%m-%dT%H:%M:%fZ','now'),
              strftime('%Y-%m-%dT%H:%M:%fZ','now')
            )
            """,
            (
                execution_id,
                payload["project_id"],
                payload["session_id"],
                model_config_id,
                payload["input"],
                payload["workspace_path"],
                trace_id,
                payload.get("user_id", "user"),
            ),
        )
        await self.conn.commit()

    async def update_execution_status(self, execution_id: str, status: str) -> None:
        await self.conn.execute(
            """
            UPDATE executions
            SET status=?, completed_at=CASE WHEN ? IN ('completed', 'failed', 'cancelled') THEN strftime('%Y-%m-%dT%H:%M:%fZ','now') ELSE completed_at END
            WHERE execution_id=?
            """,
            (status, status, execution_id),
        )
        await self.conn.commit()

    async def get_execution_status(self, execution_id: str) -> str | None:
        cursor = await self.conn.execute("SELECT status FROM executions WHERE execution_id=?", (execution_id,))
        row = await cursor.fetchone()
        if row is None:
            return None
        return str(row["status"])

    async def get_execution_trace_id(self, execution_id: str) -> str | None:
        cursor = await self.conn.execute("SELECT trace_id FROM executions WHERE execution_id=?", (execution_id,))
        row = await cursor.fetchone()
        if row is None:
            return None
        return str(row["trace_id"])

    async def next_execution_seq(self, execution_id: str) -> int:
        cursor = await self.conn.execute(
            "SELECT COALESCE(MAX(seq), 0) + 1 AS seq FROM execution_events WHERE execution_id=?",
            (execution_id,),
        )
        row = await cursor.fetchone()
        return int(row["seq"])

    async def insert_event(self, event: dict[str, Any]) -> None:
        protocol_version = str(event.get("protocol_version", PROTOCOL_VERSION))
        await self.conn.execute(
            """
            INSERT INTO execution_events(
              protocol_version,
              trace_id,
              event_id,
              execution_id,
              seq,
              ts,
              type,
              payload_json,
              created_at
            )
            VALUES(?, ?, ?, ?, ?, ?, ?, ?, strftime('%Y-%m-%dT%H:%M:%fZ','now'))
            """,
            (
                protocol_version,
                event["trace_id"],
                event["event_id"],
                event["execution_id"],
                event["seq"],
                event["ts"],
                event["type"],
                json.dumps(event["payload"]),
            ),
        )
        await self.conn.commit()

    async def list_events_by_execution(self, execution_id: str) -> list[dict[str, Any]]:
        cursor = await self.conn.execute(
            "SELECT protocol_version, trace_id, event_id, execution_id, seq, ts, type, payload_json FROM execution_events WHERE execution_id=? ORDER BY seq ASC",
            (execution_id,),
        )
        rows = await cursor.fetchall()
        return [
            {
                "protocol_version": row["protocol_version"],
                "trace_id": row["trace_id"],
                "event_id": row["event_id"],
                "execution_id": row["execution_id"],
                "seq": row["seq"],
                "ts": row["ts"],
                "type": row["type"],
                "payload": json.loads(row["payload_json"]),
            }
            for row in rows
        ]

    async def list_executions_by_session(self, session_id: str) -> list[dict[str, Any]]:
        cursor = await self.conn.execute(
            "SELECT execution_id, trace_id, status, created_at, input FROM executions WHERE session_id=? ORDER BY created_at DESC",
            (session_id,),
        )
        rows = await cursor.fetchall()
        return [dict(row) for row in rows]

    async def list_sessions_by_project(self, project_id: str) -> list[dict[str, Any]]:
        cursor = await self.conn.execute(
            """
            SELECT
              s.session_id,
              s.project_id,
              s.title,
              s.updated_at,
              e.execution_id AS last_execution_id,
              e.status AS last_status,
              CASE
                WHEN e.input IS NULL THEN NULL
                ELSE substr(e.input, 1, 160)
              END AS last_input_preview
            FROM sessions s
            LEFT JOIN executions e ON e.execution_id = (
              SELECT execution_id
              FROM executions
              WHERE session_id = s.session_id
              ORDER BY created_at DESC
              LIMIT 1
            )
            WHERE s.project_id=?
            ORDER BY s.updated_at DESC
            """,
            (project_id,),
        )
        rows = await cursor.fetchall()
        return [dict(row) for row in rows]

    async def create_session(self, session_id: str, project_id: str, title: str) -> dict[str, Any]:
        await self.conn.execute(
            """
            INSERT INTO sessions(session_id, project_id, title, created_at, updated_at)
            VALUES(?, ?, ?, strftime('%Y-%m-%dT%H:%M:%fZ','now'), strftime('%Y-%m-%dT%H:%M:%fZ','now'))
            """,
            (session_id, project_id, title),
        )
        await self.conn.commit()
        cursor = await self.conn.execute(
            """
            SELECT session_id, project_id, title, updated_at, NULL AS last_execution_id, NULL AS last_status, NULL AS last_input_preview
            FROM sessions
            WHERE session_id=?
            """,
            (session_id,),
        )
        row = await cursor.fetchone()
        return dict(row) if row else {}

    async def rename_session(self, session_id: str, title: str) -> dict[str, Any] | None:
        await self.conn.execute(
            """
            UPDATE sessions
            SET title=?, updated_at=strftime('%Y-%m-%dT%H:%M:%fZ','now')
            WHERE session_id=?
            """,
            (title, session_id),
        )
        await self.conn.commit()
        cursor = await self.conn.execute(
            """
            SELECT session_id, project_id, title, updated_at
            FROM sessions
            WHERE session_id=?
            """,
            (session_id,),
        )
        row = await cursor.fetchone()
        return dict(row) if row else None

    async def get_execution(self, execution_id: str) -> dict[str, Any] | None:
        cursor = await self.conn.execute(
            """
            SELECT
              execution_id,
              project_id,
              session_id,
              model_config_id,
              input,
              workspace_path,
              trace_id,
              status,
              created_at,
              started_at,
              completed_at
            FROM executions
            WHERE execution_id=?
            """,
            (execution_id,),
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
            SELECT model_config_id, provider, model, base_url, temperature, max_tokens, secret_ref, created_at, updated_at
            FROM model_configs
            WHERE model_config_id=?
            """,
            (model_config_id,),
        )
        row = await cursor.fetchone()
        if row is None:
            return None
        return dict(row)

    async def delete_model_config(self, model_config_id: str) -> bool:
        # Keep historical runs but detach them from this model config before delete.
        try:
            await self.conn.execute(
                "UPDATE runs SET model_config_id=NULL WHERE model_config_id=?",
                (model_config_id,),
            )
        except sqlite3.OperationalError as exc:
            # Newer schemas removed `runs`; ignore this compatibility step.
            if "no such table: runs" not in str(exc):
                raise
        cursor = await self.conn.execute(
            "DELETE FROM model_configs WHERE model_config_id=?",
            (model_config_id,),
        )
        await self.conn.commit()
        return cursor.rowcount > 0

    async def insert_audit(
        self,
        *,
        audit_id: str,
        trace_id: str,
        user_id: str,
        execution_id: str | None,
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
              audit_id, trace_id, user_id, execution_id, event_id, call_id, action, tool_name, args_json, result_json,
              requires_confirmation, user_decision, decision_ts, outcome, created_at
            ) VALUES(
              ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?,
              CASE WHEN ? != 'n/a' THEN strftime('%Y-%m-%dT%H:%M:%fZ','now') ELSE NULL END,
              ?, strftime('%Y-%m-%dT%H:%M:%fZ','now')
            )
            """,
            (
                audit_id,
                trace_id,
                user_id,
                execution_id,
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

    async def upsert_confirmation_status(
        self,
        execution_id: str,
        call_id: str,
        status: str,
        *,
        decided_by: str = "user",
    ) -> None:
        if status not in {"pending", "approved", "denied"}:
            raise ValueError(f"Invalid confirmation status: {status}")

        await self.conn.execute(
            """
            INSERT INTO execution_confirmations(execution_id, call_id, status, decided_at, decided_by, created_at, updated_at)
            VALUES(
              ?, ?, ?,
              CASE WHEN ? = 'pending' THEN NULL ELSE strftime('%Y-%m-%dT%H:%M:%fZ','now') END,
              ?,
              strftime('%Y-%m-%dT%H:%M:%fZ','now'),
              strftime('%Y-%m-%dT%H:%M:%fZ','now')
            )
            ON CONFLICT(execution_id, call_id) DO UPDATE SET
              status=excluded.status,
              decided_at=excluded.decided_at,
              decided_by=excluded.decided_by,
              updated_at=excluded.updated_at
            """,
            (execution_id, call_id, status, status, decided_by),
        )
        await self.conn.commit()

    async def resolve_pending_confirmation(
        self,
        execution_id: str,
        call_id: str,
        status: str,
        *,
        decided_by: str,
    ) -> bool:
        if status not in {"approved", "denied"}:
            raise ValueError(f"Invalid confirmation status: {status}")
        cursor = await self.conn.execute(
            """
            UPDATE execution_confirmations
            SET
              status = ?,
              decided_at = strftime('%Y-%m-%dT%H:%M:%fZ','now'),
              decided_by = ?,
              updated_at = strftime('%Y-%m-%dT%H:%M:%fZ','now')
            WHERE execution_id = ?
              AND call_id = ?
              AND status = 'pending'
            """,
            (status, decided_by, execution_id, call_id),
        )
        await self.conn.commit()
        return cursor.rowcount > 0

    async def get_confirmation_status(self, execution_id: str, call_id: str) -> str | None:
        cursor = await self.conn.execute(
            "SELECT status FROM execution_confirmations WHERE execution_id=? AND call_id=?",
            (execution_id, call_id),
        )
        row = await cursor.fetchone()
        if row is None:
            return None
        return str(row["status"])

    async def list_pending_confirmations(self) -> list[dict[str, Any]]:
        cursor = await self.conn.execute(
            """
            SELECT ec.execution_id, ec.call_id
            FROM execution_confirmations ec
            JOIN executions e ON e.execution_id = ec.execution_id
            WHERE ec.status='pending' AND e.status IN ('executing', 'waiting_confirmation')
            ORDER BY e.created_at ASC
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

    async def delete_project(self, project_id: str) -> bool:
        cursor = await self.conn.execute(
            "DELETE FROM projects WHERE project_id=?",
            (project_id,),
        )
        await self.conn.commit()
        return cursor.rowcount > 0

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

    async def list_audit_logs_by_execution(self, execution_id: str, limit: int = 200) -> list[dict[str, Any]]:
        cursor = await self.conn.execute(
            """
            SELECT trace_id, audit_id, execution_id, event_id, call_id, action, tool_name, args_json, result_json,
                   requires_confirmation, user_id, user_decision, decision_ts, outcome, created_at
            FROM audit_logs
            WHERE execution_id=?
            ORDER BY created_at DESC
            LIMIT ?
            """,
            (execution_id, limit),
        )
        rows = await cursor.fetchall()
        result: list[dict[str, Any]] = []
        for row in rows:
            item = dict(row)
            item["args"] = json.loads(item.pop("args_json") or "{}")
            item["result"] = json.loads(item.pop("result_json") or "{}")
            result.append(item)
        return result
