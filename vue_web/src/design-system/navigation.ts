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
  icon: IconName
}

export const NAV_ITEMS: NavItem[] = [
  { to: '/', label: 'nav.home', icon: 'home' },
  { to: '/canvas', label: 'nav.canvas', icon: 'canvas' },
  { to: '/ai', label: 'nav.ai', icon: 'commands' },
  { to: '/run-center', label: 'nav.runCenter', icon: 'commands' },
  { to: '/commands', label: 'nav.commands', icon: 'commands' },
  { to: '/assets', label: 'nav.assets', icon: 'assets' },
  { to: '/algorithm-library', label: 'nav.algorithmLibrary', icon: 'canvas' },
  { to: '/plugins', label: 'nav.plugins', icon: 'plugins' },
  { to: '/streams', label: 'nav.streams', icon: 'streams' },
  { to: '/permissions', label: 'nav.permissions', icon: 'forbidden' },
  { to: '/context-bundles', label: 'nav.contextBundles', icon: 'assets' },
  { to: '/settings', label: 'nav.settings', icon: 'settings' },
]
