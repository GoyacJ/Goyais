/**
 * SPDX-License-Identifier: Apache-2.0
 * Copyright (c) 2026 Goya
 * Author: Goya
 * Created: 2026-02-11
 * Version: v1.0.0
 * Description: Goyais source file.
 */

import { mount, RouterLinkStub } from '@vue/test-utils'
import { nextTick } from 'vue'
import MobileNavDrawer from '@/components/layout/MobileNavDrawer.vue'
import i18n from '@/i18n'

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

describe('MobileNavDrawer', () => {
  it('renders localized menu labels and updates immediately when locale changes', async () => {
    const originalLocale = i18n.global.locale.value

    try {
      i18n.global.locale.value = 'zh-CN'

      const wrapper = mount(MobileNavDrawer, {
        props: {
          open: true,
        },
        global: {
          plugins: [i18n],
          stubs: {
            RouterLink: RouterLinkStub,
          },
        },
      })

      await nextTick()

      const zhLinks = wrapper.findAll('a.ui-nav-link')
      expect(zhLinks).toHaveLength(NAV_LABELS_ZH.length)
      expect(zhLinks.map((link) => link.find('.truncate').text())).toEqual(NAV_LABELS_ZH)
      expect(zhLinks[0]?.attributes('active-class')).toBe('ui-nav-link-active')

      i18n.global.locale.value = 'en-US'
      await nextTick()

      const enLinks = wrapper.findAll('a.ui-nav-link')
      expect(enLinks.map((link) => link.find('.truncate').text())).toEqual(NAV_LABELS_EN)

      wrapper.unmount()
    } finally {
      i18n.global.locale.value = originalLocale
    }
  })
})
