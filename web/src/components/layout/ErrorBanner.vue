<template>
  <div class="ui-card border-ui-borderSubtle">
    <p class="text-sm font-semibold text-ui-fg">
      <span class="ui-empty-tone-dot ui-empty-tone-dot--error" />
      {{ translated }}
    </p>
    <p class="ui-monospace mt-1 text-xs">{{ error.code }} · {{ error.messageKey }}</p>
  </div>
</template>

<script setup lang="ts">
import { translateBackendError, type BackendErrorShape } from '@/i18n/error'
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'

const props = defineProps<{
  error: BackendErrorShape
}>()

const { t } = useI18n({ useScope: 'global' })

const translated = computed(() => translateBackendError(t, props.error))
</script>
