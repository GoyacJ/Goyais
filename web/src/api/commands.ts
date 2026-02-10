import { apiRequest } from '@/api/http'
import type { CommandCreateRequest, CommandDTO, CommandListParams, ListResponse, WriteResponse } from '@/api/types'

export async function listCommands(params: CommandListParams = {}): Promise<ListResponse<CommandDTO>> {
  return apiRequest<ListResponse<CommandDTO>>('/commands', {
    method: 'GET',
    query: {
      cursor: params.cursor,
      page: params.page,
      pageSize: params.pageSize,
    },
  })
}

export async function getCommand(commandId: string): Promise<CommandDTO> {
  return apiRequest<CommandDTO>(`/commands/${encodeURIComponent(commandId)}`, {
    method: 'GET',
  })
}

export async function createCommand(
  request: CommandCreateRequest,
  idempotencyKey?: string,
): Promise<WriteResponse<CommandDTO>> {
  return apiRequest<WriteResponse<CommandDTO>>('/commands', {
    method: 'POST',
    body: request,
    idempotencyKey,
  })
}
