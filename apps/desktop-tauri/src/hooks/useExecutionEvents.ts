import { useCallback, useEffect, useMemo, useRef } from "react";

import { getSessionDataSource } from "@/api/sessionDataSource";
import { useExecutionStore } from "@/stores/executionStore";
import { selectCurrentProfile, useWorkspaceStore } from "@/stores/workspaceStore";

interface UseExecutionEventsOptions {
  sessionId: string | null | undefined;
  enabled: boolean;
}

export function useExecutionEvents({ sessionId, enabled }: UseExecutionEventsOptions) {
  const profile = useWorkspaceStore(selectCurrentProfile);
  const dataSource = useMemo(() => getSessionDataSource(profile), [profile]);
  const appendRawEvent = useExecutionStore((state) => state.appendRawEvent);
  const executionId = useExecutionStore((state) => state.executionId);

  const cleanupRef = useRef<(() => void) | null>(null);
  const reconnectTimerRef = useRef<number | null>(null);

  const clearReconnectTimer = useCallback(() => {
    if (reconnectTimerRef.current !== null) {
      window.clearTimeout(reconnectTimerRef.current);
      reconnectTimerRef.current = null;
    }
  }, []);

  const connect = useCallback(() => {
    if (!sessionId || !enabled) {
      return;
    }

    cleanupRef.current?.();
    cleanupRef.current = null;
    clearReconnectTimer();

    const sinceSeq = useExecutionStore.getState().lastSeq;
    const sub = dataSource.subscribeSessionEvents(
      sessionId,
      sinceSeq,
      (type, payloadJson, seq) => {
        appendRawEvent(type, payloadJson, seq);
      },
      () => {
        cleanupRef.current?.();
        cleanupRef.current = null;
        clearReconnectTimer();
        reconnectTimerRef.current = window.setTimeout(() => {
          connect();
        }, 2000);
      }
    );

    cleanupRef.current = () => {
      sub.close();
    };
  }, [appendRawEvent, clearReconnectTimer, dataSource, enabled, sessionId]);

  useEffect(() => {
    connect();
    return () => {
      cleanupRef.current?.();
      cleanupRef.current = null;
      clearReconnectTimer();
    };
    // reconnect when execution switches in the same session
  }, [connect, clearReconnectTimer, executionId]);
}
