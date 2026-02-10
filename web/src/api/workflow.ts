import { apiRequest } from '@/api/http'
import type {
  ListResponse,
  StepRunDTO,
  WorkflowListParams,
  WorkflowRunCreateRequest,
  WorkflowRunDTO,
  WorkflowTemplateCreateRequest,
  WorkflowTemplateDTO,
  WorkflowTemplatePatchRequest,
  WriteResponse,
} from '@/api/types'

export async function listWorkflowTemplates(params: WorkflowListParams = {}): Promise<ListResponse<WorkflowTemplateDTO>> {
  return apiRequest<ListResponse<WorkflowTemplateDTO>>('/workflow-templates', {
    method: 'GET',
    query: {
      cursor: params.cursor,
      page: params.page,
      pageSize: params.pageSize,
    },
  })
}

export async function getWorkflowTemplate(templateId: string): Promise<WorkflowTemplateDTO> {
  return apiRequest<WorkflowTemplateDTO>(`/workflow-templates/${encodeURIComponent(templateId)}`, {
    method: 'GET',
  })
}

export async function createWorkflowTemplate(
  request: WorkflowTemplateCreateRequest,
  idempotencyKey?: string,
): Promise<WriteResponse<WorkflowTemplateDTO>> {
  return apiRequest<WriteResponse<WorkflowTemplateDTO>>('/workflow-templates', {
    method: 'POST',
    body: request,
    idempotencyKey,
  })
}

export async function patchWorkflowTemplate(
  templateId: string,
  request: WorkflowTemplatePatchRequest,
  idempotencyKey?: string,
): Promise<WriteResponse<WorkflowTemplateDTO>> {
  return apiRequest<WriteResponse<WorkflowTemplateDTO>>(`/workflow-templates/${encodeURIComponent(templateId)}:patch`, {
    method: 'POST',
    body: request,
    idempotencyKey,
  })
}

export async function publishWorkflowTemplate(
  templateId: string,
  idempotencyKey?: string,
): Promise<WriteResponse<WorkflowTemplateDTO>> {
  return apiRequest<WriteResponse<WorkflowTemplateDTO>>(`/workflow-templates/${encodeURIComponent(templateId)}:publish`, {
    method: 'POST',
    body: {},
    idempotencyKey,
  })
}

export async function listWorkflowRuns(params: WorkflowListParams = {}): Promise<ListResponse<WorkflowRunDTO>> {
  return apiRequest<ListResponse<WorkflowRunDTO>>('/workflow-runs', {
    method: 'GET',
    query: {
      cursor: params.cursor,
      page: params.page,
      pageSize: params.pageSize,
    },
  })
}

export async function getWorkflowRun(runId: string): Promise<WorkflowRunDTO> {
  return apiRequest<WorkflowRunDTO>(`/workflow-runs/${encodeURIComponent(runId)}`, {
    method: 'GET',
  })
}

export async function createWorkflowRun(
  request: WorkflowRunCreateRequest,
  idempotencyKey?: string,
): Promise<WriteResponse<WorkflowRunDTO>> {
  return apiRequest<WriteResponse<WorkflowRunDTO>>('/workflow-runs', {
    method: 'POST',
    body: request,
    idempotencyKey,
  })
}

export async function cancelWorkflowRun(runId: string, idempotencyKey?: string): Promise<WriteResponse<WorkflowRunDTO>> {
  return apiRequest<WriteResponse<WorkflowRunDTO>>(`/workflow-runs/${encodeURIComponent(runId)}:cancel`, {
    method: 'POST',
    body: {},
    idempotencyKey,
  })
}

export async function listWorkflowStepRuns(
  runId: string,
  params: WorkflowListParams = {},
): Promise<ListResponse<StepRunDTO>> {
  return apiRequest<ListResponse<StepRunDTO>>(`/workflow-runs/${encodeURIComponent(runId)}/steps`, {
    method: 'GET',
    query: {
      cursor: params.cursor,
      page: params.page,
      pageSize: params.pageSize,
    },
  })
}
