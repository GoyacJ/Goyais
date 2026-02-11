import { ApiHttpError, apiRequest, getApiRuntimeConfig } from '@/api/http'
import type {
  AISessionCreateRequest,
  AISessionDTO,
  AISessionTurnCreateRequest,
  AISessionTurnDTO,
  ApiError,
  ApiErrorEnvelope,
  ListResponse,
  ResourceSnapshot,
  WorkflowListParams,
  WriteResponse,
} from '@/api/types'

export interface AISessionEvent {
  id?: string
  event?: string
  data: AISessionTurnDTO | Record<string, unknown> | string | null
}

export async function listAISessions(params: WorkflowListParams = {}): Promise<ListResponse<AISessionDTO>> {
  return apiRequest<ListResponse<AISessionDTO>>('/ai/sessions', {
    method: 'GET',
    query: {
      cursor: params.cursor,
      page: params.page,
      pageSize: params.pageSize,
    },
  })
}

export async function getAISession(sessionId: string): Promise<AISessionDTO> {
  return apiRequest<AISessionDTO>(`/ai/sessions/${encodeURIComponent(sessionId)}`, {
    method: 'GET',
  })
}

export async function createAISession(
  request: AISessionCreateRequest,
  idempotencyKey?: string,
): Promise<WriteResponse<ResourceSnapshot>> {
  return apiRequest<WriteResponse<ResourceSnapshot>>('/ai/sessions', {
    method: 'POST',
    body: request,
    idempotencyKey,
  })
}

export async function archiveAISession(
  sessionId: string,
  idempotencyKey?: string,
): Promise<WriteResponse<ResourceSnapshot>> {
  return apiRequest<WriteResponse<ResourceSnapshot>>(`/ai/sessions/${encodeURIComponent(sessionId)}:archive`, {
    method: 'POST',
    body: {},
    idempotencyKey,
  })
}

export async function createAISessionTurn(
  sessionId: string,
  request: AISessionTurnCreateRequest,
  idempotencyKey?: string,
): Promise<WriteResponse<ResourceSnapshot>> {
  return apiRequest<WriteResponse<ResourceSnapshot>>(`/ai/sessions/${encodeURIComponent(sessionId)}/turns`, {
    method: 'POST',
    body: request,
    idempotencyKey,
  })
}

export async function getAISessionEvents(sessionId: string): Promise<AISessionEvent[]> {
  const cfg = getApiRuntimeConfig()
  const url = `${cfg.apiBaseUrl}/ai/sessions/${encodeURIComponent(sessionId)}/events`
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

function parseSSE(raw: string): AISessionEvent[] {
  const events: AISessionEvent[] = []
  let currentID = ''
  let currentEvent = ''
  const dataLines: string[] = []

  const flush = (): void => {
    if (!currentID && !currentEvent && dataLines.length === 0) {
      return
    }

    const dataRaw = dataLines.join('\n').trim()
    let data: AISessionEvent['data'] = null
    if (dataRaw.length > 0) {
      try {
        data = JSON.parse(dataRaw) as AISessionTurnDTO | Record<string, unknown>
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
