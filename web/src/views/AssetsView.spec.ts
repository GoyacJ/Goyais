import { mount } from '@vue/test-utils'
import AssetsView from '@/views/AssetsView.vue'
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
}

describe('AssetsView', () => {
  it('updates detail content when selecting rows via mouse and keyboard', async () => {
    const wrapper = mount(AssetsView, {
      global: {
        plugins: [i18n],
        stubs,
      },
    })

    expect(wrapper.text()).toContain('warehouse-cam-01.mp4')

    const rows = wrapper.findAll('tbody tr')
    await rows[1]?.trigger('click')
    expect(wrapper.text()).toContain('dashboard-snapshot.png')

    await rows[2]?.trigger('keydown.enter')
    expect(wrapper.text()).toContain('daily-report.json')
  })
})
