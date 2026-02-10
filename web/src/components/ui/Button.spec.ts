import { mount } from '@vue/test-utils'
import Button from '@/components/ui/Button.vue'

describe('Button', () => {
  it('blocks interaction while loading by default', () => {
    const wrapper = mount(Button, {
      props: {
        loading: true,
      },
      slots: {
        default: 'Submit',
      },
    })

    expect(wrapper.attributes('disabled')).toBeDefined()
    expect(wrapper.classes()).toContain('ui-loading')
    expect(wrapper.classes()).toContain('ui-loading-block')
    expect(wrapper.attributes('data-block-loading')).toBe('true')
  })

  it('keeps button interactive while loading when blockWhileLoading is false', () => {
    const wrapper = mount(Button, {
      props: {
        loading: true,
        blockWhileLoading: false,
      },
      slots: {
        default: 'Submit',
      },
    })

    expect(wrapper.attributes('disabled')).toBeUndefined()
    expect(wrapper.classes()).toContain('ui-loading')
    expect(wrapper.classes()).not.toContain('ui-loading-block')
    expect(wrapper.attributes('data-block-loading')).toBe('false')
  })
})
