<template>
  <section
    class="conversation-layout grid min-h-[calc(100dvh-120px)] gap-[var(--component-space-md)] [grid-template-columns:minmax(220px,280px)_1fr] max-[900px]:[grid-template-columns:minmax(0,1fr)]"
  >
    <ConversationSidebarPlaceholder />
    <div class="main-pane grid content-start gap-[var(--component-space-md)]">
      <ConversationEventStreamPlaceholder />
      <ConversationComposerPlaceholder />
    </div>
  </section>
</template>

<script setup lang="ts">
import ConversationComposerPlaceholder from "@/modules/session/components/ConversationComposerPlaceholder.vue";
import ConversationEventStreamPlaceholder from "@/modules/session/components/ConversationEventStreamPlaceholder.vue";
import ConversationSidebarPlaceholder from "@/modules/session/components/ConversationSidebarPlaceholder.vue";
</script>
