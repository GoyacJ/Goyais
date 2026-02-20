import { useEffect, useMemo, useRef } from "react";

import { getRunDataSource } from "@/api/runDataSource";

import { assertEventEnvelope } from "../api/protocolValidators";
import { useRunStore } from "../stores/runStore";
import { selectCurrentProfile, useWorkspaceStore } from "../stores/workspaceStore";

export function useRunEvents(runId?: string) {
  const appendEvent = useRunStore((state) => state.appendEvent);
  const currentProfile = useWorkspaceStore(selectCurrentProfile);
  const runDataSource = useMemo(() => getRunDataSource(currentProfile), [currentProfile]);
  const sourceRef = useRef<{ close: () => void } | null>(null);

  useEffect(() => {
    if (!runId) {
      sourceRef.current?.close();
      sourceRef.current = null;
      return;
    }

    const source = runDataSource.subscribeRunEvents(
      runId,
      (event) => {
        if (!assertEventEnvelope(event)) {
          return;
        }
        appendEvent(event);
      },
      () => {
        sourceRef.current?.close();
      }
    );

    sourceRef.current = source;

    return () => {
      source.close();
      if (sourceRef.current === source) {
        sourceRef.current = null;
      }
    };
  }, [runId, appendEvent, runDataSource]);
}
