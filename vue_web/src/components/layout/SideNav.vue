<template>
  <aside
    class="ui-surface hidden h-full flex-col rounded-none border-y-0 border-l-0 transition-[width] duration-150 lg:flex"
    :class="collapsed ? 'w-[4.75rem]' : 'w-64'"
    @mouseenter="hovering = true"
    @mouseleave="hovering = false"
  >
    <div class="flex h-[4.25rem] shrink-0 items-center justify-between gap-2 border-b border-ui-border px-3 py-2">
      <div
        class="min-w-0 flex-1"
        :class="collapsed
          ? 'pointer-events-none max-w-0 overflow-hidden opacity-0'
          : 'max-w-[11rem] opacity-100 transition-[opacity,max-width] duration-150'"
      >
        <p class="text-[11px] uppercase leading-none tracking-[0.14em] text-ui-muted">{{ t('common.workspaceLabel') }}</p>
        <p class="truncate text-sm font-semibold leading-tight text-ui-fg">{{ t('common.workspace') }}</p>
      </div>
      <button
        type="button"
        class="ui-control ui-focus-ring ui-pressable inline-flex h-8 w-10 min-h-0 shrink-0 items-center justify-center px-1 py-1 text-xs"
        :aria-label="pinned ? t('common.unpinNav') : t('common.pinNav')"
        :data-pinned="pinned ? 'true' : 'false'"
        @click="onTogglePinned"
      >
        {{ pinned ? t('common.unpinShort') : t('common.pinShort') }}
      </button>
    </div>

    <nav class="ui-page min-h-0 flex-1 overflow-auto p-3">
      <RouterLink
        v-for="item in NAV_ITEMS"
        :key="item.to"
        :to="item.to"
        class="ui-control ui-focus-ring ui-pressable flex items-center border-transparent text-sm font-medium"
        :class="collapsed ? 'justify-center px-1' : 'justify-between'"
        :title="collapsed ? t(item.label) : undefined"
        active-class="ui-nav-link-active"
      >
        <span class="flex min-w-0 items-center gap-2">
          <Icon :name="item.icon" :size="16" decorative class="opacity-90" />
          <span v-if="!collapsed" class="truncate">{{ t(item.label) }}</span>
        </span>
        <span v-if="!collapsed" class="ui-monospace text-xs text-ui-muted">{{ item.shortcut }}</span>
      </RouterLink>
    </nav>
  </aside>
</template>

<script setup lang="ts">
import Icon from '@/components/ui/Icon.vue'
import { NAV_ITEMS } from '@/design-system/navigation'
import { computed, onMounted, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import { RouterLink } from 'vue-router'

const { t } = useI18n({ useScope: 'global' })

const PINNED_STORAGE_KEY = 'goyais.ui.sidenav.pinned'

function readPinnedState(): boolean {
  try {
    return localStorage.getItem(PINNED_STORAGE_KEY) === 'true'
  } catch {
    return false
  }
}

function persistPinnedState(value: boolean): void {
  try {
    localStorage.setItem(PINNED_STORAGE_KEY, value ? 'true' : 'false')
  } catch {
    // Ignore storage failures (private mode / quota).
  }
}

const pinned = ref(false)
const hovering = ref(false)

const collapsed = computed(() => !pinned.value && !hovering.value)

onMounted(() => {
  pinned.value = readPinnedState()
})

watch(pinned, (value) => {
  persistPinnedState(value)
})

function onTogglePinned(): void {
  const nextPinned = !pinned.value
  pinned.value = nextPinned

  // When switching to floating mode, collapse immediately even if cursor stays inside sidenav.
  if (!nextPinned) {
    hovering.value = false
  }
}
</script>
