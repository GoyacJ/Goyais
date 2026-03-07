import { computed } from "vue";

import { projectStore } from "@/modules/project/store";
import { t } from "@/shared/i18n";
import { useWorkspaceStatusSync } from "@/shared/stores/workspaceStatusStore";
import type { ConnectionStatus, ConversationStatus } from "@/shared/types/api";

type ConnectionTone = "connected" | "disconnected";

export function useConfigRuntimeStatus() {
  const workspaceStatus = useWorkspaceStatusSync({
    conversationId: computed(() => projectStore.activeConversationId)
  });

  const conversationStatus = computed<ConversationStatus>(() => workspaceStatus.conversationStatus.value);
  const connectionStatus = computed<ConnectionStatus>(() => workspaceStatus.connectionStatus.value);
  const userDisplayName = computed(() => workspaceStatus.userDisplayName.value);
  const hubUrl = computed(() => workspaceStatus.hubURL.value);

  const conversationLabel = computed(() => {
    return `${t("statusPanel.conversationPrefix")}: ${t(conversationStatusLabelKey(conversationStatus.value))}`;
  });

  const connectionLabel = computed(() => t(connectionStatusLabelKey(connectionStatus.value)));

  const connectionTone = computed<ConnectionTone>(() => {
    return connectionStatus.value === "connected" ? "connected" : "disconnected";
  });

  return {
    runtimeStatusMode: computed(() => true),
    conversationStatus,
    connectionStatus,
    userDisplayName,
    hubUrl,
    conversationLabel,
    connectionLabel,
    connectionTone
  };
}

function conversationStatusLabelKey(status: ConversationStatus): string {
  switch (status) {
    case "running":
      return "statusPanel.conversationStatus.running";
    case "queued":
      return "statusPanel.conversationStatus.queued";
    case "done":
      return "statusPanel.conversationStatus.done";
    case "error":
      return "statusPanel.conversationStatus.error";
    default:
      return "statusPanel.conversationStatus.stopped";
  }
}

function connectionStatusLabelKey(status: ConnectionStatus): string {
  switch (status) {
    case "connected":
      return "statusPanel.connectionStatus.connected";
    case "reconnecting":
      return "statusPanel.connectionStatus.reconnecting";
    default:
      return "statusPanel.connectionStatus.disconnected";
  }
}
