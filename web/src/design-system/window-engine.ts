import type { WindowRect } from '@/design-system/types'

export type ResizeDirection = 'right' | 'bottom' | 'corner'
export type KeyboardDirection = 'left' | 'right' | 'up' | 'down'

export interface WindowBounds {
  width: number
  height: number
}

export interface DragStartPayload {
  startClientX: number
  startClientY: number
  startRect: WindowRect
  bounds: WindowBounds
}

export interface ResizeStartPayload {
  startClientX: number
  startClientY: number
  startRect: WindowRect
  bounds: WindowBounds
  direction: ResizeDirection
  minWidth: number
  minHeight: number
}

export interface KeyboardMovePayload {
  startRect: WindowRect
  bounds: WindowBounds
  direction: KeyboardDirection
  step: number
}

export interface KeyboardResizePayload {
  startRect: WindowRect
  bounds: WindowBounds
  direction: KeyboardDirection
  step: number
  minWidth: number
  minHeight: number
}

export interface WindowEngine {
  projectDrag(payload: DragStartPayload, currentClientX: number, currentClientY: number): WindowRect
  projectResize(payload: ResizeStartPayload, currentClientX: number, currentClientY: number): WindowRect
  projectKeyboardMove(payload: KeyboardMovePayload): WindowRect
  projectKeyboardResize(payload: KeyboardResizePayload): WindowRect
}
