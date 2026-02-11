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
  { to: '/commands', label: 'nav.commands', shortcut: '04', icon: 'commands' },
  { to: '/assets', label: 'nav.assets', shortcut: '05', icon: 'assets' },
  { to: '/plugins', label: 'nav.plugins', shortcut: '06', icon: 'plugins' },
  { to: '/streams', label: 'nav.streams', shortcut: '07', icon: 'streams' },
  { to: '/settings', label: 'nav.settings', shortcut: '08', icon: 'settings' },
]
