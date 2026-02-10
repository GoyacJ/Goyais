<template>
  <div class="ui-shell-root ui-bg-host flex h-full bg-ui-bg text-ui-fg" :class="shellClass">
    <MobileNavDrawer :open="mobileNavOpen" @close="closeMobileNav" />

    <SideNav
      v-if="showSideNav"
      class="ui-bg-content border-r border-ui-border"
    />

    <div class="ui-bg-content flex min-w-0 flex-1 flex-col">
      <TopBar
        :show-mobile-nav-button="!isDesktop"
        :focus-mode="effectiveLayout === 'focus'"
        @toggle-mobile-nav="openMobileNav"
      />

      <TopNavBar v-if="showTopNav" />

      <main class="ui-shell-main ui-scrollbar min-h-0 flex-1 overflow-auto p-[var(--ui-shell-content-pad)]">
        <RouterView />
      </main>
    </div>

    <ToastViewport />
  </div>
</template>

<script setup lang="ts">
import MobileNavDrawer from '@/components/layout/MobileNavDrawer.vue'
import SideNav from '@/components/layout/SideNav.vue'
import TopBar from '@/components/layout/TopBar.vue'
import TopNavBar from '@/components/layout/TopNavBar.vue'
import ToastViewport from '@/components/ui/ToastViewport.vue'
import { useLayoutStore } from '@/design-system/layout'
import { computed, nextTick, onBeforeUnmount, onMounted, ref } from 'vue'
import { RouterView } from 'vue-router'

const { effectiveLayout } = useLayoutStore()

const mobileNavOpen = ref(false)
const lastFocusedBeforeDrawer = ref<HTMLElement | null>(null)
const isDesktop = ref(true)
let mediaQuery: MediaQueryList | null = null

const showSideNav = computed(() => isDesktop.value && effectiveLayout.value === 'console')
const showTopNav = computed(() => isDesktop.value && effectiveLayout.value === 'topnav')

const shellClass = computed(() => {
  if (effectiveLayout.value === 'focus') {
    return 'ui-bg-gradient'
  }

  if (effectiveLayout.value === 'topnav') {
    return 'ui-bg-grid'
  }

  return 'ui-bg-stack-console'
})

function syncDesktopState(): void {
  isDesktop.value = mediaQuery ? mediaQuery.matches : true
  if (isDesktop.value) {
    mobileNavOpen.value = false
  }
}

function openMobileNav(): void {
  lastFocusedBeforeDrawer.value = document.activeElement as HTMLElement | null
  mobileNavOpen.value = true
}

function closeMobileNav(): void {
  mobileNavOpen.value = false
  const target = lastFocusedBeforeDrawer.value
  lastFocusedBeforeDrawer.value = null
  if (!target) {
    return
  }
  nextTick(() => {
    target.focus()
  })
}

onMounted(() => {
  mediaQuery = window.matchMedia('(min-width: 1024px)')
  mediaQuery.addEventListener('change', syncDesktopState)
  syncDesktopState()
})

onBeforeUnmount(() => {
  mediaQuery?.removeEventListener('change', syncDesktopState)
})
</script>
