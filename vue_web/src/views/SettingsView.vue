<template>
  <section class="ui-page">
    <PageHeader :title="t('page.settings.title')" :subtitle="t('page.settings.subtitle')" />

    <WindowBoard route-key="settings" :panes="windowPanes">
      <template #preferences>
        <SectionCard :title="t('page.settings.sections.preferences.title')" :subtitle="t('page.settings.sections.preferences.subtitle')">
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
        <SectionCard :title="t('page.settings.sections.matrix.title')" :subtitle="t('page.settings.sections.matrix.subtitle')">
          <Tabs v-model="activeTab" :tabs="tabItems" :aria-label="t('page.settings.sections.matrix.ariaLabel')" />

          <div class="mt-3">
            <template v-if="activeTab === 'controls'">
              <div class="grid gap-3 md:grid-cols-2">
                <Input v-model="textValue" :placeholder="t('page.settings.controls.inputPlaceholder')" />
                <Textarea v-model="areaValue" :placeholder="t('page.settings.controls.textareaPlaceholder')" />
              </div>
              <div class="mt-3 flex flex-wrap gap-2">
                <Button>{{ t('common.primary') }}</Button>
                <Button variant="secondary">{{ t('common.secondary') }}</Button>
                <Button variant="ghost">{{ t('common.ghost') }}</Button>
                <Button variant="destructive">{{ t('common.destructive') }}</Button>
                <Button loading>{{ t('common.loading') }}</Button>
                <Button disabled>{{ t('common.disabled') }}</Button>
              </div>
            </template>

            <template v-else-if="activeTab === 'overlay'">
              <div class="flex flex-wrap gap-2">
                <Button ref="dialogTriggerRef" @click="openDialog">{{ t('page.settings.dialog.open') }}</Button>
                <Dropdown :items="menuItems" :label="t('page.settings.menu.openDropdown')" @select="onSelectMenu" />
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
      :title="t('page.settings.dialog.title')"
      :description="t('page.settings.dialog.description')"
      :cancel-label="t('common.close')"
      :confirm-label="t('common.confirm')"
      @close="closeDialog"
      @confirm="onConfirmDialog"
    >
      <div class="space-y-3">
        <Input v-model="dialogInput" :placeholder="t('page.settings.dialog.inputPlaceholder')" />
        <Button variant="secondary">{{ t('page.settings.dialog.focusableAction') }}</Button>
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
const lastMenuAction = ref(t('page.settings.menu.noAction'))
const dialogTriggerRef = ref<{ focus: () => void } | null>(null)

const windowPanes = computed(() => [
  { id: 'preferences', title: t('page.settings.sections.preferences.title') },
  { id: 'component-matrix', title: t('page.settings.sections.matrix.title') },
])

const tabItems = computed(() => [
  { id: 'controls', label: t('page.settings.tabs.controls') },
  { id: 'overlay', label: t('page.settings.tabs.overlay') },
  { id: 'loading', label: t('page.settings.tabs.loading') },
])

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

const menuItems = computed<DropdownItem[]>(() => [
  { label: t('page.settings.menu.runAudit'), value: 'audit', hint: 'A' },
  { label: t('page.settings.menu.clearCache'), value: 'clear', hint: 'C' },
  { label: t('page.settings.menu.delete'), value: 'delete', danger: true },
])

function onSelectMenu(action: string): void {
  lastMenuAction.value = t('page.settings.menu.selectedAction', { action })
  pushToast({ title: t('page.settings.menu.toastTitle'), message: action, tone: 'info' })
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
  pushToast({ title: t('page.settings.dialog.toastTitle'), message: t('page.settings.dialog.toastMessage'), tone: 'success' })
}
</script>
