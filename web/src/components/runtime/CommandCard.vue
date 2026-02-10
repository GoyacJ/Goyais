<template>
  <article class="ui-card">
    <header class="flex flex-wrap items-center justify-between gap-2">
      <div>
        <p class="ui-monospace text-xs text-ui-muted">{{ command.commandId }}</p>
        <p class="mt-1 text-sm font-semibold">{{ command.commandType }}</p>
      </div>
      <StatusBadge :status="command.status" />
    </header>

    <dl class="mt-3 grid gap-2 text-xs text-ui-muted md:grid-cols-3">
      <div>
        <dt>acceptedAt</dt>
        <dd class="ui-monospace mt-1 text-ui-fg">{{ command.acceptedAt }}</dd>
      </div>
      <div>
        <dt>startedAt</dt>
        <dd class="ui-monospace mt-1 text-ui-fg">{{ command.startedAt }}</dd>
      </div>
      <div>
        <dt>finishedAt</dt>
        <dd class="ui-monospace mt-1 text-ui-fg">{{ command.finishedAt ?? '-' }}</dd>
      </div>
    </dl>

    <p class="mt-3 text-sm text-ui-muted">{{ command.resultSummary }}</p>
  </article>
</template>

<script setup lang="ts">
import StatusBadge from '@/components/runtime/StatusBadge.vue'
import type { MockCommand } from '@/mocks/commands'

defineProps<{
  command: MockCommand
}>()
</script>
