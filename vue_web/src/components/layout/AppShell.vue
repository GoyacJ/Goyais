<template>
  <div class="ui-shell-root ui-bg-host flex h-full bg-ui-bg text-ui-fg" :class="shellClass">
    <MobileNavDrawer v-if="!isImmersiveRoute" :open="mobileNavOpen" @close="closeMobileNav" />

    <SideNav
      v-if="showSideNav"
      class="ui-bg-content border-r border-ui-border"
    />

    <div class="ui-bg-content flex min-w-0 flex-1 flex-col">
      <TopBar
        v-if="!isImmersiveRoute"
        :show-mobile-nav-button="!isDesktop"
        @toggle-mobile-nav="openMobileNav"
      />
      <RouteTabBar v-if="!isImmersiveRoute" />

      <main :class="mainClass">
        <RouterView />
      </main>
    </div>

    <ToastViewport />
  </div>
</template>

<script setup lang="ts">
/**
 * SPDX-License-Identifier: Apache-2.0
 * Copyright (c) 2026 Goya
 * Author: Goya
 * Created: 2026-02-11
 * Version: v1.0.0
 * Description: Goyais source file.
 */
import MobileNavDrawer from '@/components/layout/MobileNavDrawer.vue'
import RouteTabBar from '@/components/layout/RouteTabBar.vue'
import SideNav from '@/components/layout/SideNav.vue'
import TopBar from '@/components/layout/TopBar.vue'
import ToastViewport from '@/components/ui/ToastViewport.vue'
import { windowManifestFor } from '@/design-system/window-manifests'
import { computed, nextTick, onBeforeUnmount, onMounted, ref } from 'vue'
import { RouterView, useRoute } from 'vue-router'

const route = useRoute()

const mobileNavOpen = ref(false)
const lastFocusedBeforeDrawer = ref<HTMLElement | null>(null)
const isDesktop = ref(true)
let mediaQuery: MediaQueryList | null = null

const isImmersiveRoute = computed(() => {
  const mode = readQueryString(route.query.wbMode)
  const paneId = readQueryString(route.query.wbPane)
  if (mode !== 'immersive' || !paneId) {
    return false
  }
  const key = typeof route.meta?.windowManifestKey === 'string' ? route.meta.windowManifestKey : null
  if (!key) {
    return false
  }
  return windowManifestFor(key).some((pane) => pane.id === paneId)
})

const showSideNav = computed(() => !isImmersiveRoute.value && isDesktop.value)

const mainClass = computed(() => [
  'ui-shell-main ui-scrollbar min-h-0 flex-1 overflow-auto',
  isImmersiveRoute.value ? 'ui-shell-main--immersive' : 'p-[var(--ui-shell-content-pad)]',
])

const shellClass = computed(() => 'ui-bg-stack-console')

function syncDesktopState(): void {
  isDesktop.value = mediaQuery ? mediaQuery.matches : true
  if (isDesktop.value) {
    mobileNavOpen.value = false
  }
}

function readQueryString(value: unknown): string | null {
  if (typeof value === 'string') {
    const next = value.trim()
    return next.length > 0 ? next : null
  }
  if (Array.isArray(value)) {
    const first = value.find((item) => typeof item === 'string')
    if (typeof first === 'string') {
      const next = first.trim()
      return next.length > 0 ? next : null
    }
  }
  return null
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
