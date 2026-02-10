import { mount } from '@vue/test-utils'
import Table, { type TableColumn } from '@/components/ui/Table.vue'

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
      },
    })

    const row = wrapper.findAll('tbody tr')[0]
    await row.trigger('keydown.enter')
    await row.trigger('keydown.space')

    const emissions = wrapper.emitted('rowClick')
    expect(emissions?.length).toBe(2)
    expect(emissions?.[0]?.[0]).toMatchObject({ index: 0 })
    expect(emissions?.[1]?.[0]).toMatchObject({ index: 0 })

    const selected = wrapper.findAll('tbody tr')[1]
    expect(selected.attributes('aria-selected')).toBe('true')
  })
})
