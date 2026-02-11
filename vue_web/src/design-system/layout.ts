import { computed, ref, watch } from 'vue'
import type { Router, RouteLocationNormalizedLoaded } from 'vue-router'
import type { LayoutMode, LayoutPreference } from '@/design-system/types'

const STORAGE_KEY = 'goyais.ui.layout'

const layoutPreference = ref<LayoutPreference>('auto')
const routeLayoutDefault = ref<LayoutMode>('console')

const effectiveLayout = computed<LayoutMode>(() =>
  layoutPreference.value === 'auto' ? routeLayoutDefault.value : layoutPreference.value,
)

let initialized = false
let unwatchRouter: (() => void) | null = null

function isLayoutMode(value: string | null): value is LayoutMode {
  return value === 'console' || value === 'topnav' || value === 'focus'
}

function isLayoutPreference(value: string | null): value is LayoutPreference {
  return value === 'auto' || isLayoutMode(value)
}

function readLayoutPreference(): LayoutPreference {
  const stored = localStorage.getItem(STORAGE_KEY)
  return isLayoutPreference(stored) ? stored : 'auto'
}

function resolveRouteLayout(route: RouteLocationNormalizedLoaded | null): LayoutMode {
  const candidate = route?.meta?.layoutDefault
  if (typeof candidate === 'string' && isLayoutMode(candidate)) {
    return candidate
  }
  return 'console'
}

function applyLayoutAttributes(): void {
  document.documentElement.setAttribute('data-layout', effectiveLayout.value)
  document.documentElement.setAttribute('data-layout-pref', layoutPreference.value)
}

function bindRouter(router: Router): void {
  routeLayoutDefault.value = resolveRouteLayout(router.currentRoute.value)
  unwatchRouter?.()
  unwatchRouter = router.afterEach((to) => {
    routeLayoutDefault.value = resolveRouteLayout(to)
  })
}

export function initLayoutSystem(router?: Router): void {
  if (initialized) {
    if (router) {
      bindRouter(router)
    }
    return
  }

  initialized = true
  layoutPreference.value = readLayoutPreference()
  if (router) {
    bindRouter(router)
  }

  applyLayoutAttributes()

  watch(
    layoutPreference,
    (value) => {
      localStorage.setItem(STORAGE_KEY, value)
      applyLayoutAttributes()
    },
    { flush: 'post' },
  )

  watch(routeLayoutDefault, () => {
    applyLayoutAttributes()
  })
}

export function useLayoutStore() {
  return {
    layoutPreference,
    routeLayoutDefault,
    effectiveLayout,
    setLayoutPreference: (value: LayoutPreference) => {
      layoutPreference.value = value
    },
    setRouteLayoutDefault: (value: LayoutMode) => {
      routeLayoutDefault.value = value
    },
  }
}

export function layoutStorageKey(): string {
  return STORAGE_KEY
}

export function __resetLayoutSystemForTests(): void {
  unwatchRouter?.()
  unwatchRouter = null
  initialized = false
  layoutPreference.value = 'auto'
  routeLayoutDefault.value = 'console'
}
