/**
 * SPDX-License-Identifier: Apache-2.0
 * Copyright (c) 2026 Goya
 * Author: Goya
 * Created: 2026-02-11
 * Version: v1.0.0
 * Description: Verify SideNav rendering, pin behavior, and locale sync.
 */

import { mount, RouterLinkStub } from '@vue/test-utils'
import { nextTick } from 'vue'
import SideNav from '@/components/layout/SideNav.vue'
import { useDensityStore } from '@/design-system/density'
import i18n from '@/i18n'

const PINNED_STORAGE_KEY = 'goyais.ui.sidenav.pinned'
const NAV_LABELS_ZH = ['首页', '画布', 'AI 工作台', '运行中心', '命令', '资源', '算法库', '插件', '流媒体', '权限管理', '上下文包', '设置']
const NAV_LABELS_EN = [
  'Home',
  'Canvas',
  'AI Workbench',
  'Run Center',
  'Commands',
  'Assets',
  'Algorithm Library',
  'Plugins',
  'Streams',
  'Permissions',
  'Context Bundles',
  'Settings',
]

function mountSideNav() {
  return mount(SideNav, {
    global: {
      plugins: [i18n],
      stubs: {
        RouterLink: RouterLinkStub,
        WorkspaceAccountMenu: {
          template: '<div data-testid="workspace-account-menu" />',
        },
      },
    },
  })
}

describe('SideNav', () => {
  it('renders localized menu labels and keeps collapsed titles in sync with locale', async () => {
    const originalLocale = i18n.global.locale.value
    localStorage.removeItem(PINNED_STORAGE_KEY)

    try {
      const wrapper = mountSideNav()

      i18n.global.locale.value = 'zh-CN'
      await nextTick()

      const zhCollapsedLinks = wrapper.findAll('a.ui-nav-link')
      expect(zhCollapsedLinks).toHaveLength(NAV_LABELS_ZH.length)
      expect(zhCollapsedLinks.map((link) => link.attributes('title'))).toEqual(NAV_LABELS_ZH)
      expect(zhCollapsedLinks[0]?.attributes('active-class')).toBe('ui-nav-link-active')

      const pinButton = wrapper.get('button[data-pinned]')
      await pinButton.trigger('click')
      await nextTick()

      const zhExpandedLinks = wrapper.findAll('a.ui-nav-link')
      expect(zhExpandedLinks.map((link) => link.find('.truncate').text())).toEqual(NAV_LABELS_ZH)

      i18n.global.locale.value = 'en-US'
      await nextTick()

      const enExpandedLinks = wrapper.findAll('a.ui-nav-link')
      expect(enExpandedLinks.map((link) => link.find('.truncate').text())).toEqual(NAV_LABELS_EN)

      await pinButton.trigger('click')
      await nextTick()

      const enCollapsedLinks = wrapper.findAll('a.ui-nav-link')
      expect(enCollapsedLinks.map((link) => link.attributes('title'))).toEqual(NAV_LABELS_EN)
      expect(wrapper.find('[data-testid="workspace-account-menu"]').exists()).toBe(true)

      wrapper.unmount()
    } finally {
      localStorage.removeItem(PINNED_STORAGE_KEY)
      i18n.global.locale.value = originalLocale
    }
  })

  it('keeps header and menu region dimensions stable across locale switches', async () => {
    const originalLocale = i18n.global.locale.value
    const wrapper = mountSideNav()

    const header = wrapper.find('aside > div')
    const nav = wrapper.find('nav')

    expect(header.classes()).toContain('h-[4.5rem]')
    expect(header.classes()).toContain('shrink-0')
    expect(nav.classes()).toContain('flex-1')
    expect(nav.classes()).toContain('overflow-auto')

    i18n.global.locale.value = 'zh-CN'
    await nextTick()
    expect(header.classes()).toContain('h-[4.5rem]')
    expect(header.classes()).toContain('shrink-0')

    i18n.global.locale.value = 'en-US'
    await nextTick()
    expect(header.classes()).toContain('h-[4.5rem]')
    expect(header.classes()).toContain('shrink-0')

    i18n.global.locale.value = originalLocale
    wrapper.unmount()
  })

  it('toggles pinned floating state and restores it from storage', async () => {
    const originalLocale = i18n.global.locale.value
    const { densityMode } = useDensityStore()
    const originalDensity = densityMode.value
    localStorage.removeItem(PINNED_STORAGE_KEY)
    i18n.global.locale.value = 'en-US'
    densityMode.value = 'compact'

    try {
      const wrapper = mountSideNav()

      const aside = wrapper.get('aside')
      const pinButton = wrapper.get('button[data-pinned]')

      expect(aside.classes()).toContain('w-[4.75rem]')
      expect(pinButton.attributes('aria-label')).toBe('Pin navigation')

      await pinButton.trigger('click')
      await nextTick()

      expect(pinButton.attributes('data-pinned')).toBe('true')
      expect(pinButton.attributes('aria-label')).toBe('Unpin navigation')
      expect(aside.classes()).toContain('w-64')
      expect(localStorage.getItem(PINNED_STORAGE_KEY)).toBe('true')

      wrapper.unmount()

      const restored = mountSideNav()
      await nextTick()

      const restoredAside = restored.get('aside')
      const restoredPinButton = restored.get('button[data-pinned]')

      expect(restoredPinButton.attributes('data-pinned')).toBe('true')
      expect(restoredAside.classes()).toContain('w-64')

      await restoredPinButton.trigger('click')
      await nextTick()
      expect(restoredPinButton.attributes('data-pinned')).toBe('false')
      expect(restoredAside.classes()).toContain('w-[4.75rem]')
      expect(localStorage.getItem(PINNED_STORAGE_KEY)).toBe('false')

      restored.unmount()
    } finally {
      localStorage.removeItem(PINNED_STORAGE_KEY)
      i18n.global.locale.value = originalLocale
      densityMode.value = originalDensity
    }
  })

  it('collapses when floating even under comfortable density', async () => {
    const originalLocale = i18n.global.locale.value
    const { densityMode } = useDensityStore()
    const originalDensity = densityMode.value
    localStorage.removeItem(PINNED_STORAGE_KEY)
    i18n.global.locale.value = 'en-US'
    densityMode.value = 'comfortable'

    try {
      const wrapper = mountSideNav()

      const aside = wrapper.get('aside')
      const pinButton = wrapper.get('button[data-pinned]')

      expect(aside.classes()).toContain('w-[4.75rem]')
      expect(pinButton.attributes('data-pinned')).toBe('false')

      await pinButton.trigger('click')
      await nextTick()
      expect(aside.classes()).toContain('w-64')
      expect(pinButton.attributes('data-pinned')).toBe('true')

      await pinButton.trigger('click')
      await nextTick()
      expect(aside.classes()).toContain('w-[4.75rem]')
      expect(pinButton.attributes('data-pinned')).toBe('false')

      wrapper.unmount()
    } finally {
      localStorage.removeItem(PINNED_STORAGE_KEY)
      i18n.global.locale.value = originalLocale
      densityMode.value = originalDensity
    }
  })
})
