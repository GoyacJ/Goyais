<template>
  <section class="ui-window-board-wrap">
    <div
      v-if="isImmersive"
      ref="immersiveRef"
      class="ui-window-immersive ui-bg-host rounded-card border border-ui-border"
    >
      <div class="ui-window-immersive-toolbar ui-bg-content">
        <div class="min-w-0">
          <p class="truncate text-xs font-semibold uppercase tracking-[0.08em] text-ui-muted">
            {{ immersivePane?.title ?? immersivePaneId }}
          </p>
          <p class="ui-monospace truncate text-[11px] text-ui-muted">{{ immersivePaneId }}</p>
        </div>
        <div class="ui-window-immersive-actions">
          <button
            type="button"
            class="ui-control ui-focus-ring ui-pressable h-8 min-h-0 px-2 py-1 text-xs"
            :disabled="!canUseFullscreenApi"
            @click="onToggleImmersiveFullscreen"
          >
            {{ immersiveFullscreen ? t('common.exitFullscreen') : t('common.enterFullscreen') }}
          </button>
          <button
            type="button"
            class="ui-control ui-focus-ring ui-pressable h-8 min-h-0 px-2 py-1 text-xs"
            @click="onExitImmersive"
          >
            {{ t('common.returnToBoard') }}
          </button>
        </div>
      </div>
      <section class="ui-window-immersive-body">
        <slot :name="immersivePaneId ?? ''" />
      </section>
    </div>

    <div
      v-else-if="windowedEnabled"
      ref="boardRef"
      class="ui-window-board ui-bg-host rounded-card border border-ui-border"
      :class="[`ui-window-board--${effectiveLayout}`]"
      :style="{ height: `${boardHeight}px` }"
    >
      <div class="ui-window-board-toolbar ui-bg-content absolute right-3 top-3 z-50">
        <button
          type="button"
          class="ui-control ui-focus-ring ui-pressable h-8 min-h-0 px-2 py-1 text-xs"
          @click="onResetLayout"
        >
          {{ t('common.resetWindowLayout') }}
        </button>
      </div>

      <WindowPane
        v-for="pane in resolvedPanes"
        :key="pane.id"
        :pane-id="pane.id"
        :title="pane.title"
        :rect="paneRects[pane.id]"
        :bounds="bounds"
        :min-width="pane.minWidth"
        :min-height="pane.minHeight"
        @focus="bringToFront"
        @open-new-page="onOpenNewPage"
        @update:rect="onRectUpdate"
      >
        <slot :name="pane.id" />
      </WindowPane>
    </div>

    <div v-else class="ui-window-flow ui-page">
      <section v-for="pane in resolvedPanes" :key="pane.id" class="ui-window-flow-pane">
        <slot :name="pane.id" />
      </section>
    </div>
  </section>
</template>

<script setup lang="ts">
import WindowPane from '@/components/layout/WindowPane.vue'
import { useLayoutStore } from '@/design-system/layout'
import { fitManifestToBoardWidth } from '@/design-system/window-fit'
import { windowManifestFor, type WindowPaneManifest } from '@/design-system/window-manifests'
import { useWindowLayout } from '@/design-system/window-layout'
import type { WindowRect } from '@/design-system/types'
import type { LocationQueryRaw } from 'vue-router'
import { computed, onBeforeUnmount, onMounted, ref, watch } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { useI18n } from 'vue-i18n'

interface WindowPaneDefinition {
  id: string
  title: string
  minWidth?: number
  minHeight?: number
}

interface WindowPaneResolved extends WindowPaneDefinition {
  x: number
  y: number
  w: number
  h: number
}

const props = withDefaults(
  defineProps<{
    routeKey: string
    panes: WindowPaneDefinition[]
    windowed?: boolean
  }>(),
  {
    windowed: true,
  },
)

const { t } = useI18n({ useScope: 'global' })
const { effectiveLayout } = useLayoutStore()
const route = useRoute()
const router = useRouter()

const boardRef = ref<HTMLElement | null>(null)
const immersiveRef = ref<HTMLElement | null>(null)
const boardWidth = ref(1200)
const isDesktop = ref(true)
const immersiveFullscreen = ref(false)
const BOARD_SIDE_PADDING = 24
let mediaQuery: MediaQueryList | null = null
let resizeObserver: ResizeObserver | null = null

const resolvedPanes = computed<WindowPaneResolved[]>(() => {
  const manifest = new Map(windowManifestFor(props.routeKey).map((item) => [item.id, item]))

  return props.panes.map((pane, index) => {
    const fallback = fallbackManifest(index, pane.id)
    const m = manifest.get(pane.id) ?? fallback

    return {
      ...pane,
      x: m.x,
      y: m.y,
      w: m.w,
      h: m.h,
      minWidth: pane.minWidth ?? m.minWidth ?? 300,
      minHeight: pane.minHeight ?? m.minHeight ?? 200,
    }
  })
})

const baseManifestForStore = computed<WindowPaneManifest[]>(() =>
  resolvedPanes.value.map((pane) => ({
    id: pane.id,
    x: pane.x,
    y: pane.y,
    w: pane.w,
    h: pane.h,
    minWidth: pane.minWidth,
    minHeight: pane.minHeight,
  })),
)

const { panes, bringToFront, updatePaneRect, resetLayout, flushPersist, storageKey } = useWindowLayout(
  computed(() => props.routeKey),
  effectiveLayout,
  baseManifestForStore,
)

const windowedEnabled = computed(() => props.windowed && isDesktop.value)
const fittedManifestForBoard = computed<WindowPaneManifest[]>(() =>
  fitManifestToBoardWidth(baseManifestForStore.value, boardWidth.value, BOARD_SIDE_PADDING),
)

const immersivePaneId = computed(() => {
  const mode = readQueryString(route.query.wbMode)
  const pane = readQueryString(route.query.wbPane)
  if (mode !== 'immersive' || !pane) {
    return null
  }
  if (!resolvedPanes.value.some((item) => item.id === pane)) {
    return null
  }
  return pane
})

const isImmersive = computed(() => immersivePaneId.value !== null)
const immersivePane = computed(() => {
  if (!immersivePaneId.value) {
    return null
  }
  return resolvedPanes.value.find((item) => item.id === immersivePaneId.value) ?? null
})

const canUseFullscreenApi = computed(
  () => typeof document !== 'undefined' && typeof document.exitFullscreen === 'function',
)

const paneRects = computed<Record<string, WindowRect>>(() => {
  const fittedMap = new Map(fittedManifestForBoard.value.map((pane) => [pane.id, pane]))
  const result: Record<string, WindowRect> = {}
  resolvedPanes.value.forEach((pane, index) => {
    const fallback = fittedMap.get(pane.id) ?? pane
    result[pane.id] = panes.value[pane.id] ?? {
      x: fallback.x,
      y: fallback.y,
      w: fallback.w,
      h: fallback.h,
      z: index + 1,
    }
  })
  return result
})

const boardHeight = computed(() => {
  if (!windowedEnabled.value) {
    return 0
  }
  const maxBottom = Object.values(paneRects.value).reduce((acc, pane) => Math.max(acc, pane.y + pane.h), 0)
  return Math.max(560, maxBottom + 32)
})

const bounds = computed(() => ({
  width: Math.max(640, boardWidth.value),
  height: Math.max(560, boardHeight.value),
}))

function fallbackManifest(index: number, paneId: string): WindowPaneManifest {
  const column = index % 2
  const row = Math.floor(index / 2)
  return {
    id: paneId,
    x: 24 + column * 520,
    y: 24 + row * 280,
    w: 500,
    h: 260,
  }
}

function onRectUpdate(payload: { paneId: string; rect: WindowRect }): void {
  updatePaneRect(payload.paneId, payload.rect)
}

function fitCurrentLayoutToBoard(): void {
  if (!windowedEnabled.value || isImmersive.value) {
    return
  }

  const layout = resolvedPanes.value
    .map((pane) => ({
      id: pane.id,
      minWidth: pane.minWidth ?? 300,
      rect: paneRects.value[pane.id],
    }))
    .filter((item): item is { id: string; minWidth: number; rect: WindowRect } => item.rect !== undefined)

  if (layout.length === 0) {
    return
  }

  const minX = Math.min(...layout.map((item) => item.rect.x))
  const maxRight = Math.max(...layout.map((item) => item.rect.x + item.rect.w))
  const sourceWidth = Math.max(1, maxRight - minX)
  const targetWidth = Math.max(320, boardWidth.value - BOARD_SIDE_PADDING * 2)
  const scale = targetWidth / sourceWidth
  const rightLimit = Math.max(BOARD_SIDE_PADDING, boardWidth.value - BOARD_SIDE_PADDING)

  layout.forEach((item) => {
    const nextWidth = Math.max(item.minWidth, Math.round(item.rect.w * scale))
    const projectedX = BOARD_SIDE_PADDING + Math.round((item.rect.x - minX) * scale)
    const maxX = Math.max(BOARD_SIDE_PADDING, rightLimit - nextWidth)
    const nextX = Math.min(maxX, Math.max(BOARD_SIDE_PADDING, projectedX))

    if (nextX === item.rect.x && nextWidth === item.rect.w) {
      return
    }

    updatePaneRect(item.id, {
      ...item.rect,
      x: nextX,
      w: nextWidth,
    })
  })
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

function hasPersistedLayout(): boolean {
  try {
    return localStorage.getItem(storageKey.value) !== null
  } catch {
    return false
  }
}

function onResetLayout(): void {
  resetLayout(fittedManifestForBoard.value)
}

function onOpenNewPage(paneId: string): void {
  if (paneId.trim().length === 0 || typeof window === 'undefined') {
    return
  }
  const query: LocationQueryRaw = {
    ...route.query,
    wbMode: 'immersive',
    wbPane: paneId,
  }
  const href = router.resolve({ path: route.path, query }).href
  window.open(href, '_blank', 'noopener,noreferrer')
}

function onExitImmersive(): void {
  const nextQuery: LocationQueryRaw = { ...route.query }
  delete (nextQuery as Record<string, unknown>).wbMode
  delete (nextQuery as Record<string, unknown>).wbPane
  void router.replace({ path: route.path, query: nextQuery })
}

function syncImmersiveFullscreen(): void {
  immersiveFullscreen.value = document.fullscreenElement === immersiveRef.value
}

async function onToggleImmersiveFullscreen(): Promise<void> {
  if (!canUseFullscreenApi.value) {
    return
  }
  try {
    if (document.fullscreenElement === immersiveRef.value) {
      await document.exitFullscreen()
      return
    }
    const node = immersiveRef.value
    if (!node || typeof node.requestFullscreen !== 'function') {
      return
    }
    await node.requestFullscreen()
  } catch {
    // Ignore fullscreen API errors in unsupported contexts.
  }
}

function updateDesktopFlag(): void {
  isDesktop.value = mediaQuery ? mediaQuery.matches : true
}

function measureBoard(): void {
  boardWidth.value = boardRef.value?.clientWidth ?? 1200
}

function onBeforeUnload(): void {
  flushPersist(true)
}

onMounted(() => {
  mediaQuery = window.matchMedia('(min-width: 1024px)')
  mediaQuery.addEventListener('change', updateDesktopFlag)
  updateDesktopFlag()

  measureBoard()
  if (!hasPersistedLayout()) {
    onResetLayout()
  } else {
    fitCurrentLayoutToBoard()
  }
  if (typeof ResizeObserver !== 'undefined') {
    resizeObserver = new ResizeObserver(() => measureBoard())
    if (boardRef.value) {
      resizeObserver.observe(boardRef.value)
    }
  } else {
    window.addEventListener('resize', measureBoard)
  }
  window.addEventListener('beforeunload', onBeforeUnload)
  document.addEventListener('fullscreenchange', syncImmersiveFullscreen)
})

onBeforeUnmount(() => {
  flushPersist(true)
  mediaQuery?.removeEventListener('change', updateDesktopFlag)
  resizeObserver?.disconnect()
  window.removeEventListener('resize', measureBoard)
  window.removeEventListener('beforeunload', onBeforeUnload)
  document.removeEventListener('fullscreenchange', syncImmersiveFullscreen)
})

watch(boardWidth, (next, prev) => {
  if (typeof prev === 'number' && Math.abs(next - prev) < 2) {
    return
  }
  fitCurrentLayoutToBoard()
})
</script>
