import { mount } from '@vue/test-utils'
import Table, { type TableColumn } from '@/components/ui/Table.vue'
import i18n from '@/i18n'

const columns: TableColumn[] = [
  { key: 'name', label: 'Name' },
  { key: 'status', label: 'Status' },
]

const rows = [
  { name: 'row-a', status: 'ok' },
  { name: 'row-b', status: 'running' },
]

describe('Table', () => {
  it('emits rowClick for Enter and Space when interactive rows are enabled', async () => {
    const wrapper = mount(Table, {
      props: {
        columns,
        rows,
        interactiveRows: true,
        selectedRowIndex: 1,
        rowKey: 'name',
      },
      global: {
        plugins: [i18n],
      },
    })

    const row = wrapper.findAll('tbody tr')[0]
    await row.trigger('keydown.enter')
    await row.trigger('keydown.space')

    const emissions = wrapper.emitted('rowClick')
    expect(emissions?.length).toBe(2)
    expect(emissions?.[0]?.[0]).toMatchObject({ index: 0, rowKey: 'row-a' })
    expect(emissions?.[1]?.[0]).toMatchObject({ index: 0, rowKey: 'row-a' })

    const selected = wrapper.findAll('tbody tr')[1]
    expect(selected.attributes('aria-selected')).toBe('true')
  })

  it('supports selectedRowKey and cell slots without breaking default rendering', async () => {
    const wrapper = mount(Table, {
      props: {
        columns,
        rows,
        interactiveRows: true,
        rowKey: 'name',
        selectedRowKey: 'row-b',
      },
      slots: {
        'cell-status': '<template #cell-status="{ value }"><span data-testid="status-cell">{{ value }}</span></template>',
      },
      global: {
        plugins: [i18n],
      },
    })

    const allRows = wrapper.findAll('tbody tr')
    expect(allRows[1]?.attributes('aria-selected')).toBe('true')
    expect(wrapper.find('[data-testid="status-cell"]').text()).toBe('ok')
  })
})
