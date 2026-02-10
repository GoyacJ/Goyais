<template>
  <article
    ref="paneRef"
    class="ui-window-pane ui-surface absolute flex min-h-0 flex-col overflow-hidden"
    :style="paneStyle"
    @pointerdown="emit('focus', paneId)"
  >
    <header
      class="ui-window-pane-header ui-bg-content flex h-[var(--ui-window-header-h)] cursor-move items-center justify-between border-b border-ui-border px-3"
      tabindex="0"
      role="button"
      :aria-label="t('common.windowMoveHandle', { title })"
      aria-keyshortcuts="Alt+ArrowLeft Alt+ArrowRight Alt+ArrowUp Alt+ArrowDown Alt+Shift+ArrowLeft Alt+Shift+ArrowRight Alt+Shift+ArrowUp Alt+Shift+ArrowDown"
      @pointerdown="onDragStart"
      @keydown="onHeaderKeydown"
    >
      <p class="truncate text-xs font-semibold uppercase tracking-[0.08em] text-ui-muted">{{ title }}</p>
      <span class="ui-monospace text-[11px] text-ui-muted">{{ paneId }}</span>
    </header>

    <div class="ui-window-pane-body ui-scrollbar min-h-0 flex-1 overflow-auto p-1">
      <slot />
    </div>

    <button
      type="button"
      :aria-label="t('common.windowResizeRight', { title })"
      class="ui-window-resize-right"
      data-testid="resize-right"
      @pointerdown="onResizeStart($event, 'right')"
    />
    <button
      type="button"
      :aria-label="t('common.windowResizeBottom', { title })"
      class="ui-window-resize-bottom"
      data-testid="resize-bottom"
      @pointerdown="onResizeStart($event, 'bottom')"
    />
    <button
      type="button"
      :aria-label="t('common.windowResizeCorner', { title })"
      class="ui-window-resize-corner"
      data-testid="resize-corner"
      @pointerdown="onResizeStart($event, 'corner')"
    />
  </article>
</template>

<script setup lang="ts">
import type {
  DragStartPayload,
  KeyboardDirection,
  KeyboardMovePayload,
  KeyboardResizePayload,
  ResizeDirection,
  ResizeStartPayload,
  WindowBounds,
} from '@/design-system/window-engine'
import { NativePointerWindowEngine } from '@/design-system/window-engine-native'
import type { WindowRect } from '@/design-system/types'
import { computed, onBeforeUnmount, ref } from 'vue'
import { useI18n } from 'vue-i18n'

const props = withDefaults(
  defineProps<{
    paneId: string
    title: string
    rect: WindowRect
    bounds: WindowBounds
    minWidth?: number
    minHeight?: number
  }>(),
  {
    minWidth: 260,
    minHeight: 180,
  },
)

const emit = defineEmits<{
  (e: 'focus', paneId: string): void
  (e: 'update:rect', payload: { paneId: string; rect: WindowRect }): void
}>()

const engine = new NativePointerWindowEngine()
const { t } = useI18n({ useScope: 'global' })
const KEYBOARD_STEP = 16
const AUTO_SCROLL_EDGE = 48
const AUTO_SCROLL_MAX_STEP = 24

const paneStyle = computed(() => ({
  left: `${props.rect.x}px`,
  top: `${props.rect.y}px`,
  width: `${props.rect.w}px`,
  height: `${props.rect.h}px`,
  zIndex: String(props.rect.z),
}))

const paneRef = ref<HTMLElement | null>(null)

let activeMoveHandler: ((event: PointerEvent) => void) | null = null
let activeUpHandler: ((event: PointerEvent) => void) | null = null

interface AutoScrollContext {
  container: HTMLElement | null
  startScrollTop: number
  startScrollLeft: number
}

function isScrollableOverflow(value: string): boolean {
  return value === 'auto' || value === 'scroll' || value === 'overlay'
}

function findScrollContainer(element: HTMLElement | null): HTMLElement | null {
  let current = element?.parentElement ?? null

  while (current) {
    const style = window.getComputedStyle(current)
    const canScrollY = isScrollableOverflow(style.overflowY) && current.scrollHeight > current.clientHeight
    const canScrollX = isScrollableOverflow(style.overflowX) && current.scrollWidth > current.clientWidth

    if (canScrollX || canScrollY) {
      return current
    }

    current = current.parentElement
  }

  const pageScroll = document.scrollingElement
  return pageScroll instanceof HTMLElement ? pageScroll : null
}

function createAutoScrollContext(): AutoScrollContext {
  const container = findScrollContainer(paneRef.value)

  return {
    container,
    startScrollTop: container?.scrollTop ?? 0,
    startScrollLeft: container?.scrollLeft ?? 0,
  }
}

function clamp(value: number, min: number, max: number): number {
  return Math.min(max, Math.max(min, value))
}

function autoScrollIfNeeded(context: AutoScrollContext, clientX: number, clientY: number): { offsetX: number; offsetY: number } {
  const container = context.container
  if (!container) {
    return { offsetX: 0, offsetY: 0 }
  }

  const rect = container.getBoundingClientRect()
  let stepY = 0
  let stepX = 0

  if (clientY > rect.bottom - AUTO_SCROLL_EDGE) {
    const ratio = (clientY - (rect.bottom - AUTO_SCROLL_EDGE)) / AUTO_SCROLL_EDGE
    stepY = Math.ceil(clamp(ratio, 0, 1) * AUTO_SCROLL_MAX_STEP)
  } else if (clientY < rect.top + AUTO_SCROLL_EDGE) {
    const ratio = ((rect.top + AUTO_SCROLL_EDGE) - clientY) / AUTO_SCROLL_EDGE
    stepY = -Math.ceil(clamp(ratio, 0, 1) * AUTO_SCROLL_MAX_STEP)
  }

  if (clientX > rect.right - AUTO_SCROLL_EDGE) {
    const ratio = (clientX - (rect.right - AUTO_SCROLL_EDGE)) / AUTO_SCROLL_EDGE
    stepX = Math.ceil(clamp(ratio, 0, 1) * AUTO_SCROLL_MAX_STEP)
  } else if (clientX < rect.left + AUTO_SCROLL_EDGE) {
    const ratio = ((rect.left + AUTO_SCROLL_EDGE) - clientX) / AUTO_SCROLL_EDGE
    stepX = -Math.ceil(clamp(ratio, 0, 1) * AUTO_SCROLL_MAX_STEP)
  }

  if (stepY !== 0) {
    const maxTop = Math.max(0, container.scrollHeight - container.clientHeight)
    container.scrollTop = clamp(container.scrollTop + stepY, 0, maxTop)
  }

  if (stepX !== 0) {
    const maxLeft = Math.max(0, container.scrollWidth - container.clientWidth)
    container.scrollLeft = clamp(container.scrollLeft + stepX, 0, maxLeft)
  }

  return {
    offsetX: container.scrollLeft - context.startScrollLeft,
    offsetY: container.scrollTop - context.startScrollTop,
  }
}

function cleanupPointerTracking(): void {
  if (activeMoveHandler) {
    window.removeEventListener('pointermove', activeMoveHandler)
    activeMoveHandler = null
  }
  if (activeUpHandler) {
    window.removeEventListener('pointerup', activeUpHandler)
    window.removeEventListener('pointercancel', activeUpHandler)
    activeUpHandler = null
  }
}

function trackPointer(onMove: (event: PointerEvent) => void): void {
  cleanupPointerTracking()
  activeMoveHandler = onMove
  activeUpHandler = () => {
    cleanupPointerTracking()
  }
  window.addEventListener('pointermove', activeMoveHandler)
  window.addEventListener('pointerup', activeUpHandler)
  window.addEventListener('pointercancel', activeUpHandler)
}

function onDragStart(event: PointerEvent): void {
  if (event.button !== 0) {
    return
  }
  event.preventDefault()
  event.stopPropagation()
  emit('focus', props.paneId)

  const payload: DragStartPayload = {
    startClientX: event.clientX,
    startClientY: event.clientY,
    startRect: { ...props.rect },
    bounds: props.bounds,
  }
  const autoScrollContext = createAutoScrollContext()

  trackPointer((moveEvent) => {
    const scrollOffset = autoScrollIfNeeded(autoScrollContext, moveEvent.clientX, moveEvent.clientY)
    const next = engine.projectDrag(
      payload,
      moveEvent.clientX + scrollOffset.offsetX,
      moveEvent.clientY + scrollOffset.offsetY,
    )
    emit('update:rect', { paneId: props.paneId, rect: next })
  })
}

function onResizeStart(event: PointerEvent, direction: ResizeDirection): void {
  if (event.button !== 0) {
    return
  }
  event.preventDefault()
  event.stopPropagation()
  emit('focus', props.paneId)

  const payload: ResizeStartPayload = {
    startClientX: event.clientX,
    startClientY: event.clientY,
    startRect: { ...props.rect },
    bounds: props.bounds,
    direction,
    minWidth: props.minWidth,
    minHeight: props.minHeight,
  }
  const autoScrollContext = createAutoScrollContext()

  trackPointer((moveEvent) => {
    const scrollOffset = autoScrollIfNeeded(autoScrollContext, moveEvent.clientX, moveEvent.clientY)
    const next = engine.projectResize(
      payload,
      moveEvent.clientX + scrollOffset.offsetX,
      moveEvent.clientY + scrollOffset.offsetY,
    )
    emit('update:rect', { paneId: props.paneId, rect: next })
  })
}

function parseKeyboardDirection(key: string): KeyboardDirection | null {
  if (key === 'ArrowLeft') {
    return 'left'
  }
  if (key === 'ArrowRight') {
    return 'right'
  }
  if (key === 'ArrowUp') {
    return 'up'
  }
  if (key === 'ArrowDown') {
    return 'down'
  }
  return null
}

function onHeaderKeydown(event: KeyboardEvent): void {
  if (!event.altKey) {
    return
  }

  const direction = parseKeyboardDirection(event.key)
  if (!direction) {
    return
  }

  event.preventDefault()
  emit('focus', props.paneId)

  if (event.shiftKey) {
    const payload: KeyboardResizePayload = {
      startRect: { ...props.rect },
      bounds: props.bounds,
      direction,
      step: KEYBOARD_STEP,
      minWidth: props.minWidth,
      minHeight: props.minHeight,
    }
    const next = engine.projectKeyboardResize(payload)
    emit('update:rect', { paneId: props.paneId, rect: next })
    return
  }

  const payload: KeyboardMovePayload = {
    startRect: { ...props.rect },
    bounds: props.bounds,
    direction,
    step: KEYBOARD_STEP,
  }
  const next = engine.projectKeyboardMove(payload)
  emit('update:rect', { paneId: props.paneId, rect: next })
}

onBeforeUnmount(() => {
  cleanupPointerTracking()
})
</script>
