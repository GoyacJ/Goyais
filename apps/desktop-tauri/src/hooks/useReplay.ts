import { useEffect, useState } from "react";

import { replayRunEvents } from "../api/runtimeClient";
import type { EventEnvelope } from "../types/generated";

export function useReplay(runId?: string) {
  const [events, setEvents] = useState<EventEnvelope[]>([]);
  const [loading, setLoading] = useState(false);

  useEffect(() => {
    if (!runId) {
      setEvents([]);
      return;
    }

    setLoading(true);
    replayRunEvents(runId)
      .then((payload) => setEvents(payload.events))
      .finally(() => setLoading(false));
  }, [runId]);

  return { events, loading };
}
