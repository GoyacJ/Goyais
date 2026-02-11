/**
 * SPDX-License-Identifier: Apache-2.0
 * Copyright (c) 2026 Goya
 * Author: Goya
 * Created: 2026-02-11
 * Version: v1.0.0
 * Description: Goyais source file.
 */

import { ApiHttpError, apiRequest, getApiRuntimeConfig } from '@/api/http'
import type {
  ApiError,
  ApiErrorEnvelope,
  ListResponse,
  StepRunDTO,
  WorkflowRunEventDTO,
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

export async function getWorkflowRunEvents(runId: string): Promise<WorkflowRunEventDTO[]> {
  const cfg = getApiRuntimeConfig()
  const url = `${cfg.apiBaseUrl}/workflow-runs/${encodeURIComponent(runId)}/events`
  const headers = new Headers({
    'X-Tenant-Id': cfg.tenantId,
    'X-Workspace-Id': cfg.workspaceId,
    'X-User-Id': cfg.userId,
    'X-Roles': cfg.roles,
    'X-Policy-Version': cfg.policyVersion,
  })

  let response: Response
  try {
    response = await fetch(url, { method: 'GET', headers })
  } catch (error) {
    const apiError: ApiError = {
      code: 'NETWORK_ERROR',
      messageKey: 'error.common.internal',
      details: { reason: error instanceof Error ? error.message : 'fetch_failed' },
    }
    throw new ApiHttpError(0, apiError)
  }

  const raw = await response.text()
  if (!response.ok) {
    let apiError: ApiError = {
      code: `HTTP_${response.status}`,
      messageKey: 'error.common.internal',
    }
    try {
      const parsed = JSON.parse(raw) as ApiErrorEnvelope
      if (parsed?.error?.code && parsed?.error?.messageKey) {
        apiError = parsed.error
      }
    } catch {
      // keep default error
    }
    throw new ApiHttpError(response.status, apiError)
  }

  return parseSSE(raw)
}

function parseSSE(raw: string): WorkflowRunEventDTO[] {
  const events: WorkflowRunEventDTO[] = []
  let currentID = ''
  let currentEvent = ''
  const dataLines: string[] = []

  const flush = (): void => {
    if (!currentID && !currentEvent && dataLines.length === 0) {
      return
    }
    const dataRaw = dataLines.join('\n').trim()
    let data: WorkflowRunEventDTO['data'] = null
    if (dataRaw.length > 0) {
      try {
        data = JSON.parse(dataRaw) as Record<string, unknown>
      } catch {
        data = dataRaw
      }
    }
    events.push({
      id: currentID || undefined,
      event: currentEvent || undefined,
      data,
    })
    currentID = ''
    currentEvent = ''
    dataLines.length = 0
  }

  for (const rawLine of raw.split(/\r?\n/)) {
    const line = rawLine.trimEnd()
    if (line.length === 0) {
      flush()
      continue
    }
    if (line.startsWith('id:')) {
      currentID = line.slice(3).trim()
      continue
    }
    if (line.startsWith('event:')) {
      currentEvent = line.slice(6).trim()
      continue
    }
    if (line.startsWith('data:')) {
      dataLines.push(line.slice(5).trim())
    }
  }
  flush()

  return events
}
