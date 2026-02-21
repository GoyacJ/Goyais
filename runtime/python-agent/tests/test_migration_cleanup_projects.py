from __future__ import annotations

import asyncio
from pathlib import Path

from app.db.connection import open_connection
from app.db.migrations import apply_migrations


async def _apply_migrations_until_0008(db_path):
    conn = await open_connection(db_path)
    try:
        await conn.execute(
            """
            CREATE TABLE IF NOT EXISTS schema_migrations (
              version TEXT PRIMARY KEY,
              applied_at TEXT NOT NULL
            )
            """
        )
        migrations_dir = Path(__file__).resolve().parents[1] / "migrations"
        for path in sorted(migrations_dir.glob("*.sql")):
            if path.name > "0008_model_provider_matrix.sql":
                continue
            sql = path.read_text(encoding="utf-8")
            await conn.executescript(sql)
            await conn.execute(
                "INSERT OR IGNORE INTO schema_migrations(version, applied_at) VALUES(?, strftime('%Y-%m-%dT%H:%M:%fZ','now'))",
                (path.name,),
            )
        await conn.commit()
    finally:
        await conn.close()


async def _prepare_and_assert(db_path):
    conn = await open_connection(db_path)
    try:
        await conn.execute(
            """
            INSERT INTO projects(project_id, name, workspace_path, created_at, updated_at)
            VALUES('project-sessions-abcd', 'Session Project', '/tmp/s1', '2026-01-01T00:00:00.000Z', '2026-01-01T00:00:00.000Z')
            """
        )
        await conn.execute(
            """
            INSERT INTO projects(project_id, name, workspace_path, created_at, updated_at)
            VALUES('project-rename-abcd', 'Session Project', '/tmp/s2', '2026-01-01T00:00:00.000Z', '2026-01-01T00:00:00.000Z')
            """
        )
        await conn.execute(
            """
            INSERT INTO projects(project_id, name, workspace_path, created_at, updated_at)
            VALUES('diag-project-diag-execution-abcd', 'diag-project-diag-execution-abcd', '/tmp/s3', '2026-01-01T00:00:00.000Z', '2026-01-01T00:00:00.000Z')
            """
        )
        await conn.execute(
            """
            INSERT INTO projects(project_id, name, workspace_path, created_at, updated_at)
            VALUES('real-project-1', 'Real Project', '/tmp/real', '2026-01-01T00:00:00.000Z', '2026-01-01T00:00:00.000Z')
            """
        )
        await conn.commit()
    finally:
        await conn.close()

    await apply_migrations()

    conn = await open_connection(db_path)
    try:
        cursor = await conn.execute("SELECT project_id FROM projects ORDER BY project_id ASC")
        rows = await cursor.fetchall()
        project_ids = [str(row["project_id"]) for row in rows]
        assert "real-project-1" in project_ids
        assert all(not item.startswith("project-sessions") for item in project_ids)
        assert all(not item.startswith("project-rename") for item in project_ids)
        assert all(not item.startswith("diag-project-diag-execution-") for item in project_ids)
    finally:
        await conn.close()


def test_cleanup_test_projects_migration(tmp_path, monkeypatch):
    db_path = tmp_path / "runtime-cleanup.db"
    monkeypatch.setenv("GOYAIS_DB_PATH", str(db_path))
    asyncio.run(_apply_migrations_until_0008(db_path))
    asyncio.run(_prepare_and_assert(db_path))
