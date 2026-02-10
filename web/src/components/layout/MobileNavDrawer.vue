<template>
  <transition
    enter-active-class="transition-opacity duration-120"
    enter-from-class="opacity-0"
    enter-to-class="opacity-100"
    leave-active-class="transition-opacity duration-90"
    leave-from-class="opacity-100"
    leave-to-class="opacity-0"
  >
    <div
      v-if="open"
      class="ui-overlay-backdrop fixed inset-0 z-40 lg:hidden"
      role="presentation"
      @click="$emit('close')"
    />
  </transition>

  <transition
    enter-active-class="transition-transform duration-150 ease-out"
    enter-from-class="-translate-x-full"
    enter-to-class="translate-x-0"
    leave-active-class="transition-transform duration-120 ease-in"
    leave-from-class="translate-x-0"
    leave-to-class="-translate-x-full"
  >
    <aside
      v-if="open"
      class="ui-surface fixed inset-y-0 left-0 z-50 flex w-72 flex-col rounded-none border-y-0 border-l-0 lg:hidden"
      aria-label="mobile-navigation"
    >
      <header class="flex items-center justify-between border-b border-ui-border px-4 py-3">
        <p class="text-sm font-semibold">{{ t('common.appName') }}</p>
        <button
          type="button"
          class="ui-control ui-focus-ring ui-pressable h-8 min-h-0 px-2 py-1 text-xs"
          @click="$emit('close')"
        >
          {{ t('common.close') }}
        </button>
      </header>

      <nav class="ui-page p-3">
        <RouterLink
          v-for="item in NAV_ITEMS"
          :key="item.to"
          :to="item.to"
          class="ui-control ui-focus-ring ui-pressable flex items-center justify-between border-transparent text-sm"
          active-class="!border-primary-500 !bg-primary-500/10 !text-primary-700 dark:!text-primary-500"
          @click="$emit('close')"
        >
          <span class="flex min-w-0 items-center gap-2">
            <Icon :name="item.icon" :size="16" decorative />
            <span class="truncate">{{ t(item.label) }}</span>
          </span>
          <span class="ui-monospace text-xs text-ui-muted">{{ item.shortcut }}</span>
        </RouterLink>
      </nav>
    </aside>
  </transition>
</template>

<script setup lang="ts">
import Icon from '@/components/ui/Icon.vue'
import { NAV_ITEMS } from '@/design-system/navigation'
import { useI18n } from 'vue-i18n'
import { RouterLink } from 'vue-router'

defineProps<{
  open: boolean
}>()

defineEmits<{
  (e: 'close'): void
}>()

const { t } = useI18n({ useScope: 'global' })
</script>
