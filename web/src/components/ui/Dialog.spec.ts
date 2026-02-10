import { mount } from '@vue/test-utils'
import Dialog from '@/components/ui/Dialog.vue'
import i18n from '@/i18n'

describe('Dialog', () => {
  it('uses localized default footer labels when none are provided', () => {
    const wrapper = mount(Dialog, {
      props: {
        open: true,
        title: 'Title',
      },
      global: {
        plugins: [i18n],
        stubs: {
          TransitionRoot: { template: '<div><slot /></div>' },
          TransitionChild: { template: '<div><slot /></div>' },
          Dialog: { template: '<div><slot /></div>' },
          DialogPanel: { template: '<section><slot /></section>' },
          DialogTitle: { template: '<h2><slot /></h2>' },
          DialogDescription: { template: '<p><slot /></p>' },
        },
      },
      attachTo: document.body,
    })

    const renderedText = wrapper.text()
    expect(renderedText).toContain(i18n.global.t('common.cancel'))
    expect(renderedText).toContain(i18n.global.t('common.confirm'))

    wrapper.unmount()
  })
})
