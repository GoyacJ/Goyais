<template>
  <div class="chips">
    <template v-if="runtimeMode">
      <StatusBadge :tone="conversationTone" :label="conversationLabel" />
    </template>
    <template v-else-if="scopeMode === 'local'">
      <StatusBadge tone="running" :label="t('statusPanel.scope.localWorkspace')" />
      <span class="mode-tag">{{ t("statusPanel.mode.local") }}</span>
    </template>
    <template v-else>
      <StatusBadge tone="running" :label="t('statusPanel.scope.currentWorkspace')" />
      <StatusBadge tone="connected" :label="t('statusPanel.mode.remote')" />
    </template>
  </div>
</template>

<script setup lang="ts">
import { computed } from "vue";

import { t } from "@/shared/i18n";
import type { ConversationStatus } from "@/shared/types/api";
import StatusBadge from "@/shared/ui/StatusBadge.vue";

const props = withDefaults(
  defineProps<{
    runtimeMode?: boolean;
    conversationStatus?: ConversationStatus;
    scopeMode: "local" | "remote";
  }>(),
  {
    runtimeMode: false,
    conversationStatus: "stopped"
  }
);

const conversationTone = computed(() => {
  switch (props.conversationStatus) {
    case "running":
      return "running";
    case "queued":
      return "queued";
    case "done":
      return "success";
    case "error":
      return "failed";
    default:
      return "cancelled";
  }
});

const conversationLabel = computed(() => {
  const statusKey = conversationStatusLabelKey(props.conversationStatus);
  return `${t("statusPanel.conversationPrefix")}: ${t(statusKey)}`;
});

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
</script>

<style scoped>
.chips {
  display: inline-flex;
  align-items: center;
  gap: var(--global-space-8);
}

.mode-tag {
  color: var(--semantic-text-muted);
  font-size: var(--global-font-size-11);
}
</style>
