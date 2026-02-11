/**
 * SPDX-License-Identifier: Apache-2.0
 * Copyright (c) 2026 Goya
 * Author: Goya
 * Created: 2026-02-11
 * Version: v1.0.0
 * Description: Goyais source file.
 */

import type { WindowPaneManifest } from '@/design-system/window-manifests'

const DEFAULT_SIDE_PADDING = 24
const MIN_INNER_WIDTH = 320

function clamp(value: number, min: number, max: number): number {
  return Math.min(max, Math.max(min, value))
}

function sanitizeBoardWidth(value: number): number {
  if (!Number.isFinite(value) || value <= 0) {
    return 1200
  }
  return value
}

export function fitManifestToBoardWidth(
  manifest: WindowPaneManifest[],
  boardWidthInput: number,
  sidePadding = DEFAULT_SIDE_PADDING,
): WindowPaneManifest[] {
  if (manifest.length === 0) {
    return []
  }

  const boardWidth = sanitizeBoardWidth(boardWidthInput)
  const minX = Math.min(...manifest.map((pane) => pane.x))
  const maxRight = Math.max(...manifest.map((pane) => pane.x + pane.w))
  const sourceWidth = Math.max(1, maxRight - minX)
  const targetWidth = Math.max(MIN_INNER_WIDTH, boardWidth - sidePadding * 2)
  const scale = targetWidth / sourceWidth
  const rightLimit = Math.max(sidePadding, boardWidth - sidePadding)

  return manifest.map((pane) => {
    const scaledWidth = Math.round(pane.w * scale)
    const minWidth = pane.minWidth ?? 0
    const width = Math.max(minWidth, scaledWidth)
    const projectedX = sidePadding + Math.round((pane.x - minX) * scale)
    const maxX = Math.max(sidePadding, rightLimit - width)

    return {
      ...pane,
      x: clamp(projectedX, sidePadding, maxX),
      w: width,
    }
  })
}
