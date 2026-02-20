from __future__ import annotations

import asyncio
from pathlib import Path

from app.config import load_settings
from app.db.connection import open_connection


async def apply_migrations() -> None:
    settings = load_settings()
    migrations_dir = Path(__file__).resolve().parents[2] / "migrations"
    migration_files = sorted(migrations_dir.glob("*.sql"))

    conn = await open_connection(settings.db_path)
    try:
        await conn.execute(
            """
            CREATE TABLE IF NOT EXISTS schema_migrations (
              version TEXT PRIMARY KEY,
              applied_at TEXT NOT NULL
            )
            """
        )
        await conn.commit()

        for migration_path in migration_files:
            version = migration_path.name
            cursor = await conn.execute(
                "SELECT 1 FROM schema_migrations WHERE version = ?", (version,)
            )
            row = await cursor.fetchone()
            if row:
                continue

            sql = migration_path.read_text(encoding="utf-8")
            await conn.executescript(sql)
            await conn.execute(
                "INSERT INTO schema_migrations(version, applied_at) VALUES(?, strftime('%Y-%m-%dT%H:%M:%fZ','now'))",
                (version,),
            )
            await conn.commit()
    finally:
        await conn.close()


if __name__ == "__main__":
    asyncio.run(apply_migrations())
