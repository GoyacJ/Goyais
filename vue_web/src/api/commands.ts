/**
 * SPDX-License-Identifier: Apache-2.0
 * Copyright (c) 2026 Goya
 * Author: Goya
 * Created: 2026-02-11
 * Version: v1.0.0
 * Description: Goyais source file.
 */

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
