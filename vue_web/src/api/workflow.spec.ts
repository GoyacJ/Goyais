/**
 * SPDX-License-Identifier: Apache-2.0
 * Copyright (c) 2026 Goya
 * Author: Goya
 * Created: 2026-02-11
 * Version: v1.0.0
 * Description: Goyais source file.
 */

import { beforeEach, describe, expect, it, vi } from 'vitest'
import {
  cancelWorkflowRun,
  createWorkflowRun,
  createWorkflowTemplate,
  getWorkflowRunEvents,
  listWorkflowRuns,
  listWorkflowStepRuns,
  listWorkflowTemplates,
  patchWorkflowTemplate,
  publishWorkflowTemplate,
} from '@/api/workflow'

const apiRequestMock = vi.fn()

vi.mock('@/api/http', () => ({
  apiRequest: (...args: unknown[]) => apiRequestMock(...args),
  ApiHttpError: class ApiHttpError extends Error {
    status: number
    error: { code: string; messageKey: string; details?: Record<string, unknown> }
    constructor(status: number, error: { code: string; messageKey: string; details?: Record<string, unknown> }) {
      super(error.messageKey)
      this.name = 'ApiHttpError'
      this.status = status
      this.error = error
    }
  },
  getApiRuntimeConfig: () => ({
    apiBaseUrl: '/api/v1',
    tenantId: 'tenant-demo',
    workspaceId: 'workspace-demo',
    userId: 'user-demo',
    roles: 'admin',
    policyVersion: 'v1',
  }),
}))

describe('workflow api', () => {
  beforeEach(() => {
    apiRequestMock.mockReset()
    apiRequestMock.mockResolvedValue({})
  })

  it('calls workflow template endpoints with expected paths', async () => {
    await listWorkflowTemplates({ page: 2, pageSize: 50 })
    await createWorkflowTemplate({ name: 'wf-1', graph: { nodes: [], edges: [] } }, 'idem-template-create')
    await patchWorkflowTemplate('tpl_1', { graph: { nodes: [], edges: [] } }, 'idem-template-patch')
    await publishWorkflowTemplate('tpl_1', 'idem-template-publish')

    expect(apiRequestMock.mock.calls[0]?.[0]).toBe('/workflow-templates')
    expect(apiRequestMock.mock.calls[1]?.[0]).toBe('/workflow-templates')
    expect(apiRequestMock.mock.calls[2]?.[0]).toBe('/workflow-templates/tpl_1:patch')
    expect(apiRequestMock.mock.calls[3]?.[0]).toBe('/workflow-templates/tpl_1:publish')
  })

  it('calls workflow run endpoints with expected paths', async () => {
    await listWorkflowRuns({ cursor: 'cursor_1' })
    await createWorkflowRun(
      { templateId: 'tpl_1', mode: 'sync', inputs: {}, fromStepKey: 'n2', testNode: true },
      'idem-run-create',
    )
    await cancelWorkflowRun('run_1', 'idem-run-cancel')
    await listWorkflowStepRuns('run_1', { page: 1, pageSize: 20 })

    expect(apiRequestMock.mock.calls[0]?.[0]).toBe('/workflow-runs')
    expect(apiRequestMock.mock.calls[1]?.[0]).toBe('/workflow-runs')
    expect(apiRequestMock.mock.calls[1]?.[1]).toMatchObject({
      body: {
        templateId: 'tpl_1',
        fromStepKey: 'n2',
        testNode: true,
      },
    })
    expect(apiRequestMock.mock.calls[2]?.[0]).toBe('/workflow-runs/run_1:cancel')
    expect(apiRequestMock.mock.calls[3]?.[0]).toBe('/workflow-runs/run_1/steps')
  })

  it('parses sse payload from workflow run events endpoint', async () => {
    const fetchMock = vi.fn().mockResolvedValue({
      ok: true,
      status: 200,
      text: () =>
        Promise.resolve(
          [
            'id: evt_1',
            'event: workflow.step.running',
            'data: {"runId":"run_1","stepKey":"s1"}',
            '',
            'id: evt_2',
            'event: workflow.run.succeeded',
            'data: {"runId":"run_1","status":"succeeded"}',
            '',
          ].join('\n'),
        ),
    })
    const originalFetch = globalThis.fetch
    globalThis.fetch = fetchMock as unknown as typeof globalThis.fetch

    try {
      const events = await getWorkflowRunEvents('run_1')
      expect(fetchMock).toHaveBeenCalledWith('/api/v1/workflow-runs/run_1/events', expect.any(Object))
      expect(events).toHaveLength(2)
      expect(events[0]?.event).toBe('workflow.step.running')
      expect(events[1]?.event).toBe('workflow.run.succeeded')
    } finally {
      globalThis.fetch = originalFetch
    }
  })
})
