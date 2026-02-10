<template>
  <aside
    class="ui-surface hidden h-full flex-col rounded-none border-y-0 border-l-0 transition-[width] duration-150 lg:flex"
    :class="collapsed ? 'w-[4.75rem]' : 'w-64'"
    @mouseenter="hovering = true"
    @mouseleave="hovering = false"
  >
    <div class="flex items-center justify-between border-b border-ui-border px-3 py-3">
      <div class="min-w-0" :class="collapsed ? 'opacity-0' : 'opacity-100 transition-opacity'">
        <p class="text-[11px] uppercase tracking-[0.14em] text-ui-muted">Workspace</p>
        <p class="truncate text-sm font-semibold text-ui-fg">{{ t('common.workspace') }}</p>
      </div>
      <button
        type="button"
        class="ui-control ui-focus-ring ui-pressable h-8 min-h-0 px-2 py-1 text-xs"
        :aria-label="pinned ? t('common.unpinNav') : t('common.pinNav')"
        @click="pinned = !pinned"
      >
        {{ pinned ? 'PIN' : 'UNP' }}
      </button>
    </div>

    <nav class="ui-page p-3">
      <RouterLink
        v-for="item in navItems"
        :key="item.to"
        :to="item.to"
        class="ui-control ui-focus-ring ui-pressable flex items-center border-transparent text-sm font-medium"
        :class="collapsed ? 'justify-center px-1' : 'justify-between'"
        :title="collapsed ? t(item.label) : undefined"
        active-class="!border-primary-500 !bg-primary-500/10 !text-primary-700 dark:!text-primary-500"
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
import { useDensityStore } from '@/design-system/density'
import type { IconName } from '@/design-system/icon-registry'
import { computed, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import { RouterLink } from 'vue-router'

const { t } = useI18n({ useScope: 'global' })
const { densityMode } = useDensityStore()

const pinned = ref(false)
const hovering = ref(false)

const collapsed = computed(() => densityMode.value === 'compact' && !pinned.value && !hovering.value)

const navItems: Array<{ to: string; label: string; shortcut: string; icon: IconName }> = [
  { to: '/', label: 'nav.home', shortcut: '01', icon: 'home' },
  { to: '/canvas', label: 'nav.canvas', shortcut: '02', icon: 'canvas' },
  { to: '/commands', label: 'nav.commands', shortcut: '03', icon: 'commands' },
  { to: '/assets', label: 'nav.assets', shortcut: '04', icon: 'assets' },
  { to: '/plugins', label: 'nav.plugins', shortcut: '05', icon: 'plugins' },
  { to: '/streams', label: 'nav.streams', shortcut: '06', icon: 'streams' },
  { to: '/settings', label: 'nav.settings', shortcut: '07', icon: 'settings' },
]
</script>
