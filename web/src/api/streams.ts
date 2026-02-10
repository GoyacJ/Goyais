import { apiRequest } from '@/api/http'
import type {
  ResourceSnapshot,
  StreamCreateRequest,
  StreamDTO,
  WorkflowListParams,
  WriteResponse,
  ListResponse,
} from '@/api/types'

export type StreamWriteResource = StreamDTO | ResourceSnapshot

export async function listStreams(params: WorkflowListParams = {}): Promise<ListResponse<StreamDTO>> {
  return apiRequest<ListResponse<StreamDTO>>('/streams', {
    method: 'GET',
    query: {
      cursor: params.cursor,
      page: params.page,
      pageSize: params.pageSize,
    },
  })
}

export async function getStream(streamId: string): Promise<StreamDTO> {
  return apiRequest<StreamDTO>(`/streams/${encodeURIComponent(streamId)}`, {
    method: 'GET',
  })
}

export async function createStream(
  request: StreamCreateRequest,
  idempotencyKey?: string,
): Promise<WriteResponse<StreamWriteResource>> {
  return apiRequest<WriteResponse<StreamWriteResource>>('/streams', {
    method: 'POST',
    body: request,
    idempotencyKey,
  })
}

export async function startStreamRecording(
  streamId: string,
  idempotencyKey?: string,
): Promise<WriteResponse<StreamWriteResource>> {
  return apiRequest<WriteResponse<StreamWriteResource>>(`/streams/${encodeURIComponent(streamId)}:record-start`, {
    method: 'POST',
    body: {},
    idempotencyKey,
  })
}

export async function stopStreamRecording(
  streamId: string,
  idempotencyKey?: string,
): Promise<WriteResponse<StreamWriteResource>> {
  return apiRequest<WriteResponse<StreamWriteResource>>(`/streams/${encodeURIComponent(streamId)}:record-stop`, {
    method: 'POST',
    body: {},
    idempotencyKey,
  })
}

export async function kickStream(
  streamId: string,
  idempotencyKey?: string,
): Promise<WriteResponse<StreamWriteResource>> {
  return apiRequest<WriteResponse<StreamWriteResource>>(`/streams/${encodeURIComponent(streamId)}:kick`, {
    method: 'POST',
    body: {},
    idempotencyKey,
  })
}
