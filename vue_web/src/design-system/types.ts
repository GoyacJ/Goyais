/**
 * SPDX-License-Identifier: Apache-2.0
 * Copyright (c) 2026 Goya
 * Author: Goya
 * Created: 2026-02-11
 * Version: v1.0.0
 * Description: Goyais source file.
 */

export type ThemeMode = 'system' | 'light' | 'dark'

export type DensityMode = 'compact' | 'comfortable'

export type SupportedLocale = 'zh-CN' | 'en-US'

export type LayoutMode = 'console' | 'topnav' | 'focus'

export type LayoutPreference = 'auto' | LayoutMode

export type CommandStatus = 'accepted' | 'running' | 'succeeded' | 'failed' | 'canceled'

export type TableState = 'ready' | 'loading' | 'empty' | 'error'

export type Visibility = 'PRIVATE' | 'WORKSPACE' | 'TENANT' | 'PUBLIC'

export type WindowPaneId = string

export interface WindowRect {
  x: number
  y: number
  w: number
  h: number
  z: number
}

export interface WindowState {
  panes: Record<WindowPaneId, WindowRect>
  nextZ: number
}
