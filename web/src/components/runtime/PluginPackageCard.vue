<template>
  <article class="ui-card space-y-3">
    <header class="flex flex-wrap items-start justify-between gap-2">
      <div>
        <p class="text-sm font-semibold text-ui-fg">{{ item.name }}</p>
        <p class="ui-monospace mt-1 text-xs text-ui-muted">{{ item.version }}</p>
      </div>
      <Badge :tone="installTone">{{ installStatusLabel }}</Badge>
    </header>

    <dl class="grid gap-2 text-xs text-ui-muted">
      <div class="flex items-center justify-between gap-2">
        <dt>{{ t('page.plugins.fieldPackageType') }}</dt>
        <dd class="ui-monospace text-ui-fg">{{ item.packageType }}</dd>
      </div>
      <div class="flex items-center justify-between gap-2">
        <dt>{{ t('page.plugins.fieldPackageId') }}</dt>
        <dd class="ui-monospace text-ui-fg">{{ item.id }}</dd>
      </div>
      <div v-if="item.installId" class="flex items-center justify-between gap-2">
        <dt>{{ t('page.plugins.fieldInstallId') }}</dt>
        <dd class="ui-monospace text-ui-fg">{{ item.installId }}</dd>
      </div>
      <div v-if="item.lastCommandId" class="flex items-center justify-between gap-2">
        <dt>{{ t('page.plugins.fieldCommandId') }}</dt>
        <dd class="ui-monospace text-ui-fg">{{ item.lastCommandId }}</dd>
      </div>
    </dl>

    <div class="flex flex-wrap gap-2">
      <Button
        v-if="showInstall"
        :disabled="busy"
        @click="$emit('install', item.id)"
      >
        {{ t('page.plugins.actionInstall') }}
      </Button>
      <Button
        v-if="showEnable"
        :disabled="busy || !item.installId"
        @click="$emit('enable', item.installId as string)"
      >
        {{ t('common.enable') }}
      </Button>
      <Button
        v-if="showDisable"
        variant="secondary"
        :disabled="busy || !item.installId"
        @click="$emit('disable', item.installId as string)"
      >
        {{ t('common.disable') }}
      </Button>
      <Button
        v-if="showRollback"
        variant="ghost"
        :disabled="busy || !item.installId"
        @click="$emit('rollback', item.installId as string)"
      >
        {{ t('page.plugins.actionRollback') }}
      </Button>
    </div>
  </article>
</template>

<script setup lang="ts">
import Badge from '@/components/ui/Badge.vue'
import Button from '@/components/ui/Button.vue'
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'

export interface PluginPackageCardItem {
  id: string
  name: string
  version: string
  packageType: string
  installId: string | null
  installStatus: string
  lastCommandId: string | null
}

const props = withDefaults(
  defineProps<{
    item: PluginPackageCardItem
    busy?: boolean
  }>(),
  {
    busy: false,
  },
)

defineEmits<{
  (e: 'install', packageId: string): void
  (e: 'enable', installId: string): void
  (e: 'disable', installId: string): void
  (e: 'rollback', installId: string): void
}>()

const { t } = useI18n({ useScope: 'global' })

const normalizedInstallStatus = computed(() => props.item.installStatus.trim().toLowerCase())

const installTone = computed(() => {
  switch (normalizedInstallStatus.value) {
    case 'enabled':
      return 'success'
    case 'disabled':
      return 'warn'
    case 'failed':
      return 'error'
    case 'rolled_back':
      return 'neutral'
    case 'installing':
    case 'validating':
    case 'uploaded':
      return 'primary'
    default:
      return 'neutral'
  }
})

const installStatusLabel = computed(() => {
  const key = `page.plugins.installStatus.${normalizedInstallStatus.value}`
  const resolved = t(key)
  return resolved === key ? props.item.installStatus : resolved
})

const showInstall = computed(() => {
  if (!props.item.installId) {
    return true
  }
  return normalizedInstallStatus.value === 'failed'
})

const showEnable = computed(() => normalizedInstallStatus.value === 'disabled' || normalizedInstallStatus.value === 'rolled_back')
const showDisable = computed(() => normalizedInstallStatus.value === 'enabled')
const showRollback = computed(
  () =>
    normalizedInstallStatus.value === 'enabled' ||
    normalizedInstallStatus.value === 'disabled' ||
    normalizedInstallStatus.value === 'failed',
)
</script>
