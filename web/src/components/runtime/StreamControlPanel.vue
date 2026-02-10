<template>
  <div class="space-y-4">
    <section class="ui-surface p-3">
      <h3 class="text-sm font-semibold text-ui-fg">{{ t('page.streams.createTitle') }}</h3>
      <div class="mt-3 grid gap-2">
        <Input v-model="path" :placeholder="t('page.streams.fieldPath')" />
        <div class="grid gap-2 md:grid-cols-2">
          <Select v-model="protocol" :options="protocolOptions" />
          <Select v-model="source" :options="sourceOptions" />
        </div>
        <div class="grid gap-2 md:grid-cols-2">
          <Select v-model="visibility" :options="visibilityOptions" />
          <Input v-model="onPublishTemplateId" :placeholder="t('page.streams.fieldOnPublishTemplateId')" />
        </div>
      </div>
      <Button class="mt-3" :disabled="busy || path.trim().length === 0" @click="onCreate">
        {{ t('page.streams.actionCreate') }}
      </Button>
    </section>

    <section class="ui-surface p-3">
      <header class="flex items-center justify-between gap-2">
        <h3 class="text-sm font-semibold text-ui-fg">{{ t('page.streams.selectionTitle') }}</h3>
        <Badge :tone="statusTone">{{ selectedStream?.status ?? '-' }}</Badge>
      </header>
      <div v-if="selectedStream" class="mt-2 grid gap-2 text-xs text-ui-muted">
        <div class="flex items-center justify-between gap-2">
          <span>{{ t('page.streams.fieldPath') }}</span>
          <span class="ui-monospace text-ui-fg">{{ selectedStream.path }}</span>
        </div>
        <div class="flex items-center justify-between gap-2">
          <span>{{ t('page.streams.fieldProtocol') }}</span>
          <span class="ui-monospace text-ui-fg">{{ selectedStream.protocol }}</span>
        </div>
        <div class="flex items-center justify-between gap-2">
          <span>{{ t('page.streams.fieldStreamId') }}</span>
          <span class="ui-monospace text-ui-fg">{{ selectedStream.id }}</span>
        </div>
      </div>
      <p v-else class="mt-2 text-sm text-ui-muted">{{ t('page.streams.selectionEmpty') }}</p>

      <div class="mt-3 flex flex-wrap gap-2">
        <Button variant="secondary" :disabled="busy || !selectedStream" @click="$emit('recordStart')">
          {{ t('page.streams.actionRecordStart') }}
        </Button>
        <Button variant="secondary" :disabled="busy || !selectedStream" @click="$emit('recordStop')">
          {{ t('page.streams.actionRecordStop') }}
        </Button>
        <Button variant="ghost" :disabled="busy || !selectedStream" @click="$emit('kick')">
          {{ t('page.streams.actionKick') }}
        </Button>
      </div>
    </section>
  </div>
</template>

<script setup lang="ts">
import type { StreamDTO } from '@/api/types'
import Badge from '@/components/ui/Badge.vue'
import Button from '@/components/ui/Button.vue'
import Input from '@/components/ui/Input.vue'
import Select from '@/components/ui/Select.vue'
import { computed, ref } from 'vue'
import { useI18n } from 'vue-i18n'

export interface StreamCreateFormValue {
  path: string
  protocol: string
  source: string
  visibility: string
  onPublishTemplateId?: string
}

const props = withDefaults(
  defineProps<{
    selectedStream: StreamDTO | null
    busy?: boolean
  }>(),
  {
    busy: false,
  },
)

const emit = defineEmits<{
  (e: 'create', payload: StreamCreateFormValue): void
  (e: 'recordStart'): void
  (e: 'recordStop'): void
  (e: 'kick'): void
}>()

const { t } = useI18n({ useScope: 'global' })

const path = ref('')
const protocol = ref('rtmp')
const source = ref('push')
const visibility = ref('PRIVATE')
const onPublishTemplateId = ref('')

const protocolOptions = computed(() => [
  { value: 'rtsp', label: 'rtsp' },
  { value: 'rtmp', label: 'rtmp' },
  { value: 'srt', label: 'srt' },
  { value: 'webrtc', label: 'webrtc' },
  { value: 'hls', label: 'hls' },
])

const sourceOptions = computed(() => [
  { value: 'push', label: 'push' },
  { value: 'pull', label: 'pull' },
])

const visibilityOptions = computed(() => [
  { value: 'PRIVATE', label: 'PRIVATE' },
  { value: 'WORKSPACE', label: 'WORKSPACE' },
])

const statusTone = computed(() => {
  const status = props.selectedStream?.status?.toLowerCase()
  switch (status) {
    case 'online':
      return 'success'
    case 'recording':
      return 'warn'
    case 'offline':
      return 'neutral'
    case 'error':
      return 'error'
    default:
      return 'neutral'
  }
})

function onCreate(): void {
  if (path.value.trim().length === 0) {
    return
  }
  emit('create', {
    path: path.value.trim(),
    protocol: protocol.value,
    source: source.value,
    visibility: visibility.value,
    onPublishTemplateId: onPublishTemplateId.value.trim(),
  })
}
</script>
