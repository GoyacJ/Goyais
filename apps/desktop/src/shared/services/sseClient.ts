import type { ExecutionEvent } from "@/shared/types/api";

type SseOptions = {
  token?: string;
  initialLastEventId?: string;
  onEvent: (event: ExecutionEvent) => void;
  onStatusChange: (status: "connected" | "reconnecting" | "disconnected") => void;
  onError: (error: Error) => void;
};

type SseHandle = {
  close: () => void;
  lastEventId: () => string;
};

const RECONNECT_DELAY_MS = 2_000;

export function connectConversationEvents(url: string, options: SseOptions): SseHandle {
  let closed = false;
  let source: EventSource | null = null;
  let lastId = options.initialLastEventId?.trim() ?? "";
  let reconnectTimer: ReturnType<typeof setTimeout> | null = null;

  const open = () => {
    if (closed) {
      return;
    }

    const fullUrl = appendQuery(url, {
      last_event_id: lastId,
      access_token: options.token ?? ""
    });
    source = new EventSource(fullUrl, { withCredentials: false });

    source.onopen = () => {
      options.onStatusChange("connected");
    };

    source.onmessage = (messageEvent) => {
      if (messageEvent.lastEventId) {
        lastId = messageEvent.lastEventId;
      }

      try {
        const parsed = JSON.parse(messageEvent.data) as ExecutionEvent;
        options.onEvent(parsed);
      } catch (error) {
        options.onError(toError(error));
      }
    };

    source.onerror = () => {
      source?.close();
      source = null;

      if (closed) {
        options.onStatusChange("disconnected");
        return;
      }

      options.onStatusChange("reconnecting");
      reconnectTimer = setTimeout(open, RECONNECT_DELAY_MS);
    };
  };

  open();

  return {
    close: () => {
      closed = true;
      if (reconnectTimer) {
        clearTimeout(reconnectTimer);
      }
      source?.close();
      source = null;
      options.onStatusChange("disconnected");
    },
    lastEventId: () => lastId
  };
}

function appendQuery(url: string, query: Record<string, string>): string {
  const search = new URLSearchParams();
  for (const [key, value] of Object.entries(query)) {
    if (value !== "") {
      search.set(key, value);
    }
  }

  if (search.size === 0) {
    return url;
  }

  const separator = url.includes("?") ? "&" : "?";
  return `${url}${separator}${search.toString()}`;
}

function toError(value: unknown): Error {
  if (value instanceof Error) {
    return value;
  }

  return new Error("Unknown SSE error");
}
