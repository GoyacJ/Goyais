<template>
  <article
    class="ui-card"
    :class="[
      interactive ? 'ui-pressable cursor-pointer' : '',
      selected ? 'ui-state-selected' : '',
    ]"
    :role="interactive ? 'button' : undefined"
    :tabindex="interactive ? 0 : undefined"
    :aria-selected="interactive ? selected : undefined"
    @click="onActivate"
    @keydown.enter.prevent="onActivate"
    @keydown.space.prevent="onActivate"
  >
    <header class="flex flex-wrap items-center justify-between gap-2">
      <div>
        <p class="ui-monospace text-xs text-ui-muted">{{ command.commandId }}</p>
        <p class="mt-1 text-sm font-semibold">{{ command.commandType }}</p>
      </div>
      <StatusBadge :status="command.status" />
    </header>

    <dl class="mt-3 grid gap-2 text-xs text-ui-muted md:grid-cols-3">
      <div>
        <dt>{{ t('page.commands.fieldAcceptedAt') }}</dt>
        <dd class="ui-monospace mt-1 text-ui-fg">{{ command.acceptedAt }}</dd>
      </div>
      <div>
        <dt>{{ t('page.commands.fieldOwner') }}</dt>
        <dd class="ui-monospace mt-1 text-ui-fg">{{ command.owner }}</dd>
      </div>
      <div>
        <dt>{{ t('page.commands.fieldTraceId') }}</dt>
        <dd class="ui-monospace mt-1 text-ui-fg">{{ command.traceId }}</dd>
      </div>
    </dl>

    <p class="mt-3 text-sm text-ui-muted">{{ command.resultSummary }}</p>
  </article>
</template>

<script setup lang="ts">
import StatusBadge from '@/components/runtime/StatusBadge.vue'
import type { CommandStatus } from '@/design-system/types'
import { useI18n } from 'vue-i18n'

interface CommandCardModel {
  commandId: string
  commandType: string
  status: CommandStatus
  acceptedAt: string
  owner: string
  traceId: string
  resultSummary: string
}

const props = withDefaults(
  defineProps<{
    command: CommandCardModel
    selected?: boolean
    interactive?: boolean
  }>(),
  {
    selected: false,
    interactive: false,
  },
)

const emit = defineEmits<{
  (e: 'select', commandId: string): void
}>()

const { t } = useI18n({ useScope: 'global' })

function onActivate(): void {
  if (!props.interactive) {
    return
  }
  emit('select', props.command.commandId)
}
</script>
