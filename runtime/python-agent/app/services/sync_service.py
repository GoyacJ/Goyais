from __future__ import annotations

from typing import Any

import httpx

from app.db.repositories import Repository


class SyncService:
    def __init__(self, repo: Repository, sync_server_url: str, token: str, device_id: str):
        self.repo = repo
        self.sync_server_url = sync_server_url.rstrip("/")
        self.token = token
        self.device_id = device_id

    async def sync_now(self) -> dict[str, Any]:
        state = await self.repo.get_sync_state()
        local_since = int(state["last_pushed_global_seq"])
        remote_since = int(state["last_pulled_server_seq"])

        events = await self.repo.list_unsynced_events(local_since)
        push_payload = {
            "device_id": self.device_id,
            "since_global_seq": local_since,
            "events": events,
            "artifacts_meta": [],
        }

        headers = {"Authorization": f"Bearer {self.token}"}

        async with httpx.AsyncClient(timeout=30) as client:
            push_resp = await client.post(f"{self.sync_server_url}/v1/sync/push", json=push_payload, headers=headers)
            push_resp.raise_for_status()
            push_data = push_resp.json()

            pull_resp = await client.get(
                f"{self.sync_server_url}/v1/sync/pull",
                params={"since_server_seq": remote_since},
                headers=headers,
            )
            pull_resp.raise_for_status()
            pull_data = pull_resp.json()

        max_local_seq = local_since
        for event in events:
            max_local_seq = max(max_local_seq, int(event["global_seq"]))

        max_server_seq = remote_since
        for event in pull_data.get("events", []):
            inserted = await self.repo.insert_event_if_missing(event)
            if inserted:
                await self.repo.upsert_synced_event(event["event_id"], int(event.get("server_seq", 0)))
            max_server_seq = max(max_server_seq, int(event.get("server_seq", 0)))

        new_last_pushed_global_seq = max_local_seq
        await self.repo.insert_or_update_sync_state(
            last_pushed_global_seq=new_last_pushed_global_seq,
            last_pulled_server_seq=max_server_seq,
        )

        return {
            "pushed": int(push_data.get("inserted", 0)),
            "pulled": len(pull_data.get("events", [])),
            "last_pushed_global_seq": new_last_pushed_global_seq,
            "last_pulled_server_seq": max_server_seq,
        }
