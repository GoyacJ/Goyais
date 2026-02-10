import type { WindowRect } from '@/design-system/types'

export type ResizeDirection = 'right' | 'bottom' | 'corner'

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

export interface WindowEngine {
  projectDrag(payload: DragStartPayload, currentClientX: number, currentClientY: number): WindowRect
  projectResize(payload: ResizeStartPayload, currentClientX: number, currentClientY: number): WindowRect
}
