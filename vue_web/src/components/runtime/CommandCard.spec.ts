import { mount } from '@vue/test-utils'
import CommandCard from '@/components/runtime/CommandCard.vue'
import i18n from '@/i18n'
import { mockCommands } from '@/mocks/commands'

describe('CommandCard', () => {
  it('emits select on Enter and Space for interactive cards', async () => {
    const command = mockCommands[0]
    const wrapper = mount(CommandCard, {
      props: {
        command,
        interactive: true,
        selected: true,
      },
      global: {
        plugins: [i18n],
      },
    })

    await wrapper.trigger('keydown.enter')
    await wrapper.trigger('keydown.space')

    const emissions = wrapper.emitted('select')
    expect(emissions?.length).toBe(2)
    expect(emissions?.[0]?.[0]).toBe(command.commandId)
    expect(emissions?.[1]?.[0]).toBe(command.commandId)
    expect(wrapper.attributes('aria-selected')).toBe('true')
  })
})
