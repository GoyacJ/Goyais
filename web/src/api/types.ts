import type { Visibility } from '@/design-system/types'

export interface ApiError {
  code: string
  messageKey: string
  details?: Record<string, unknown>
}

export interface ApiErrorEnvelope {
  error: ApiError
}

export interface PageInfo {
  page: number
  pageSize: number
  total: number
}

export interface CursorInfo {
  nextCursor?: string | null
}

export interface ListResponse<T> {
  items: T[]
  pageInfo?: PageInfo
  cursorInfo?: CursorInfo
}

export interface CommandRef {
  commandId: string
  status: string
  acceptedAt: string
}

export interface WriteResponse<T> {
  resource: T
  commandRef: CommandRef
}

export interface ACLItem {
  subjectType: string
  subjectId: string
  permissions: string[]
  expiresAt?: string | null
}

export interface ResourceBase {
  id: string
  tenantId: string
  workspaceId: string
  ownerId: string
  visibility: Visibility
  acl: ACLItem[]
  status: string
  createdAt: string
  updatedAt: string
}

export interface CommandDTO extends ResourceBase {
  commandType: string
  payload: Record<string, unknown>
  result?: Record<string, unknown>
  error?: ApiError
  acceptedAt: string
  finishedAt?: string
  traceId: string
}

export interface AssetDTO extends ResourceBase {
  name: string
  type: string
  mime: string
  size: number
  hash: string
  uri: string
  metadata?: Record<string, unknown>
}

export interface ResourceSnapshot {
  id: string
  status: string
}

export interface CommandCreateRequest {
  commandType: string
  payload: Record<string, unknown>
  visibility?: Visibility
}

export interface CommandListParams {
  cursor?: string
  page?: number
  pageSize?: number
}

export interface AssetListParams {
  cursor?: string
  page?: number
  pageSize?: number
}
