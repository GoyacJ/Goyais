import { computed, onBeforeUnmount, reactive, toValue, watch, type MaybeRefOrGetter } from "vue";

import { streamConversationEvents } from "@/modules/conversation/services";
import { getWorkspaceStatus } from "@/modules/workspace/services";
import { ApiError } from "@/shared/services/http";
import { authStore } from "@/shared/stores/authStore";
import { getCurrentWorkspace, workspaceStore } from "@/shared/stores/workspaceStore";
import type { ConnectionStatus, ConversationStatus, WorkspaceStatusResponse } from "@/shared/types/api";

type WorkspaceStatusSyncOptions = {
  conversationId?: MaybeRefOrGetter<string | undefined | null>;
};

type WorkspaceStatusSyncState = {
  snapshot: WorkspaceStatusResponse;
  loading: boolean;
  error: string;
};

type StreamHandle = {
  close: () => void;
};

const DEFAULT_HUB_URL = "local://workspace";
const DEFAULT_USER_DISPLAY_NAME = "local-user";

export function useWorkspaceStatusSync(options: WorkspaceStatusSyncOptions = {}) {
  const state = reactive<WorkspaceStatusSyncState>({
    snapshot: buildDegradedSnapshot("", ""),
    loading: false,
    error: ""
  });

  const requestedConversationID = computed(() => normalizeID(toValue(options.conversationId)));

  let disposed = false;
  let refreshSequence = 0;
  let streamConversationID = "";
  let streamHandle: StreamHandle | null = null;
  let syncContext = {
    workspaceID: "",
    conversationID: "",
    token: ""
  };

  const stopWatch = watch(
    [
      () => workspaceStore.currentWorkspaceId,
      () => requestedConversationID.value,
      () => {
        const workspaceID = workspaceStore.currentWorkspaceId;
        if (workspaceID === "") {
          return "";
        }
        return (authStore.tokensByWorkspaceId[workspaceID] ?? "").trim();
      }
    ],
    ([workspaceID, conversationID, token]) => {
      syncContext = { workspaceID, conversationID, token };
      void refreshStatus();
    },
    { immediate: true }
  );

  async function refreshStatus(): Promise<void> {
    if (disposed) {
      return;
    }

    const context = { ...syncContext };
    if (context.workspaceID === "") {
      state.snapshot = buildDegradedSnapshot("", context.conversationID);
      state.error = "";
      closeStream();
      return;
    }

    const currentRequest = ++refreshSequence;
    state.loading = true;
    try {
      const response = await getWorkspaceStatus(context.workspaceID, {
        conversationId: context.conversationID === "" ? undefined : context.conversationID,
        token: context.token === "" ? undefined : context.token
      });
      if (disposed || currentRequest !== refreshSequence) {
        return;
      }

      state.snapshot = normalizeSnapshot(response);
      state.error = "";
      rebindStream(state.snapshot.conversation_id ?? "", context.token);
    } catch (error) {
      if (disposed || currentRequest !== refreshSequence) {
        return;
      }

      state.error = formatStatusError(error);
      state.snapshot = buildDegradedSnapshot(context.workspaceID, context.conversationID);
      closeStream();
    } finally {
      if (!disposed && currentRequest === refreshSequence) {
        state.loading = false;
      }
    }
  }

  function rebindStream(conversationID: string, token: string): void {
    const normalizedConversationID = normalizeID(conversationID);
    if (normalizedConversationID === "") {
      closeStream();
      return;
    }
    if (streamConversationID === normalizedConversationID && streamHandle) {
      return;
    }

    closeStream();
    if (typeof EventSource === "undefined") {
      return;
    }

    streamConversationID = normalizedConversationID;
    streamHandle = streamConversationEvents(normalizedConversationID, {
      token: token === "" ? undefined : token,
      onEvent: () => {
        void refreshStatus();
      },
      onStatusChange: (status) => {
        if (disposed) {
          return;
        }
        if (status === "reconnecting") {
          state.snapshot = { ...state.snapshot, connection_status: "reconnecting" };
          return;
        }
        if (status === "disconnected") {
          state.snapshot = { ...state.snapshot, connection_status: "disconnected" };
        }
      },
      onError: (error) => {
        if (disposed) {
          return;
        }
        state.error = formatStatusError(error);
      }
    });
  }

  function closeStream(): void {
    streamHandle?.close();
    streamHandle = null;
    streamConversationID = "";
  }

  onBeforeUnmount(() => {
    disposed = true;
    stopWatch();
    closeStream();
  });

  return {
    status: computed(() => state.snapshot),
    conversationStatus: computed(() => state.snapshot.conversation_status),
    conversationID: computed(() => state.snapshot.conversation_id ?? ""),
    hubURL: computed(() => state.snapshot.hub_url),
    connectionStatus: computed(() => state.snapshot.connection_status),
    userDisplayName: computed(() => state.snapshot.user_display_name),
    updatedAt: computed(() => state.snapshot.updated_at),
    loading: computed(() => state.loading),
    error: computed(() => state.error),
    refresh: refreshStatus
  };
}

function buildDegradedSnapshot(workspaceID: string, conversationID: string): WorkspaceStatusResponse {
  const workspace = getCurrentWorkspace();
  const hubURL = normalizeURL(workspace?.hub_url ?? "");
  const normalizedConversationID = normalizeID(conversationID);

  return {
    workspace_id: workspaceID !== "" ? workspaceID : workspace?.id ?? "",
    ...(normalizedConversationID === "" ? {} : { conversation_id: normalizedConversationID }),
    conversation_status: "stopped",
    hub_url: hubURL === "" ? DEFAULT_HUB_URL : hubURL,
    connection_status: "disconnected",
    user_display_name: resolveUserDisplayName(),
    updated_at: new Date().toISOString()
  };
}

function normalizeSnapshot(input: WorkspaceStatusResponse): WorkspaceStatusResponse {
  const workspace = getCurrentWorkspace();
  const normalizedConversationID = normalizeID(input.conversation_id);

  return {
    workspace_id: normalizeID(input.workspace_id) !== "" ? normalizeID(input.workspace_id) : workspace?.id ?? "",
    ...(normalizedConversationID === "" ? {} : { conversation_id: normalizedConversationID }),
    conversation_status: normalizeConversationStatus(input.conversation_status),
    hub_url: firstNonEmpty(normalizeURL(input.hub_url), normalizeURL(workspace?.hub_url ?? ""), DEFAULT_HUB_URL),
    connection_status: normalizeConnectionStatus(input.connection_status),
    user_display_name: firstNonEmpty(normalizeID(input.user_display_name), resolveUserDisplayName(), DEFAULT_USER_DISPLAY_NAME),
    updated_at: normalizeID(input.updated_at) || new Date().toISOString()
  };
}

function normalizeConversationStatus(value: string): ConversationStatus {
  switch (value) {
    case "running":
    case "queued":
    case "done":
    case "error":
    case "stopped":
      return value;
    default:
      return "stopped";
  }
}

function normalizeConnectionStatus(value: string): ConnectionStatus {
  switch (value) {
    case "connected":
    case "reconnecting":
    case "disconnected":
      return value;
    default:
      return "disconnected";
  }
}

function resolveUserDisplayName(): string {
  return firstNonEmpty(
    normalizeID(authStore.me?.display_name ?? ""),
    normalizeID(authStore.me?.user_id ?? ""),
    DEFAULT_USER_DISPLAY_NAME
  );
}

function normalizeURL(value: string | null): string {
  return (value ?? "").trim();
}

function normalizeID(value: string | undefined | null): string {
  return (value ?? "").trim();
}

function firstNonEmpty(...values: string[]): string {
  for (const value of values) {
    if (value !== "") {
      return value;
    }
  }
  return "";
}

function formatStatusError(error: unknown): string {
  if (error instanceof ApiError) {
    return `${error.message} (${error.code})`;
  }
  if (error instanceof Error) {
    return error.message;
  }
  return "Unknown workspace status error";
}
