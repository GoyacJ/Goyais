import { mount } from '@vue/test-utils'
import { nextTick } from 'vue'
import { beforeEach, describe, expect, it, vi } from 'vitest'

import type { StepRunDTO, WorkflowRunDTO, WorkflowTemplateDTO } from '@/api/types'
import i18n from '@/i18n'
import CanvasView from '@/views/CanvasView.vue'

const listWorkflowTemplatesMock = vi.fn()
const listWorkflowRunsMock = vi.fn()
const getWorkflowRunMock = vi.fn()
const listWorkflowStepRunsMock = vi.fn()
const getWorkflowTemplateMock = vi.fn()

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
  listWorkflowRuns: (...args: unknown[]) => listWorkflowRunsMock(...args),
  getWorkflowRun: (...args: unknown[]) => getWorkflowRunMock(...args),
  listWorkflowStepRuns: (...args: unknown[]) => listWorkflowStepRunsMock(...args),
  getWorkflowTemplate: (...args: unknown[]) => getWorkflowTemplateMock(...args),
  cancelWorkflowRun: vi.fn(),
  createWorkflowRun: vi.fn(),
  createWorkflowTemplate: vi.fn(),
  patchWorkflowTemplate: vi.fn(),
  publishWorkflowTemplate: vi.fn(),
}))

vi.mock('@vue-flow/core', () => ({
  ConnectionMode: { Loose: 'Loose' },
  VueFlow: {
    template: '<div class="vue-flow-stub"><slot /></div>',
  },
}))

vi.mock('@vue-flow/background', () => ({
  Background: { template: '<div />' },
}))

vi.mock('@vue-flow/minimap', () => ({
  MiniMap: { template: '<div />' },
}))

vi.mock('@vue-flow/controls', () => ({
  Controls: { template: '<div />' },
}))

const stubs = {
  WindowBoard: {
    props: ['routeKey', 'panes'],
    template: '<div><slot name="canvas-templates" /><slot name="canvas-board" /><slot name="canvas-inspector" /></div>',
  },
  PageHeader: {
    props: ['title', 'subtitle'],
    template: '<header><h1>{{ title }}</h1><p>{{ subtitle }}</p><slot name="actions" /></header>',
  },
  SectionCard: {
    props: ['title', 'subtitle'],
    template: '<section><h2>{{ title }}</h2><p>{{ subtitle }}</p><slot /></section>',
  },
  Button: {
    emits: ['click'],
    template: '<button @click="$emit(\'click\')"><slot /></button>',
  },
  Input: {
    props: ['modelValue'],
    emits: ['update:modelValue'],
    template: '<input :value="modelValue" @input="$emit(\'update:modelValue\', $event.target && $event.target.value ? $event.target.value : \'\')" />',
  },
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
    createdAt: '2026-02-11T10:00:00Z',
    updatedAt: '2026-02-11T10:00:01Z',
    name: 'tpl-1',
    description: 'template',
    graph: {
      nodes: [
        {
          id: 'step_1',
          type: 'source.text',
          position: { x: 120, y: 80 },
          data: { label: 'Step 1', inputType: 'none', outputType: 'text', nodeType: 'source.text' },
        },
      ],
      edges: [],
    },
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
    status: 'succeeded',
    createdAt: '2026-02-11T10:00:00Z',
    updatedAt: '2026-02-11T10:00:02Z',
    templateId: 'tpl_1',
    templateVersion: 'v1',
    attempt: 1,
    traceId: 'trace-run-1',
    inputs: {},
    outputs: {},
    startedAt: '2026-02-11T10:00:00Z',
    finishedAt: '2026-02-11T10:00:02Z',
    durationMs: 2000,
  }
}

function buildStep(): StepRunDTO {
  return {
    id: 'step_run_1',
    runId: 'run_1',
    tenantId: 't1',
    workspaceId: 'w1',
    ownerId: 'u1',
    visibility: 'PRIVATE',
    acl: [],
    status: 'failed',
    createdAt: '2026-02-11T10:00:00Z',
    updatedAt: '2026-02-11T10:00:01Z',
    stepKey: 'step_1',
    stepType: 'source.text',
    attempt: 1,
    traceId: 'trace-step-1',
    input: {},
    output: {},
    artifacts: { assets: ['asset_1', 'asset_2'] },
    logRef: 'log://step-1',
    startedAt: '2026-02-11T10:00:00Z',
    finishedAt: '2026-02-11T10:00:01Z',
    durationMs: 120,
    error: {
      code: 'E_STEP',
      messageKey: 'error.workflow.step_failed',
    },
  }
}

async function flushAll(): Promise<void> {
  await Promise.resolve()
  await nextTick()
  await Promise.resolve()
  await nextTick()
}

describe('CanvasView', () => {
  beforeEach(() => {
    listWorkflowTemplatesMock.mockReset()
    listWorkflowRunsMock.mockReset()
    getWorkflowRunMock.mockReset()
    listWorkflowStepRunsMock.mockReset()
    getWorkflowTemplateMock.mockReset()

    const tpl = buildTemplate()
    const run = buildRun()
    const step = buildStep()
    listWorkflowTemplatesMock.mockResolvedValue({
      items: [tpl],
      pageInfo: { page: 1, pageSize: 20, total: 1 },
    })
    listWorkflowRunsMock.mockResolvedValue({
      items: [run],
      pageInfo: { page: 1, pageSize: 20, total: 1 },
    })
    getWorkflowRunMock.mockResolvedValue(run)
    getWorkflowTemplateMock.mockResolvedValue(tpl)
    listWorkflowStepRunsMock.mockResolvedValue({
      items: [step],
      pageInfo: { page: 1, pageSize: 20, total: 1 },
    })
  })

  it('renders node runtime details after selecting a run', async () => {
    const wrapper = mount(CanvasView, {
      global: {
        plugins: [i18n],
        stubs,
      },
    })
    await flushAll()

    const runButton = wrapper.findAll('button').find((item) => item.text().includes('run_1'))
    expect(runButton).toBeDefined()
    await runButton!.trigger('click')
    await flushAll()

    expect(listWorkflowStepRunsMock).toHaveBeenCalled()
    const runtimeCard = wrapper.find('.canvas-node-runtime')
    expect(runtimeCard.exists()).toBe(true)
    expect(runtimeCard.text()).toContain('120')
    expect(runtimeCard.text()).toContain('2')
    expect(runtimeCard.text()).toContain('E_STEP')
  })
})
