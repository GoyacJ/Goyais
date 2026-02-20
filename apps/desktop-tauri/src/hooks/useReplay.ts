import { useEffect, useMemo, useState } from "react";

import { getRunDataSource } from "@/api/runDataSource";
import { selectCurrentProfile, useWorkspaceStore } from "@/stores/workspaceStore";

import type { EventEnvelope } from "../types/generated";

export function useReplay(runId?: string) {
  const currentProfile = useWorkspaceStore(selectCurrentProfile);
  const runDataSource = useMemo(() => getRunDataSource(currentProfile), [currentProfile]);
  const [events, setEvents] = useState<EventEnvelope[]>([]);
  const [loading, setLoading] = useState(false);

  useEffect(() => {
    if (!runId) {
      setEvents([]);
      return;
    }

    setLoading(true);
    runDataSource
      .replayRunEvents(runId)
      .then((payload) => setEvents(payload.events))
      .finally(() => setLoading(false));
  }, [runId, runDataSource]);

  return { events, loading };
}
