<template>
  <aside class="ui-surface flex h-full w-64 flex-col rounded-none border-y-0 border-l-0">
    <div class="border-b border-ui-border px-4 py-4">
      <p class="text-xs uppercase tracking-[0.14em] text-ui-muted">Workspace</p>
      <p class="mt-1 text-sm font-semibold text-ui-fg">{{ t('common.workspace') }}</p>
    </div>

    <nav class="ui-page p-3">
      <RouterLink
        v-for="item in navItems"
        :key="item.to"
        :to="item.to"
        class="ui-control ui-focus-ring ui-pressable flex items-center justify-between border-transparent text-sm font-medium"
        active-class="!border-primary-500 !bg-primary-500/10 !text-primary-700 dark:!text-primary-500"
      >
        <span>{{ t(item.label) }}</span>
        <span class="ui-monospace text-xs text-ui-muted">{{ item.shortcut }}</span>
      </RouterLink>
    </nav>
  </aside>
</template>

<script setup lang="ts">
import { useI18n } from 'vue-i18n'
import { RouterLink } from 'vue-router'

const { t } = useI18n({ useScope: 'global' })

const navItems = [
  { to: '/', label: 'nav.home', shortcut: '01' },
  { to: '/canvas', label: 'nav.canvas', shortcut: '02' },
  { to: '/commands', label: 'nav.commands', shortcut: '03' },
  { to: '/assets', label: 'nav.assets', shortcut: '04' },
  { to: '/plugins', label: 'nav.plugins', shortcut: '05' },
  { to: '/streams', label: 'nav.streams', shortcut: '06' },
  { to: '/settings', label: 'nav.settings', shortcut: '07' },
]
</script>
