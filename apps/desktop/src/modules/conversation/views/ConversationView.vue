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
import ConversationComposerPlaceholder from "@/modules/conversation/components/ConversationComposerPlaceholder.vue";
import ConversationEventStreamPlaceholder from "@/modules/conversation/components/ConversationEventStreamPlaceholder.vue";
import ConversationSidebarPlaceholder from "@/modules/conversation/components/ConversationSidebarPlaceholder.vue";
</script>
