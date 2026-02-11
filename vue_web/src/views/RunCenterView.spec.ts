/**
 * SPDX-License-Identifier: Apache-2.0
 * Copyright (c) 2026 Goya
 * Author: Goya
 * Created: 2026-02-11
 * Version: v1.0.0
 * Description: Goyais source file.
 */

import { mount } from '@vue/test-utils'
import { nextTick } from 'vue'
import { beforeEach, describe, expect, it, vi } from 'vitest'

import i18n from '@/i18n'
import RunCenterView from '@/views/RunCenterView.vue'

const listWorkflowRunsMock = vi.fn()
const getWorkflowRunMock = vi.fn()
const listWorkflowStepRunsMock = vi.fn()
const getWorkflowRunEventsMock = vi.fn()

vi.mock('@/api/http', () => ({
  ApiHttpError: class ApiHttpError extends Error {
    readonly status: number
    readonly error: { code: string; messageKey: string }

    constructor(status: number, error: { code: string; messageKey: string }) {
      super(error.messageKey)
      this.status = status
      this.error = error
    }
  },
  getApiRuntimeConfig: () => ({
    apiBaseUrl: '/api/v1',
    tenantId: 't1',
    workspaceId: 'w1',
    userId: 'u1',
    roles: 'admin',
    policyVersion: 'v1',
  }),
}))

vi.mock('@/api/workflow', () => ({
  listWorkflowRuns: (...args: unknown[]) => listWorkflowRunsMock(...args),
  getWorkflowRun: (...args: unknown[]) => getWorkflowRunMock(...args),
  listWorkflowStepRuns: (...args: unknown[]) => listWorkflowStepRunsMock(...args),
  getWorkflowRunEvents: (...args: unknown[]) => getWorkflowRunEventsMock(...args),
}))

const stubs = {
  WindowBoard: {
    props: ['routeKey', 'panes'],
    template: '<div><slot name="run-center-list" /><slot name="run-center-detail" /></div>',
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
  Icon: { template: '<span />' },
  Table: {
    props: ['rows', 'interactiveRows'],
    emits: ['row-click'],
    template:
      '<div><button v-for="row in rows" :key="row.id" :data-row-key="row.id" @click="interactiveRows && $emit(\'row-click\', { rowKey: row.id })">{{ row.id }}</button></div>',
  },
}

async function flushAll(): Promise<void> {
  await Promise.resolve()
  await nextTick()
  await Promise.resolve()
  await nextTick()
}

describe('RunCenterView', () => {
  beforeEach(() => {
    listWorkflowRunsMock.mockReset()
    getWorkflowRunMock.mockReset()
    listWorkflowStepRunsMock.mockReset()
    getWorkflowRunEventsMock.mockReset()

    listWorkflowRunsMock.mockResolvedValue({
      items: [
        {
          id: 'run_1',
          tenantId: 't1',
          workspaceId: 'w1',
          ownerId: 'u1',
          visibility: 'PRIVATE',
          acl: [],
          status: 'succeeded',
          createdAt: '2026-02-11T00:00:00Z',
          updatedAt: '2026-02-11T00:00:10Z',
          templateId: 'tpl_1',
          templateVersion: 'v1',
          attempt: 1,
          inputs: { imageId: 'asset_in' },
          outputs: { assetId: 'asset_out' },
          startedAt: '2026-02-11T00:00:00Z',
          finishedAt: '2026-02-11T00:00:10Z',
          durationMs: 10000,
        },
      ],
      pageInfo: { page: 1, pageSize: 20, total: 1 },
    })

    getWorkflowRunMock.mockResolvedValue({
      id: 'run_1',
      tenantId: 't1',
      workspaceId: 'w1',
      ownerId: 'u1',
      visibility: 'PRIVATE',
      acl: [],
      status: 'succeeded',
      createdAt: '2026-02-11T00:00:00Z',
      updatedAt: '2026-02-11T00:00:10Z',
      templateId: 'tpl_1',
      templateVersion: 'v1',
      attempt: 1,
      inputs: { imageId: 'asset_in' },
      outputs: { assetId: 'asset_out' },
      startedAt: '2026-02-11T00:00:00Z',
      finishedAt: '2026-02-11T00:00:10Z',
      durationMs: 10000,
      traceId: 'trace_run_1',
    })

    listWorkflowStepRunsMock.mockResolvedValue({
      items: [
        {
          id: 'step_2',
          runId: 'run_1',
          tenantId: 't1',
          workspaceId: 'w1',
          ownerId: 'u1',
          visibility: 'PRIVATE',
          acl: [],
          status: 'succeeded',
          createdAt: '2026-02-11T00:00:05Z',
          updatedAt: '2026-02-11T00:00:10Z',
          stepKey: 'encode',
          stepType: 'output',
          attempt: 1,
          traceId: 'trace_step_2',
          input: { codec: 'h264' },
          output: { stream: 'rtmp://example' },
          artifacts: {
            assetIds: ['asset_out_1'],
            previewUrl: 'https://cdn.example.com/output.mp4',
          },
          logRef: 'https://logs.example.com/run_1/step_2.log',
          startedAt: '2026-02-11T00:00:05Z',
          finishedAt: '2026-02-11T00:00:10Z',
          durationMs: 5000,
        },
        {
          id: 'step_1',
          runId: 'run_1',
          tenantId: 't1',
          workspaceId: 'w1',
          ownerId: 'u1',
          visibility: 'PRIVATE',
          acl: [],
          status: 'succeeded',
          createdAt: '2026-02-11T00:00:00Z',
          updatedAt: '2026-02-11T00:00:05Z',
          stepKey: 'decode',
          stepType: 'input',
          attempt: 1,
          traceId: 'trace_step_1',
          input: { source: 'asset_in' },
          output: { frameCount: 120 },
          artifacts: {},
          startedAt: '2026-02-11T00:00:00Z',
          finishedAt: '2026-02-11T00:00:05Z',
          durationMs: 5000,
        },
      ],
      pageInfo: { page: 1, pageSize: 20, total: 2 },
    })

    getWorkflowRunEventsMock.mockResolvedValue([
      {
        id: 'evt_1',
        event: 'workflow.run.started',
        data: {
          runId: 'run_1',
          status: 'running',
          createdAt: '2026-02-11T00:00:00Z',
        },
      },
    ])
  })

  it('renders step detail with actionable references', async () => {
    const wrapper = mount(RunCenterView, {
      global: {
        plugins: [i18n],
        stubs,
      },
    })

    await flushAll()

    expect(listWorkflowRunsMock).toHaveBeenCalledWith({ page: 1, pageSize: 200 })
    expect(getWorkflowRunMock).toHaveBeenCalledWith('run_1')
    expect(getWorkflowRunEventsMock).toHaveBeenCalledWith('run_1')

    const text = wrapper.text()
    expect(text).toContain('https://logs.example.com/run_1/step_2.log')
    expect(text).toContain('asset_out_1')
    expect(text).toContain('https://cdn.example.com/output.mp4')
  })
})
