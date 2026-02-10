<template>
  <section class="ui-page">
    <PageHeader :title="t('page.settings.title')" :subtitle="t('page.settings.subtitle')" />

    <WindowBoard route-key="settings" :panes="windowPanes">
      <template #preferences>
        <SectionCard title="Theme / Density / Locale" subtitle="Persisted in localStorage">
          <div class="grid gap-3 md:grid-cols-3">
            <label class="text-xs text-ui-muted">
              {{ t('common.theme') }}
              <Select v-model="themeModeModel" :options="themeOptions" />
            </label>

            <label class="text-xs text-ui-muted">
              {{ t('common.density') }}
              <Select v-model="densityModeModel" :options="densityOptions" />
            </label>

            <label class="text-xs text-ui-muted">
              {{ t('common.language') }}
              <Select v-model="localeModel" :options="localeOptions" />
            </label>
          </div>

          <p class="ui-monospace mt-3 text-xs text-ui-muted">
            theme={{ themeModeModel }} density={{ densityModeModel }} locale={{ localeModel }}
          </p>
        </SectionCard>
      </template>

      <template #component-matrix>
        <SectionCard title="Component Matrix" subtitle="State semantics and focus behavior">
          <Tabs v-model="activeTab" :tabs="tabItems" aria-label="settings tabs" />

          <div class="mt-3">
            <template v-if="activeTab === 'controls'">
              <div class="grid gap-3 md:grid-cols-2">
                <Input v-model="textValue" placeholder="Input tokenized control" />
                <Textarea v-model="areaValue" placeholder="Textarea tokenized control" />
              </div>
              <div class="mt-3 flex flex-wrap gap-2">
                <Button>Primary</Button>
                <Button variant="secondary">Secondary</Button>
                <Button variant="ghost">Ghost</Button>
                <Button variant="destructive">Destructive</Button>
                <Button loading>Loading</Button>
                <Button disabled>Disabled</Button>
              </div>
            </template>

            <template v-else-if="activeTab === 'overlay'">
              <div class="flex flex-wrap gap-2">
                <Button ref="dialogTriggerRef" @click="openDialog">Open Dialog</Button>
                <Dropdown :items="menuItems" label="Open Dropdown" @select="onSelectMenu" />
              </div>
              <p class="mt-3 text-sm text-ui-muted">{{ lastMenuAction }}</p>
            </template>

            <template v-else>
              <SkeletonBlock />
            </template>
          </div>
        </SectionCard>
      </template>
    </WindowBoard>

    <Dialog
      :open="dialogOpen"
      title="Keyboard A11y Check"
      description="Use Tab and Shift+Tab to verify focus loop, ESC to close."
      :cancel-label="t('common.close')"
      :confirm-label="t('common.confirm')"
      @close="closeDialog"
      @confirm="onConfirmDialog"
    >
      <div class="space-y-3">
        <Input v-model="dialogInput" placeholder="Focusable input in dialog" />
        <Button variant="secondary">Focusable action</Button>
      </div>
    </Dialog>
  </section>
</template>

<script setup lang="ts">
import PageHeader from '@/components/layout/PageHeader.vue'
import SectionCard from '@/components/layout/SectionCard.vue'
import SkeletonBlock from '@/components/layout/SkeletonBlock.vue'
import WindowBoard from '@/components/layout/WindowBoard.vue'
import Button from '@/components/ui/Button.vue'
import Dialog from '@/components/ui/Dialog.vue'
import Dropdown, { type DropdownItem } from '@/components/ui/Dropdown.vue'
import Input from '@/components/ui/Input.vue'
import Select from '@/components/ui/Select.vue'
import Tabs from '@/components/ui/Tabs.vue'
import Textarea from '@/components/ui/Textarea.vue'
import { useDensityStore } from '@/design-system/density'
import { useThemeStore } from '@/design-system/theme'
import type { DensityMode, SupportedLocale, ThemeMode } from '@/design-system/types'
import { useToast } from '@/composables/useToast'
import { computed, ref } from 'vue'
import { useI18n } from 'vue-i18n'

const { t, locale } = useI18n({ useScope: 'global' })
const { themeMode, setThemeMode } = useThemeStore()
const { densityMode, setDensityMode } = useDensityStore()
const { pushToast } = useToast()

const textValue = ref('')
const areaValue = ref('')
const dialogInput = ref('')
const dialogOpen = ref(false)
const activeTab = ref('controls')
const lastMenuAction = ref('No action')
const dialogTriggerRef = ref<{ focus: () => void } | null>(null)

const windowPanes = computed(() => [
  { id: 'preferences', title: 'Theme / Density / Locale' },
  { id: 'component-matrix', title: 'Component Matrix' },
])

const tabItems = [
  { id: 'controls', label: 'Controls' },
  { id: 'overlay', label: 'Dialog/Dropdown' },
  { id: 'loading', label: 'Skeleton' },
]

const themeModeModel = computed<ThemeMode>({
  get: () => themeMode.value,
  set: (value) => setThemeMode(value),
})

const densityModeModel = computed<DensityMode>({
  get: () => densityMode.value,
  set: (value) => setDensityMode(value),
})

const localeModel = computed<SupportedLocale>({
  get: () => locale.value as SupportedLocale,
  set: (value) => {
    locale.value = value
  },
})

const themeOptions = computed(() => [
  { value: 'system', label: t('common.system') },
  { value: 'light', label: t('common.light') },
  { value: 'dark', label: t('common.dark') },
])

const densityOptions = computed(() => [
  { value: 'compact', label: t('common.compact') },
  { value: 'comfortable', label: t('common.comfortable') },
])

const localeOptions = [
  { value: 'zh-CN', label: 'zh-CN' },
  { value: 'en-US', label: 'en-US' },
]

const menuItems: DropdownItem[] = [
  { label: 'Run Audit', value: 'audit', hint: 'A' },
  { label: 'Clear Cache', value: 'clear', hint: 'C' },
  { label: 'Delete', value: 'delete', danger: true },
]

function onSelectMenu(action: string): void {
  lastMenuAction.value = `Selected action: ${action}`
  pushToast({ title: 'Action', message: action, tone: 'info' })
}

function openDialog(): void {
  dialogOpen.value = true
}

function closeDialog(): void {
  dialogOpen.value = false
  window.setTimeout(() => {
    dialogTriggerRef.value?.focus()
  }, 180)
}

function onConfirmDialog(): void {
  closeDialog()
  pushToast({ title: 'Dialog', message: 'Confirmed', tone: 'success' })
}
</script>
