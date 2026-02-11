import { mount } from '@vue/test-utils'
import type { CommandDTO } from '@/api/types'
import CommandsView from '@/views/CommandsView.vue'
import i18n from '@/i18n'
import { nextTick } from 'vue'
import { beforeEach, vi } from 'vitest'

const listCommandsMock = vi.fn()
const createCommandMock = vi.fn()

vi.mock('@/api/http', () => ({
  isMockEnabled: () => false,
  ApiHttpError: class ApiHttpError extends Error {
    readonly status: number
    readonly error: { code: string; messageKey: string }

    constructor(status: number, error: { code: string; messageKey: string }) {
      super(error.messageKey)
      this.status = status
      this.error = error
    }
  },
}))

vi.mock('@/api/commands', () => ({
  listCommands: (...args: unknown[]) => listCommandsMock(...args),
  createCommand: (...args: unknown[]) => createCommandMock(...args),
}))

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

async function flushAll(): Promise<void> {
  await Promise.resolve()
  await nextTick()
  await Promise.resolve()
}

function buildCommand(id: string, status: CommandDTO['status'], ownerId: string, commandType: string): CommandDTO {
  return {
    id,
    tenantId: 't1',
    workspaceId: 'w1',
    ownerId,
    visibility: 'PRIVATE',
    acl: [],
    status,
    createdAt: '2026-02-10T10:00:00Z',
    updatedAt: '2026-02-10T10:00:01Z',
    acceptedAt: '2026-02-10T10:00:00Z',
    traceId: `trace-${id}`,
    commandType,
    payload: { source: 'spec' },
    result: { ok: true, id },
  }
}

describe('CommandsView', () => {
  beforeEach(() => {
    listCommandsMock.mockReset()
    createCommandMock.mockReset()

    listCommandsMock.mockResolvedValue({
      items: [
        buildCommand('cmd_1', 'accepted', 'u_alice', 'workflow.run'),
        buildCommand('cmd_2', 'running', 'u_bob', 'plugin.install'),
        buildCommand('cmd_3', 'succeeded', 'u_carol', 'stream.record.start'),
      ],
      pageInfo: {
        page: 1,
        pageSize: 20,
        total: 3,
      },
    })

    createCommandMock.mockResolvedValue({
      resource: buildCommand('cmd_new', 'accepted', 'u_alice', 'ui.ping'),
      commandRef: {
        commandId: 'cmd_new',
        status: 'accepted',
        acceptedAt: '2026-02-10T10:01:00Z',
      },
    })
  })

  it('updates detail content when selecting rows via mouse and keyboard', async () => {
    const wrapper = mount(CommandsView, {
      global: {
        plugins: [i18n],
        stubs,
      },
    })

    await flushAll()

    expect(wrapper.text()).toContain('workflow.run')

    const rows = wrapper.findAll('tbody tr')
    await rows[1]?.trigger('click')
    expect(wrapper.text()).toContain('plugin.install')

    await rows[2]?.trigger('keydown.space')
    expect(wrapper.text()).toContain('stream.record.start')
  })
})
