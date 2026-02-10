<template>
  <section class="ui-window-board-wrap">
    <div
      v-if="windowedEnabled"
      ref="boardRef"
      class="ui-window-board ui-bg-host rounded-card border border-ui-border"
      :class="[`ui-window-board--${effectiveLayout}`]"
      :style="{ height: `${boardHeight}px` }"
    >
      <div class="ui-window-board-toolbar ui-bg-content absolute right-3 top-3 z-50">
        <button
          type="button"
          class="ui-control ui-focus-ring ui-pressable h-8 min-h-0 px-2 py-1 text-xs"
          @click="resetLayout"
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
import { windowManifestFor, type WindowPaneManifest } from '@/design-system/window-manifests'
import { useWindowLayout } from '@/design-system/window-layout'
import type { WindowRect } from '@/design-system/types'
import { computed, onBeforeUnmount, onMounted, ref } from 'vue'
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

const boardRef = ref<HTMLElement | null>(null)
const boardWidth = ref(1200)
const isDesktop = ref(true)
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

const manifestForStore = computed<WindowPaneManifest[]>(() =>
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

const { panes, bringToFront, updatePaneRect, resetLayout } = useWindowLayout(
  computed(() => props.routeKey),
  effectiveLayout,
  manifestForStore,
)

const windowedEnabled = computed(() => props.windowed && isDesktop.value)

const paneRects = computed<Record<string, WindowRect>>(() => {
  const result: Record<string, WindowRect> = {}
  resolvedPanes.value.forEach((pane, index) => {
    result[pane.id] = panes.value[pane.id] ?? {
      x: pane.x,
      y: pane.y,
      w: pane.w,
      h: pane.h,
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

function updateDesktopFlag(): void {
  isDesktop.value = mediaQuery ? mediaQuery.matches : true
}

function measureBoard(): void {
  boardWidth.value = boardRef.value?.clientWidth ?? 1200
}

onMounted(() => {
  mediaQuery = window.matchMedia('(min-width: 1024px)')
  mediaQuery.addEventListener('change', updateDesktopFlag)
  updateDesktopFlag()

  measureBoard()
  if (typeof ResizeObserver !== 'undefined') {
    resizeObserver = new ResizeObserver(() => measureBoard())
    if (boardRef.value) {
      resizeObserver.observe(boardRef.value)
    }
  } else {
    window.addEventListener('resize', measureBoard)
  }
})

onBeforeUnmount(() => {
  mediaQuery?.removeEventListener('change', updateDesktopFlag)
  resizeObserver?.disconnect()
  window.removeEventListener('resize', measureBoard)
})
</script>
