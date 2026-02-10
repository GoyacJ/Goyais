import { ref } from 'vue'
import { nextTick } from 'vue'
import type { LayoutMode } from '@/design-system/types'
import { useWindowLayout, windowStorageKey } from '@/design-system/window-layout'
import type { WindowPaneManifest } from '@/design-system/window-manifests'

describe('window layout store', () => {
  it('persists pane changes and separates state by layout mode', async () => {
    const routeKey = ref('commands')
    const mode = ref<LayoutMode>('console')
    const manifest = ref<WindowPaneManifest[]>([
      { id: 'filters', x: 10, y: 20, w: 400, h: 120 },
      { id: 'list', x: 10, y: 160, w: 620, h: 420 },
    ])

    const store = useWindowLayout(routeKey, mode, manifest)

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
})
