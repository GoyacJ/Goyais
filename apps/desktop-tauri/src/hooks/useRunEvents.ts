import { useEffect, useRef } from "react";

import { assertEventEnvelope } from "../api/protocolValidators";
import { subscribeRunEvents } from "../api/runtimeClient";
import { useRunStore } from "../stores/runStore";

export function useRunEvents(runId?: string) {
  const appendEvent = useRunStore((state) => state.appendEvent);
  const sourceRef = useRef<EventSource | null>(null);

  useEffect(() => {
    if (!runId) {
      sourceRef.current?.close();
      sourceRef.current = null;
      return;
    }

    const source = subscribeRunEvents(runId, (event) => {
      if (!assertEventEnvelope(event)) {
        return;
      }
      appendEvent(event);
    });

    source.onerror = () => {
      source.close();
    };

    sourceRef.current = source;

    return () => {
      source.close();
      if (sourceRef.current === source) {
        sourceRef.current = null;
      }
    };
  }, [runId, appendEvent]);
}
