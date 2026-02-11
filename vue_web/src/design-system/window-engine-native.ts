import type {
  WindowEngine,
  DragStartPayload,
  KeyboardMovePayload,
  KeyboardResizePayload,
  ResizeStartPayload,
} from '@/design-system/window-engine'
import type { WindowRect } from '@/design-system/types'

function clamp(value: number, min: number, max: number): number {
  return Math.min(max, Math.max(min, value))
}

function sanitizeBounds(value: number, fallback: number): number {
  return Number.isFinite(value) && value > 0 ? value : fallback
}

function clampPosition(next: WindowRect, boardWidth: number, boardHeight: number): WindowRect {
  const maxX = Math.max(0, boardWidth - next.w)
  const maxY = Math.max(0, boardHeight - next.h)

  return {
    ...next,
    x: clamp(next.x, 0, maxX),
    y: clamp(next.y, 0, maxY),
  }
}

export class NativePointerWindowEngine implements WindowEngine {
  projectDrag(payload: DragStartPayload, currentClientX: number, currentClientY: number): WindowRect {
    const deltaX = currentClientX - payload.startClientX
    const deltaY = currentClientY - payload.startClientY

    const boardWidth = sanitizeBounds(payload.bounds.width, payload.startRect.w)
    const boardHeight = sanitizeBounds(payload.bounds.height, payload.startRect.h)

    return clampPosition(
      {
        ...payload.startRect,
        x: payload.startRect.x + deltaX,
        y: payload.startRect.y + deltaY,
      },
      boardWidth,
      boardHeight,
    )
  }

  projectResize(payload: ResizeStartPayload, currentClientX: number, currentClientY: number): WindowRect {
    const deltaX = currentClientX - payload.startClientX
    const deltaY = currentClientY - payload.startClientY
    const boardWidth = sanitizeBounds(payload.bounds.width, payload.startRect.w)
    const boardHeight = sanitizeBounds(payload.bounds.height, payload.startRect.h)

    let next = { ...payload.startRect }

    if (payload.direction === 'right' || payload.direction === 'corner') {
      const maxWidth = Math.max(payload.minWidth, boardWidth - payload.startRect.x)
      next.w = clamp(payload.startRect.w + deltaX, payload.minWidth, maxWidth)
    }

    if (payload.direction === 'bottom' || payload.direction === 'corner') {
      const maxHeight = Math.max(payload.minHeight, boardHeight - payload.startRect.y)
      next.h = clamp(payload.startRect.h + deltaY, payload.minHeight, maxHeight)
    }

    next = clampPosition(next, boardWidth, boardHeight)
    return next
  }

  projectKeyboardMove(payload: KeyboardMovePayload): WindowRect {
    const boardWidth = sanitizeBounds(payload.bounds.width, payload.startRect.w)
    const boardHeight = sanitizeBounds(payload.bounds.height, payload.startRect.h)

    let deltaX = 0
    let deltaY = 0
    if (payload.direction === 'left') {
      deltaX = -payload.step
    }
    if (payload.direction === 'right') {
      deltaX = payload.step
    }
    if (payload.direction === 'up') {
      deltaY = -payload.step
    }
    if (payload.direction === 'down') {
      deltaY = payload.step
    }

    return clampPosition(
      {
        ...payload.startRect,
        x: payload.startRect.x + deltaX,
        y: payload.startRect.y + deltaY,
      },
      boardWidth,
      boardHeight,
    )
  }

  projectKeyboardResize(payload: KeyboardResizePayload): WindowRect {
    const boardWidth = sanitizeBounds(payload.bounds.width, payload.startRect.w)
    const boardHeight = sanitizeBounds(payload.bounds.height, payload.startRect.h)
    let next = { ...payload.startRect }

    if (payload.direction === 'left' || payload.direction === 'right') {
      const maxWidth = Math.max(payload.minWidth, boardWidth - payload.startRect.x)
      const sign = payload.direction === 'right' ? 1 : -1
      next.w = clamp(payload.startRect.w + sign * payload.step, payload.minWidth, maxWidth)
    }

    if (payload.direction === 'up' || payload.direction === 'down') {
      const maxHeight = Math.max(payload.minHeight, boardHeight - payload.startRect.y)
      const sign = payload.direction === 'down' ? 1 : -1
      next.h = clamp(payload.startRect.h + sign * payload.step, payload.minHeight, maxHeight)
    }

    return clampPosition(next, boardWidth, boardHeight)
  }
}
