import { mount, RouterLinkStub } from '@vue/test-utils'
import { nextTick } from 'vue'
import SideNav from '@/components/layout/SideNav.vue'
import i18n from '@/i18n'

describe('SideNav', () => {
  it('keeps header and menu region dimensions stable across locale switches', async () => {
    const originalLocale = i18n.global.locale.value
    const wrapper = mount(SideNav, {
      global: {
        plugins: [i18n],
        stubs: {
          RouterLink: RouterLinkStub,
        },
      },
    })

    const header = wrapper.find('aside > div')
    const nav = wrapper.find('nav')

    expect(header.classes()).toContain('h-[4.25rem]')
    expect(header.classes()).toContain('shrink-0')
    expect(nav.classes()).toContain('flex-1')
    expect(nav.classes()).toContain('overflow-auto')

    i18n.global.locale.value = 'zh-CN'
    await nextTick()
    expect(header.classes()).toContain('h-[4.25rem]')
    expect(header.classes()).toContain('shrink-0')

    i18n.global.locale.value = 'en-US'
    await nextTick()
    expect(header.classes()).toContain('h-[4.25rem]')
    expect(header.classes()).toContain('shrink-0')

    i18n.global.locale.value = originalLocale
    wrapper.unmount()
  })
})
