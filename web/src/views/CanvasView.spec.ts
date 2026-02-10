import { mount } from '@vue/test-utils'
import type { AlgorithmDTO, CapabilityDTO, ProviderDTO, WorkflowRunDTO, WorkflowTemplateDTO } from '@/api/types'
import i18n from '@/i18n'
import CanvasView from '@/views/CanvasView.vue'
import { nextTick } from 'vue'
import { beforeEach, vi } from 'vitest'

const listWorkflowTemplatesMock = vi.fn()
const createWorkflowTemplateMock = vi.fn()
const patchWorkflowTemplateMock = vi.fn()
const publishWorkflowTemplateMock = vi.fn()
const listWorkflowRunsMock = vi.fn()
const createWorkflowRunMock = vi.fn()
const cancelWorkflowRunMock = vi.fn()
const listWorkflowStepRunsMock = vi.fn()

const listCapabilitiesMock = vi.fn()
const listAlgorithmsMock = vi.fn()
const listProvidersMock = vi.fn()

const runAlgorithmMock = vi.fn()

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

vi.mock('@/api/workflow', () => ({
  listWorkflowTemplates: (...args: unknown[]) => listWorkflowTemplatesMock(...args),
  createWorkflowTemplate: (...args: unknown[]) => createWorkflowTemplateMock(...args),
  patchWorkflowTemplate: (...args: unknown[]) => patchWorkflowTemplateMock(...args),
  publishWorkflowTemplate: (...args: unknown[]) => publishWorkflowTemplateMock(...args),
  listWorkflowRuns: (...args: unknown[]) => listWorkflowRunsMock(...args),
  createWorkflowRun: (...args: unknown[]) => createWorkflowRunMock(...args),
  cancelWorkflowRun: (...args: unknown[]) => cancelWorkflowRunMock(...args),
  listWorkflowStepRuns: (...args: unknown[]) => listWorkflowStepRunsMock(...args),
}))

vi.mock('@/api/registry', () => ({
  listCapabilities: (...args: unknown[]) => listCapabilitiesMock(...args),
  listAlgorithms: (...args: unknown[]) => listAlgorithmsMock(...args),
  listProviders: (...args: unknown[]) => listProvidersMock(...args),
}))

vi.mock('@/api/algorithms', () => ({
  runAlgorithm: (...args: unknown[]) => runAlgorithmMock(...args),
}))

const stubs = {
  WindowBoard: {
    props: ['routeKey', 'panes'],
    template:
      '<div><slot name="canvas-templates" /><slot name="canvas-runs" /><slot name="canvas-steps" /><slot name="canvas-registry" /></div>',
  },
  PageHeader: {
    props: ['title', 'subtitle'],
    template: '<header><h1>{{ title }}</h1><p>{{ subtitle }}</p><slot name="actions" /></header>',
  },
  SectionCard: {
    props: ['title', 'subtitle'],
    template: '<section><h2>{{ title }}</h2><p>{{ subtitle }}</p><slot /></section>',
  },
  WorkflowTemplatePane: {
    props: ['templates', 'selectedTemplateId', 'busy', 'state'],
    template: `
      <div>
        <button data-testid="create-template" @click="$emit('create', { name: 'wf-created' })">create template</button>
      </div>
    `,
  },
  WorkflowRunPane: {
    props: ['runs', 'selectedRunId', 'selectedTemplateId', 'busy', 'state'],
    template: `
      <div>
        <button data-testid="create-run" @click="$emit('create', { mode: 'sync' })">create run</button>
      </div>
    `,
  },
  WorkflowStepPane: {
    props: ['runId', 'steps', 'loading'],
    template: '<div data-testid="step-pane">{{ steps.length }}</div>',
  },
  RegistryPane: {
    props: ['capabilities', 'algorithms', 'providers', 'busy', 'loading'],
    template: `
      <div>
        <button data-testid="run-algorithm" @click="$emit('runAlgorithm', 'algo_1')">run algorithm</button>
      </div>
    `,
  },
  Button: { template: '<button><slot /></button>' },
  Icon: { template: '<span />' },
}

async function flushAll(): Promise<void> {
  await Promise.resolve()
  await nextTick()
  await Promise.resolve()
}

function buildTemplate(): WorkflowTemplateDTO {
  return {
    id: 'tpl_1',
    tenantId: 't1',
    workspaceId: 'w1',
    ownerId: 'u1',
    visibility: 'PRIVATE',
    acl: [],
    status: 'draft',
    createdAt: '2026-02-10T10:00:00Z',
    updatedAt: '2026-02-10T10:00:01Z',
    name: 'wf-1',
    description: 'workflow',
    graph: { nodes: [], edges: [] },
    schemaInputs: {},
    schemaOutputs: {},
    uiState: {},
    currentVersion: '',
  }
}

function buildRun(): WorkflowRunDTO {
  return {
    id: 'run_1',
    tenantId: 't1',
    workspaceId: 'w1',
    ownerId: 'u1',
    visibility: 'PRIVATE',
    acl: [],
    status: 'running',
    createdAt: '2026-02-10T10:00:00Z',
    updatedAt: '2026-02-10T10:00:01Z',
    templateId: 'tpl_1',
    templateVersion: 'v1',
    attempt: 1,
    traceId: 'trace-run',
    inputs: {},
    outputs: {},
    startedAt: '2026-02-10T10:00:00Z',
  }
}

function buildCapability(): CapabilityDTO {
  return {
    id: 'cap_1',
    tenantId: 't1',
    workspaceId: 'w1',
    ownerId: 'u1',
    visibility: 'PRIVATE',
    acl: [],
    status: 'enabled',
    createdAt: '2026-02-10T10:00:00Z',
    updatedAt: '2026-02-10T10:00:01Z',
    name: 'capability-1',
    kind: 'tool',
    version: '1.0.0',
    providerId: 'provider_1',
    inputSchema: {},
    outputSchema: {},
    requiredPermissions: [],
    egressPolicy: {},
  }
}

function buildAlgorithm(): AlgorithmDTO {
  return {
    id: 'algo_1',
    tenantId: 't1',
    workspaceId: 'w1',
    ownerId: 'u1',
    visibility: 'PRIVATE',
    acl: [],
    status: 'enabled',
    createdAt: '2026-02-10T10:00:00Z',
    updatedAt: '2026-02-10T10:00:01Z',
    name: 'algorithm-1',
    version: '1.0.0',
    templateRef: 'tpl_1',
    defaults: {},
    constraints: {},
    dependencies: {},
  }
}

function buildProvider(): ProviderDTO {
  return {
    id: 'provider_1',
    tenantId: 't1',
    workspaceId: 'w1',
    ownerId: 'u1',
    visibility: 'PRIVATE',
    acl: [],
    status: 'enabled',
    createdAt: '2026-02-10T10:00:00Z',
    updatedAt: '2026-02-10T10:00:01Z',
    name: 'provider-1',
    providerType: 'http',
    endpoint: 'http://localhost',
    metadata: {},
  }
}

describe('CanvasView', () => {
  beforeEach(() => {
    listWorkflowTemplatesMock.mockReset()
    createWorkflowTemplateMock.mockReset()
    patchWorkflowTemplateMock.mockReset()
    publishWorkflowTemplateMock.mockReset()
    listWorkflowRunsMock.mockReset()
    createWorkflowRunMock.mockReset()
    cancelWorkflowRunMock.mockReset()
    listWorkflowStepRunsMock.mockReset()
    listCapabilitiesMock.mockReset()
    listAlgorithmsMock.mockReset()
    listProvidersMock.mockReset()
    runAlgorithmMock.mockReset()

    listWorkflowTemplatesMock.mockResolvedValue({
      items: [buildTemplate()],
      pageInfo: { page: 1, pageSize: 20, total: 1 },
    })
    listWorkflowRunsMock.mockResolvedValue({
      items: [buildRun()],
      pageInfo: { page: 1, pageSize: 20, total: 1 },
    })
    listWorkflowStepRunsMock.mockResolvedValue({
      items: [],
      pageInfo: { page: 1, pageSize: 20, total: 0 },
    })
    listCapabilitiesMock.mockResolvedValue({
      items: [buildCapability()],
      pageInfo: { page: 1, pageSize: 20, total: 1 },
    })
    listAlgorithmsMock.mockResolvedValue({
      items: [buildAlgorithm()],
      pageInfo: { page: 1, pageSize: 20, total: 1 },
    })
    listProvidersMock.mockResolvedValue({
      items: [buildProvider()],
      pageInfo: { page: 1, pageSize: 20, total: 1 },
    })

    createWorkflowTemplateMock.mockResolvedValue({
      resource: buildTemplate(),
      commandRef: { commandId: 'cmd_create_tpl', status: 'accepted', acceptedAt: '2026-02-10T10:00:00Z' },
    })
    createWorkflowRunMock.mockResolvedValue({
      resource: buildRun(),
      commandRef: { commandId: 'cmd_create_run', status: 'accepted', acceptedAt: '2026-02-10T10:00:00Z' },
    })
    runAlgorithmMock.mockResolvedValue({
      resource: {
        id: 'algo_run_1',
        algorithmId: 'algo_1',
        workflowRunId: 'run_1',
        status: 'succeeded',
        createdAt: '2026-02-10T10:00:00Z',
        updatedAt: '2026-02-10T10:00:01Z',
      },
      commandRef: { commandId: 'cmd_algo_run', status: 'accepted', acceptedAt: '2026-02-10T10:00:00Z' },
    })
  })

  it('loads runtime data and executes template/algorithm actions', async () => {
    const wrapper = mount(CanvasView, {
      global: {
        plugins: [i18n],
        stubs,
      },
    })

    await flushAll()

    expect(listWorkflowTemplatesMock).toHaveBeenCalled()
    expect(listWorkflowRunsMock).toHaveBeenCalled()
    expect(listAlgorithmsMock).toHaveBeenCalled()

    await wrapper.get('[data-testid="create-template"]').trigger('click')
    expect(createWorkflowTemplateMock).toHaveBeenCalledTimes(1)

    await wrapper.get('[data-testid="create-run"]').trigger('click')
    expect(createWorkflowRunMock).toHaveBeenCalledTimes(1)

    await wrapper.get('[data-testid="run-algorithm"]').trigger('click')
    expect(runAlgorithmMock).toHaveBeenCalledTimes(1)
  })
})
