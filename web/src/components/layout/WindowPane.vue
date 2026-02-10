<template>
  <article
    class="ui-window-pane ui-surface absolute flex min-h-0 flex-col overflow-hidden"
    :style="paneStyle"
    @pointerdown="emit('focus', paneId)"
  >
    <header
      class="ui-window-pane-header ui-bg-content flex h-[var(--ui-window-header-h)] cursor-move items-center justify-between border-b border-ui-border px-3"
      @pointerdown="onDragStart"
    >
      <p class="truncate text-xs font-semibold uppercase tracking-[0.08em] text-ui-muted">{{ title }}</p>
      <span class="ui-monospace text-[11px] text-ui-muted">{{ paneId }}</span>
    </header>

    <div class="ui-window-pane-body ui-scrollbar min-h-0 flex-1 overflow-auto p-1">
      <slot />
    </div>

    <button
      type="button"
      aria-label="resize-right"
      class="ui-window-resize-right"
      data-testid="resize-right"
      @pointerdown="onResizeStart($event, 'right')"
    />
    <button
      type="button"
      aria-label="resize-bottom"
      class="ui-window-resize-bottom"
      data-testid="resize-bottom"
      @pointerdown="onResizeStart($event, 'bottom')"
    />
    <button
      type="button"
      aria-label="resize-corner"
      class="ui-window-resize-corner"
      data-testid="resize-corner"
      @pointerdown="onResizeStart($event, 'corner')"
    />
  </article>
</template>

<script setup lang="ts">
import type {
  DragStartPayload,
  ResizeDirection,
  ResizeStartPayload,
  WindowBounds,
} from '@/design-system/window-engine'
import { NativePointerWindowEngine } from '@/design-system/window-engine-native'
import type { WindowRect } from '@/design-system/types'
import { computed, onBeforeUnmount } from 'vue'

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

const paneStyle = computed(() => ({
  left: `${props.rect.x}px`,
  top: `${props.rect.y}px`,
  width: `${props.rect.w}px`,
  height: `${props.rect.h}px`,
  zIndex: String(props.rect.z),
}))

let activeMoveHandler: ((event: PointerEvent) => void) | null = null
let activeUpHandler: ((event: PointerEvent) => void) | null = null

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

  trackPointer((moveEvent) => {
    const next = engine.projectDrag(payload, moveEvent.clientX, moveEvent.clientY)
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

  trackPointer((moveEvent) => {
    const next = engine.projectResize(payload, moveEvent.clientX, moveEvent.clientY)
    emit('update:rect', { paneId: props.paneId, rect: next })
  })
}

onBeforeUnmount(() => {
  cleanupPointerTracking()
})
</script>
