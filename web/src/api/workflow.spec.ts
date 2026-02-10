import { beforeEach, describe, expect, it, vi } from 'vitest'
import {
  cancelWorkflowRun,
  createWorkflowRun,
  createWorkflowTemplate,
  listWorkflowRuns,
  listWorkflowStepRuns,
  listWorkflowTemplates,
  patchWorkflowTemplate,
  publishWorkflowTemplate,
} from '@/api/workflow'

const apiRequestMock = vi.fn()

vi.mock('@/api/http', () => ({
  apiRequest: (...args: unknown[]) => apiRequestMock(...args),
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
    await createWorkflowRun({ templateId: 'tpl_1', mode: 'sync', inputs: {} }, 'idem-run-create')
    await cancelWorkflowRun('run_1', 'idem-run-cancel')
    await listWorkflowStepRuns('run_1', { page: 1, pageSize: 20 })

    expect(apiRequestMock.mock.calls[0]?.[0]).toBe('/workflow-runs')
    expect(apiRequestMock.mock.calls[1]?.[0]).toBe('/workflow-runs')
    expect(apiRequestMock.mock.calls[2]?.[0]).toBe('/workflow-runs/run_1:cancel')
    expect(apiRequestMock.mock.calls[3]?.[0]).toBe('/workflow-runs/run_1/steps')
  })
})
