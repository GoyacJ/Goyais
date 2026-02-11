<template>
  <section class="ui-detail-block">
    <header class="ui-detail-header">
      <button
        type="button"
        class="ui-control ui-focus-ring ui-pressable text-sm"
        @click="expanded = !expanded"
      >
        {{ expanded ? t('common.hideLogs') : t('common.showLogs') }}
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
      class="ui-log-surface ui-monospace ui-scrollbar max-h-64 overflow-auto p-3 text-xs leading-relaxed"
    ><code>{{ lines.join('\n') }}</code></pre>
  </section>
</template>

<script setup lang="ts">
/**
 * SPDX-License-Identifier: Apache-2.0
 * Copyright (c) 2026 Goya
 * Author: Goya
 * Created: 2026-02-11
 * Version: v1.0.0
 * Description: Goyais source file.
 */
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
