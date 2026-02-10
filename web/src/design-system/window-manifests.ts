import type { WindowPaneId } from '@/design-system/types'

export interface WindowPaneManifest {
  id: WindowPaneId
  x: number
  y: number
  w: number
  h: number
  minWidth?: number
  minHeight?: number
}

const manifests: Record<string, WindowPaneManifest[]> = {
  home: [
    { id: 'design-tokens', x: 24, y: 20, w: 420, h: 250 },
    { id: 'state-hooks', x: 456, y: 20, w: 420, h: 250 },
    { id: 'status', x: 888, y: 20, w: 420, h: 250 },
    { id: 'backgrounds', x: 24, y: 286, w: 640, h: 260 },
    { id: 'empty-states', x: 676, y: 286, w: 632, h: 500 },
  ],
  commands: [
    { id: 'filters', x: 24, y: 20, w: 1284, h: 190, minHeight: 160 },
    { id: 'list', x: 24, y: 226, w: 760, h: 620 },
    { id: 'detail', x: 800, y: 226, w: 508, h: 620 },
  ],
  assets: [
    { id: 'filters', x: 24, y: 20, w: 1284, h: 190, minHeight: 160 },
    { id: 'list', x: 24, y: 226, w: 800, h: 620 },
    { id: 'detail', x: 840, y: 226, w: 468, h: 620 },
  ],
  canvas: [{ id: 'canvas-surface', x: 24, y: 20, w: 1284, h: 760 }],
  plugins: [{ id: 'plugin-catalog', x: 24, y: 20, w: 1284, h: 700 }],
  streams: [
    { id: 'stream-overview', x: 24, y: 20, w: 1284, h: 420 },
    { id: 'stream-logs', x: 24, y: 456, w: 1284, h: 290 },
  ],
  settings: [
    { id: 'preferences', x: 24, y: 20, w: 1284, h: 260 },
    { id: 'component-matrix', x: 24, y: 296, w: 1284, h: 610 },
  ],
  forbidden: [{ id: 'forbidden-state', x: 180, y: 80, w: 980, h: 460 }],
  'not-found': [{ id: 'not-found-state', x: 180, y: 80, w: 980, h: 460 }],
}

export function windowManifestFor(routeKey: string): WindowPaneManifest[] {
  return (manifests[routeKey] ?? []).map((item) => ({ ...item }))
}
