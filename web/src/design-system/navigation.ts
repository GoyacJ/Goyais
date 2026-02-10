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
  { to: '/commands', label: 'nav.commands', shortcut: '03', icon: 'commands' },
  { to: '/assets', label: 'nav.assets', shortcut: '04', icon: 'assets' },
  { to: '/plugins', label: 'nav.plugins', shortcut: '05', icon: 'plugins' },
  { to: '/streams', label: 'nav.streams', shortcut: '06', icon: 'streams' },
  { to: '/settings', label: 'nav.settings', shortcut: '07', icon: 'settings' },
]
