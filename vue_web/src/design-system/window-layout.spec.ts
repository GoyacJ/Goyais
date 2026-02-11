import { ref } from 'vue'
import { nextTick } from 'vue'
import { afterEach, vi } from 'vitest'
import type { LayoutMode } from '@/design-system/types'
import { useWindowLayout, windowStorageKey } from '@/design-system/window-layout'
import type { WindowPaneManifest } from '@/design-system/window-manifests'

describe('window layout store', () => {
  afterEach(() => {
    vi.useRealTimers()
    localStorage.clear()
  })

  it('persists pane changes and separates state by layout mode', async () => {
    const routeKey = ref('commands')
    const mode = ref<LayoutMode>('console')
    const manifest = ref<WindowPaneManifest[]>([
      { id: 'filters', x: 10, y: 20, w: 400, h: 120 },
      { id: 'list', x: 10, y: 160, w: 620, h: 420 },
    ])

    const store = useWindowLayout(routeKey, mode, manifest, { persistDebounceMs: 0 })

    const firstRect = { ...store.panes.value.list }
    store.bringToFront('list')
    store.updatePaneRect('list', { ...firstRect, x: 48, y: 222, w: 640, h: 430, z: firstRect.z + 2 })
    await nextTick()

    const consoleKey = windowStorageKey('commands', 'console')
    const persistedConsole = localStorage.getItem(consoleKey)
    expect(persistedConsole).toBeTruthy()
    expect(persistedConsole).toContain('"x":48')

    mode.value = 'topnav'
    await nextTick()
    const topnavRect = store.panes.value.list
    expect(topnavRect.x).toBe(10)
    expect(topnavRect.y).toBe(160)

    mode.value = 'console'
    await nextTick()
    expect(store.panes.value.list.x).toBe(48)
    expect(store.panes.value.list.y).toBe(222)
  })

  it('debounces localStorage writes for rapid updates', async () => {
    vi.useFakeTimers()
    const setItemSpy = vi.spyOn(Storage.prototype, 'setItem')

    const routeKey = ref('commands')
    const mode = ref<LayoutMode>('console')
    const manifest = ref<WindowPaneManifest[]>([
      { id: 'filters', x: 10, y: 20, w: 400, h: 120 },
      { id: 'list', x: 10, y: 160, w: 620, h: 420 },
    ])

    const store = useWindowLayout(routeKey, mode, manifest, { persistDebounceMs: 120 })
    const firstRect = { ...store.panes.value.list }

    store.updatePaneRect('list', { ...firstRect, x: 44 })
    store.updatePaneRect('list', { ...firstRect, x: 66 })
    store.updatePaneRect('list', { ...firstRect, x: 88 })
    await nextTick()

    expect(setItemSpy).toHaveBeenCalledTimes(0)
    vi.advanceTimersByTime(119)
    expect(setItemSpy).toHaveBeenCalledTimes(0)
    vi.advanceTimersByTime(1)
    expect(setItemSpy).toHaveBeenCalledTimes(1)

    const persisted = localStorage.getItem(windowStorageKey('commands', 'console'))
    expect(persisted).toContain('"x":88')
  })

  it('falls back missing panes to manifest defaults when persisted state is incomplete', async () => {
    const routeKey = ref('settings')
    const mode = ref<LayoutMode>('console')
    const manifest = ref<WindowPaneManifest[]>([
      { id: 'preferences', x: 24, y: 20, w: 1284, h: 260 },
      { id: 'component-matrix', x: 24, y: 296, w: 1284, h: 610 },
    ])
    const key = windowStorageKey(routeKey.value, mode.value)

    localStorage.setItem(
      key,
      JSON.stringify({
        panes: {
          preferences: { x: 30, y: 33, w: 900, h: 240, z: 9 },
        },
        nextZ: 10,
      }),
    )

    const store = useWindowLayout(routeKey, mode, manifest, { persistDebounceMs: 0 })
    await nextTick()

    expect(store.panes.value.preferences).toMatchObject({ x: 30, y: 33, w: 900, h: 240, z: 9 })
    expect(store.panes.value['component-matrix']).toMatchObject({
      x: 24,
      y: 296,
      w: 1284,
      h: 610,
      z: 2,
    })
  })
})
