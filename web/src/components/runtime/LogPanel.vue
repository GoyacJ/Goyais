<template>
  <section class="ui-card">
    <header class="flex flex-wrap items-center justify-between gap-2">
      <button
        type="button"
        class="ui-control ui-focus-ring ui-pressable text-sm"
        @click="expanded = !expanded"
      >
        {{ expanded ? 'Hide logs' : 'Show logs' }}
      </button>
      <button
        type="button"
        class="ui-control ui-focus-ring ui-pressable text-sm"
        @click="copyLogs"
      >
        {{ copied ? t('common.copied') : t('common.copy') }}
      </button>
    </header>

    <pre
      v-if="expanded"
      class="ui-monospace ui-scrollbar mt-3 max-h-64 overflow-auto rounded-button border border-ui-border bg-slate-950 p-3 text-xs leading-relaxed text-slate-100"
    ><code>{{ lines.join('\n') }}</code></pre>
  </section>
</template>

<script setup lang="ts">
import { ref } from 'vue'
import { useI18n } from 'vue-i18n'

const props = defineProps<{
  lines: string[]
}>()

const { t } = useI18n({ useScope: 'global' })
const expanded = ref(true)
const copied = ref(false)

async function copyLogs(): Promise<void> {
  await navigator.clipboard.writeText(props.lines.join('\n'))
  copied.value = true
  window.setTimeout(() => {
    copied.value = false
  }, 1200)
}
</script>
