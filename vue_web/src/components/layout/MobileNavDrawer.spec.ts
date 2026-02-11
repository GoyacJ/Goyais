import { mount, RouterLinkStub } from '@vue/test-utils'
import { nextTick } from 'vue'
import MobileNavDrawer from '@/components/layout/MobileNavDrawer.vue'
import i18n from '@/i18n'

const NAV_LABELS_ZH = ['首页', '画布', 'AI 工作台', '命令', '资源', '插件', '流媒体', '设置']
const NAV_LABELS_EN = ['Home', 'Canvas', 'AI Workbench', 'Commands', 'Assets', 'Plugins', 'Streams', 'Settings']

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

      const zhLinks = wrapper.findAll('a.ui-control')
      expect(zhLinks).toHaveLength(NAV_LABELS_ZH.length)
      expect(zhLinks.map((link) => link.find('.truncate').text())).toEqual(NAV_LABELS_ZH)

      i18n.global.locale.value = 'en-US'
      await nextTick()

      const enLinks = wrapper.findAll('a.ui-control')
      expect(enLinks.map((link) => link.find('.truncate').text())).toEqual(NAV_LABELS_EN)

      wrapper.unmount()
    } finally {
      i18n.global.locale.value = originalLocale
    }
  })
})
