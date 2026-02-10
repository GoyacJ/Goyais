import { mount } from '@vue/test-utils'
import CommandsView from '@/views/CommandsView.vue'
import i18n from '@/i18n'

const stubs = {
  WindowBoard: {
    props: ['routeKey', 'panes'],
    template: '<div><slot name="filters" /><slot name="list" /><slot name="detail" /></div>',
  },
  PageHeader: {
    props: ['title', 'subtitle'],
    template: '<header><h1>{{ title }}</h1><p>{{ subtitle }}</p><slot name="actions" /></header>',
  },
  SectionCard: {
    props: ['title', 'subtitle'],
    template: '<section><h2>{{ title }}</h2><p>{{ subtitle }}</p><slot /></section>',
  },
  EmptyState: { template: '<div data-testid="empty-state"><slot /></div>' },
  Button: { template: '<button><slot /></button>' },
  Icon: { template: '<span />' },
  Select: { template: '<div data-testid="select-stub" />' },
  Input: { template: '<input />' },
  Tabs: { template: '<div data-testid="tabs-stub" />' },
  LogPanel: { template: '<div data-testid="log-panel" />' },
}

describe('CommandsView', () => {
  it('updates detail content when selecting rows via mouse and keyboard', async () => {
    const wrapper = mount(CommandsView, {
      global: {
        plugins: [i18n],
        stubs,
      },
    })

    expect(wrapper.text()).toContain('Queued at command gate.')

    const rows = wrapper.findAll('tbody tr')
    await rows[1]?.trigger('click')
    expect(wrapper.text()).toContain('Dispatching 3 steps.')

    await rows[2]?.trigger('keydown.space')
    expect(wrapper.text()).toContain('Plugin package verified and enabled.')
  })
})
