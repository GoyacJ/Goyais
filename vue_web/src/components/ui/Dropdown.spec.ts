import { mount } from '@vue/test-utils'
import Dropdown from '@/components/ui/Dropdown.vue'
import i18n from '@/i18n'

describe('Dropdown', () => {
  it('renders localized fallback label and emits select for menu item', async () => {
    const wrapper = mount(Dropdown, {
      props: {
        items: [
          { label: 'Run', value: 'run' },
          { label: 'Delete', value: 'delete', danger: true },
        ],
      },
      global: {
        plugins: [i18n],
      },
      attachTo: document.body,
    })

    expect(wrapper.text()).toContain(i18n.global.t('common.userMenu'))

    await wrapper.find('button').trigger('click')
    const menuButtons = wrapper.findAll('button').filter((item) => item.text() === 'Run')
    await menuButtons[0]?.trigger('click')

    const emissions = wrapper.emitted('select')
    expect(emissions?.[0]?.[0]).toBe('run')

    wrapper.unmount()
  })
})
