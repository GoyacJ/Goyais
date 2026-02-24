<template>
  <section class="conversation-layout">
    <ConversationSidebarPlaceholder />
    <div class="main-pane">
      <ConversationEventStreamPlaceholder />
      <ConversationComposerPlaceholder />
    </div>
  </section>
</template>

<script setup lang="ts">
import ConversationComposerPlaceholder from "@/modules/conversation/components/ConversationComposerPlaceholder.vue";
import ConversationEventStreamPlaceholder from "@/modules/conversation/components/ConversationEventStreamPlaceholder.vue";
import ConversationSidebarPlaceholder from "@/modules/conversation/components/ConversationSidebarPlaceholder.vue";
</script>

<style scoped>
.conversation-layout {
  display: grid;
  grid-template-columns: minmax(220px, 280px) 1fr;
  gap: var(--component-space-md);
  min-height: calc(100vh - 120px);
}

.main-pane {
  display: grid;
  gap: var(--component-space-md);
  align-content: start;
}

@media (max-width: 900px) {
  .conversation-layout {
    grid-template-columns: 1fr;
  }
}
</style>
