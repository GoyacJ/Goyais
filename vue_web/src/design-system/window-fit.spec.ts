/**
 * SPDX-License-Identifier: Apache-2.0
 * Copyright (c) 2026 Goya
 * Author: Goya
 * Created: 2026-02-11
 * Version: v1.0.0
 * Description: Goyais source file.
 */

import { fitManifestToBoardWidth } from '@/design-system/window-fit'
import type { WindowPaneManifest } from '@/design-system/window-manifests'

describe('fitManifestToBoardWidth', () => {
  it('scales panes to consume board width with side paddings', () => {
    const manifest: WindowPaneManifest[] = [
      { id: 'left', x: 24, y: 20, w: 520, h: 200 },
      { id: 'right', x: 560, y: 20, w: 520, h: 200 },
    ]

    const fitted = fitManifestToBoardWidth(manifest, 1920)
    const minX = Math.min(...fitted.map((pane) => pane.x))
    const maxRight = Math.max(...fitted.map((pane) => pane.x + pane.w))

    expect(minX).toBe(24)
    expect(maxRight).toBeGreaterThanOrEqual(1892)
    expect(maxRight).toBeLessThanOrEqual(1896)
  })

  it('keeps minWidth constraints when scaling down', () => {
    const manifest: WindowPaneManifest[] = [
      { id: 'left', x: 24, y: 20, w: 520, h: 200, minWidth: 500 },
      { id: 'right', x: 560, y: 20, w: 520, h: 200, minWidth: 500 },
    ]

    const fitted = fitManifestToBoardWidth(manifest, 1100)

    expect(fitted[0]?.w).toBeGreaterThanOrEqual(500)
    expect(fitted[1]?.w).toBeGreaterThanOrEqual(500)
  })

  it('falls back to safe board width when given invalid input', () => {
    const manifest: WindowPaneManifest[] = [{ id: 'one', x: 24, y: 20, w: 400, h: 220 }]

    const fitted = fitManifestToBoardWidth(manifest, Number.NaN)

    expect(fitted[0]?.x).toBe(24)
    expect(fitted[0]?.w).toBeGreaterThan(0)
  })
})
