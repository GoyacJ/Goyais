import type { ApiError, ApiErrorEnvelope } from '@/api/types'

const DEFAULT_API_BASE_URL = '/api/v1'
const DEFAULT_TENANT_ID = 't1'
const DEFAULT_WORKSPACE_ID = 'w1'
const DEFAULT_USER_ID = 'u1'
const DEFAULT_ROLES = 'member'
const DEFAULT_POLICY_VERSION = 'v0.1'

export interface ApiRuntimeConfig {
  apiBaseUrl: string
  tenantId: string
  workspaceId: string
  userId: string
  roles: string
  policyVersion: string
  mockEnabled: boolean
}

export interface ApiRequestOptions {
  method?: 'GET' | 'POST' | 'PATCH' | 'DELETE'
  query?: Record<string, string | number | undefined>
  body?: BodyInit | unknown
  headers?: Record<string, string>
  idempotencyKey?: string
}

export class ApiHttpError extends Error {
  readonly status: number
  readonly error: ApiError

  constructor(status: number, error: ApiError) {
    super(`${error.code}: ${error.messageKey}`)
    this.name = 'ApiHttpError'
    this.status = status
    this.error = error
  }
}

function readEnv(key: keyof ImportMetaEnv, fallback: string): string {
  const value = import.meta.env[key]
  if (typeof value !== 'string') {
    return fallback
  }

  const trimmed = value.trim()
  return trimmed.length > 0 ? trimmed : fallback
}

function parseBoolean(raw: string | undefined): boolean {
  if (!raw) {
    return false
  }

  const value = raw.trim().toLowerCase()
  return value === '1' || value === 'true' || value === 'yes' || value === 'on'
}

function trimTrailingSlash(value: string): string {
  if (value.length > 1 && value.endsWith('/')) {
    return value.slice(0, -1)
  }

  return value
}

function normalizePath(path: string): string {
  if (path.startsWith('/')) {
    return path
  }

  return `/${path}`
}

function buildQueryString(query?: Record<string, string | number | undefined>): string {
  if (!query) {
    return ''
  }

  const params = new URLSearchParams()
  for (const [key, value] of Object.entries(query)) {
    if (value === undefined || value === null) {
      continue
    }
    params.set(key, String(value))
  }

  const serialized = params.toString()
  return serialized.length > 0 ? `?${serialized}` : ''
}

function isApiErrorEnvelope(payload: unknown): payload is ApiErrorEnvelope {
  if (!payload || typeof payload !== 'object') {
    return false
  }

  const envelope = payload as Record<string, unknown>
  if (!envelope.error || typeof envelope.error !== 'object') {
    return false
  }

  const error = envelope.error as Record<string, unknown>
  return typeof error.code === 'string' && typeof error.messageKey === 'string'
}

function isBodyInitLike(value: unknown): value is BodyInit {
  if (typeof value === 'string') {
    return true
  }
  if (value instanceof Blob || value instanceof FormData || value instanceof URLSearchParams) {
    return true
  }
  if (value instanceof ArrayBuffer || ArrayBuffer.isView(value)) {
    return true
  }
  if (typeof ReadableStream !== 'undefined' && value instanceof ReadableStream) {
    return true
  }
  return false
}

async function readResponseBody(response: Response): Promise<unknown | undefined> {
  const contentType = response.headers.get('content-type') ?? ''
  if (!contentType.includes('application/json')) {
    return undefined
  }

  try {
    return await response.json()
  } catch {
    return undefined
  }
}

function defaultApiError(status: number, details?: Record<string, unknown>): ApiError {
  const code = status > 0 ? `HTTP_${status}` : 'NETWORK_ERROR'
  return {
    code,
    messageKey: 'error.common.internal',
    details,
  }
}

const runtimeConfig: ApiRuntimeConfig = {
  apiBaseUrl: trimTrailingSlash(readEnv('VITE_GOYAIS_API_BASE_URL', DEFAULT_API_BASE_URL)),
  tenantId: readEnv('VITE_GOYAIS_TENANT_ID', DEFAULT_TENANT_ID),
  workspaceId: readEnv('VITE_GOYAIS_WORKSPACE_ID', DEFAULT_WORKSPACE_ID),
  userId: readEnv('VITE_GOYAIS_USER_ID', DEFAULT_USER_ID),
  roles: readEnv('VITE_GOYAIS_ROLES', DEFAULT_ROLES),
  policyVersion: readEnv('VITE_GOYAIS_POLICY_VERSION', DEFAULT_POLICY_VERSION),
  mockEnabled: parseBoolean(import.meta.env.VITE_GOYAIS_USE_MOCK),
}

export function getApiRuntimeConfig(): ApiRuntimeConfig {
  return { ...runtimeConfig }
}

export function isMockEnabled(): boolean {
  return runtimeConfig.mockEnabled
}

export async function apiRequest<T>(path: string, options: ApiRequestOptions = {}): Promise<T> {
  const method = options.method ?? 'GET'
  const url = `${runtimeConfig.apiBaseUrl}${normalizePath(path)}${buildQueryString(options.query)}`

  const headers = new Headers(options.headers)
  headers.set('X-Tenant-Id', runtimeConfig.tenantId)
  headers.set('X-Workspace-Id', runtimeConfig.workspaceId)
  headers.set('X-User-Id', runtimeConfig.userId)
  headers.set('X-Roles', runtimeConfig.roles)
  headers.set('X-Policy-Version', runtimeConfig.policyVersion)

  const isFormData = options.body instanceof FormData
  if (!isFormData && !headers.has('Content-Type') && method !== 'GET') {
    headers.set('Content-Type', 'application/json')
  }
  if (options.idempotencyKey) {
    headers.set('Idempotency-Key', options.idempotencyKey)
  }

  let body: BodyInit | undefined
  if (options.body !== undefined) {
    if (isBodyInitLike(options.body)) {
      body = options.body
    } else {
      body = JSON.stringify(options.body)
    }
  }

  let response: Response
  try {
    response = await fetch(url, {
      method,
      headers,
      body,
    })
  } catch (error) {
    throw new ApiHttpError(0, defaultApiError(0, { reason: error instanceof Error ? error.message : 'fetch_failed' }))
  }

  const payload = await readResponseBody(response)

  if (!response.ok) {
    if (isApiErrorEnvelope(payload)) {
      throw new ApiHttpError(response.status, payload.error)
    }
    throw new ApiHttpError(response.status, defaultApiError(response.status))
  }

  return (payload ?? ({} as T)) as T
}
