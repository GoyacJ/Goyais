import { mount } from '@vue/test-utils'
import type { CommandDTO, StreamDTO } from '@/api/types'
import i18n from '@/i18n'
import StreamsView from '@/views/StreamsView.vue'
import { nextTick } from 'vue'
import { beforeEach, vi } from 'vitest'

const listStreamsMock = vi.fn()
const getStreamMock = vi.fn()
const createStreamMock = vi.fn()
const startStreamRecordingMock = vi.fn()
const stopStreamRecordingMock = vi.fn()
const kickStreamMock = vi.fn()
const listCommandsMock = vi.fn()

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

vi.mock('@/api/streams', () => ({
  listStreams: (...args: unknown[]) => listStreamsMock(...args),
  getStream: (...args: unknown[]) => getStreamMock(...args),
  createStream: (...args: unknown[]) => createStreamMock(...args),
  startStreamRecording: (...args: unknown[]) => startStreamRecordingMock(...args),
  stopStreamRecording: (...args: unknown[]) => stopStreamRecordingMock(...args),
  kickStream: (...args: unknown[]) => kickStreamMock(...args),
}))

vi.mock('@/api/commands', () => ({
  listCommands: (...args: unknown[]) => listCommandsMock(...args),
}))

const stubs = {
  WindowBoard: {
    props: ['routeKey', 'panes'],
    template: '<div><slot name="stream-overview" /><slot name="stream-logs" /></div>',
  },
  PageHeader: {
    props: ['title', 'subtitle'],
    template: '<header><h1>{{ title }}</h1><p>{{ subtitle }}</p><slot name="actions" /></header>',
  },
  SectionCard: {
    props: ['title', 'subtitle'],
    template: '<section><h2>{{ title }}</h2><p>{{ subtitle }}</p><slot /></section>',
  },
  StreamControlPanel: {
    props: ['selectedStream', 'busy'],
    template: `
      <div>
        <button data-testid="create-stream" @click="$emit('create', { path: 'live/cam-02', protocol: 'rtmp', source: 'push', visibility: 'PRIVATE', onPublishTemplateId: '' })">create</button>
        <button data-testid="record-start" @click="$emit('recordStart')">start</button>
      </div>
    `,
  },
  StreamCommandLog: {
    props: ['events'],
    template: '<div data-testid="stream-events">{{ events.length }}</div>',
  },
  Button: { template: '<button><slot /></button>' },
  Icon: { template: '<span />' },
}

async function flushAll(): Promise<void> {
  await Promise.resolve()
  await nextTick()
  await Promise.resolve()
}

function buildStream(id: string, path: string): StreamDTO {
  return {
    id,
    tenantId: 't1',
    workspaceId: 'w1',
    ownerId: 'u1',
    visibility: 'PRIVATE',
    acl: [],
    status: 'online',
    createdAt: '2026-02-10T10:00:00Z',
    updatedAt: '2026-02-10T10:00:01Z',
    path,
    protocol: 'rtmp',
    source: 'push',
    endpoints: {},
    state: {},
  }
}

function buildStreamCommand(): CommandDTO {
  return {
    id: 'cmd_stream_create',
    tenantId: 't1',
    workspaceId: 'w1',
    ownerId: 'u1',
    visibility: 'PRIVATE',
    acl: [],
    status: 'succeeded',
    createdAt: '2026-02-10T10:00:00Z',
    updatedAt: '2026-02-10T10:00:01Z',
    acceptedAt: '2026-02-10T10:00:00Z',
    traceId: 'trace-stream',
    commandType: 'stream.create',
    payload: { path: 'live/cam-01' },
    result: { stream: { id: 'str_1', path: 'live/cam-01' } },
  }
}

describe('StreamsView', () => {
  beforeEach(() => {
    listStreamsMock.mockReset()
    getStreamMock.mockReset()
    createStreamMock.mockReset()
    startStreamRecordingMock.mockReset()
    stopStreamRecordingMock.mockReset()
    kickStreamMock.mockReset()
    listCommandsMock.mockReset()

    listStreamsMock.mockResolvedValue({
      items: [buildStream('str_1', 'live/cam-01')],
      pageInfo: { page: 1, pageSize: 20, total: 1 },
    })
    getStreamMock.mockResolvedValue(buildStream('str_1', 'live/cam-01'))
    listCommandsMock.mockResolvedValue({
      items: [buildStreamCommand()],
      pageInfo: { page: 1, pageSize: 20, total: 1 },
    })
    createStreamMock.mockResolvedValue({
      resource: buildStream('str_2', 'live/cam-02'),
      commandRef: { commandId: 'cmd_create_stream', status: 'accepted', acceptedAt: '2026-02-10T10:00:00Z' },
    })
    startStreamRecordingMock.mockResolvedValue({
      resource: buildStream('str_1', 'live/cam-01'),
      commandRef: { commandId: 'cmd_record_start', status: 'accepted', acceptedAt: '2026-02-10T10:00:00Z' },
    })
  })

  it('loads stream list and executes stream actions', async () => {
    const wrapper = mount(StreamsView, {
      global: {
        plugins: [i18n],
        stubs,
      },
    })

    await flushAll()

    expect(wrapper.text()).toContain('live/cam-01')
    expect(wrapper.get('[data-testid="stream-events"]').text()).toContain('1')

    await wrapper.get('[data-testid="create-stream"]').trigger('click')
    expect(createStreamMock).toHaveBeenCalledTimes(1)
    await flushAll()

    await wrapper.get('[data-testid="record-start"]').trigger('click')
    expect(startStreamRecordingMock).toHaveBeenCalledTimes(1)
  })
})
