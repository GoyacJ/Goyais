import type { CommandStatus, Visibility } from '@/design-system/types'

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

export type ApiObject = Record<string, unknown>

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
  status: CommandStatus
  commandType: string
  payload: ApiObject
  result?: ApiObject
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
  metadata?: ApiObject
}

export interface WorkflowTemplateDTO extends ResourceBase {
  name: string
  description?: string
  graph: ApiObject
  schemaInputs: ApiObject
  schemaOutputs: ApiObject
  uiState: ApiObject
  currentVersion?: string
}

export interface WorkflowRunDTO extends ResourceBase {
  templateId: string
  templateVersion: string
  attempt: number
  retryOfRunId?: string
  replayFromStepKey?: string
  traceId?: string
  inputs: ApiObject
  outputs: ApiObject
  startedAt: string
  finishedAt?: string
  durationMs?: number
  error?: ApiError
}

export interface AISessionDTO extends ResourceBase {
  title: string
  goal?: string
  inputs: ApiObject
  constraints: ApiObject
  preferences: ApiObject
  archivedAt?: string
  lastTurnAt?: string
}

export interface AISessionTurnDTO {
  id: string
  sessionId: string
  tenantId: string
  workspaceId: string
  ownerId: string
  visibility: Visibility
  role: 'user' | 'assistant' | 'system' | string
  content: string
  commandType?: string
  commandIds?: string[]
  createdAt: string
}

export interface StepRunDTO extends ResourceBase {
  runId: string
  stepKey: string
  stepType: string
  attempt: number
  traceId?: string
  input: ApiObject
  output: ApiObject
  artifacts: ApiObject
  logRef?: string
  startedAt: string
  finishedAt?: string
  durationMs?: number
  error?: ApiError
}

export interface CapabilityDTO extends ResourceBase {
  name: string
  kind: string
  version: string
  providerId: string
  inputSchema: ApiObject
  outputSchema: ApiObject
  requiredPermissions: string[]
  egressPolicy: ApiObject
}

export interface AlgorithmDTO extends ResourceBase {
  name: string
  version: string
  templateRef: string
  defaults: ApiObject
  constraints: ApiObject
  dependencies: ApiObject
}

export interface ProviderDTO extends ResourceBase {
  name: string
  providerType: string
  endpoint: string
  metadata: ApiObject
}

export type PluginPackageType = 'tool-provider' | 'skill-pack' | 'algo-pack' | 'mcp-provider'
export type PluginInstallScope = 'workspace' | 'tenant'

export interface PluginPackageDTO extends ResourceBase {
  name: string
  version: string
  packageType: PluginPackageType | string
  manifest: ApiObject
}

export interface PluginInstallDTO extends ResourceBase {
  packageId: string
  scope: PluginInstallScope | string
  installedAt?: string
  error?: ApiError
}

export interface StreamDTO extends ResourceBase {
  path: string
  protocol: string
  source: string
  endpoints: ApiObject
  state: ApiObject
}

export interface StreamRecordingDTO {
  id: string
  streamId: string
  tenantId: string
  workspaceId: string
  ownerId: string
  visibility: Visibility
  status: string
  startedAt: string
  finishedAt?: string
  createdAt: string
  updatedAt: string
  assetId?: string
  error?: ApiError
}

export interface AlgorithmRunResourceDTO {
  id: string
  algorithmId: string
  workflowRunId: string
  status: string
  outputs?: ApiObject
  assetIds?: string[]
  createdAt: string
  updatedAt: string
  error?: ApiError
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

export interface WorkflowListParams {
  cursor?: string
  page?: number
  pageSize?: number
}

export interface WorkflowTemplateCreateRequest {
  name: string
  description?: string
  graph: ApiObject
  schemaInputs?: ApiObject
  schemaOutputs?: ApiObject
  visibility?: Visibility
}

export interface WorkflowTemplatePatchRequest {
  graph?: ApiObject
  operations?: ApiObject[]
}

export interface WorkflowRunCreateRequest {
  templateId: string
  templateVersion?: string
  inputs?: ApiObject
  mode?: 'sync' | 'running' | 'fail'
  visibility?: Visibility
}

export interface AISessionCreateRequest {
  title?: string
  goal?: string
  inputs?: ApiObject
  constraints?: ApiObject
  preferences?: ApiObject
  visibility?: Visibility
}

export interface AISessionTurnCreateRequest {
  message: string
  execute?: boolean
  inputs?: ApiObject
  constraints?: ApiObject
  preferences?: ApiObject
}

export interface PluginPackageUploadRequest {
  name: string
  version: string
  packageType: PluginPackageType | string
  manifest?: ApiObject
  visibility?: Visibility
}

export interface PluginInstallRequest {
  packageId: string
  scope?: PluginInstallScope
}

export interface StreamCreateRequest {
  path: string
  protocol: 'rtsp' | 'rtmp' | 'srt' | 'webrtc' | 'hls' | string
  source?: 'push' | 'pull' | string
  visibility?: Visibility
  metadata?: ApiObject
}

export interface AlgorithmRunRequest {
  inputs: ApiObject
  visibility?: Visibility
  mode?: string
}
