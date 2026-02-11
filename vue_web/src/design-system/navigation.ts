/**
 * SPDX-License-Identifier: Apache-2.0
 * Copyright (c) 2026 Goya
 * Author: Goya
 * Created: 2026-02-11
 * Version: v1.0.0
 * Description: Goyais source file.
 */

import type { IconName } from '@/design-system/icon-registry'

export interface NavItem {
  to: string
  label: string
  shortcut: string
  icon: IconName
}

export const NAV_ITEMS: NavItem[] = [
  { to: '/', label: 'nav.home', shortcut: '01', icon: 'home' },
  { to: '/canvas', label: 'nav.canvas', shortcut: '02', icon: 'canvas' },
  { to: '/ai', label: 'nav.ai', shortcut: '03', icon: 'commands' },
  { to: '/run-center', label: 'nav.runCenter', shortcut: '04', icon: 'commands' },
  { to: '/commands', label: 'nav.commands', shortcut: '05', icon: 'commands' },
  { to: '/assets', label: 'nav.assets', shortcut: '06', icon: 'assets' },
  { to: '/algorithm-library', label: 'nav.algorithmLibrary', shortcut: '07', icon: 'canvas' },
  { to: '/plugins', label: 'nav.plugins', shortcut: '08', icon: 'plugins' },
  { to: '/streams', label: 'nav.streams', shortcut: '09', icon: 'streams' },
  { to: '/permissions', label: 'nav.permissions', shortcut: '10', icon: 'forbidden' },
  { to: '/context-bundles', label: 'nav.contextBundles', shortcut: '11', icon: 'assets' },
  { to: '/settings', label: 'nav.settings', shortcut: '12', icon: 'settings' },
]
