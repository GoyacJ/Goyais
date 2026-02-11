/**
 * SPDX-License-Identifier: Apache-2.0
 * Copyright (c) 2026 Goya
 * Author: Goya
 * Created: 2026-02-11
 * Version: v1.0.0
 * Description: Goyais source file.
 */

import { apiRequest } from '@/api/http'
import type {
  ListResponse,
  ResourceSnapshot,
  ShareCreateRequest,
  ShareDTO,
  WorkflowListParams,
  WriteResponse,
} from '@/api/types'

export type ShareWriteResource = ShareDTO | ResourceSnapshot

export async function listShares(params: WorkflowListParams = {}): Promise<ListResponse<ShareDTO>> {
  return apiRequest<ListResponse<ShareDTO>>('/shares', {
    method: 'GET',
    query: {
      cursor: params.cursor,
      page: params.page,
      pageSize: params.pageSize,
    },
  })
}

export async function createShare(
  request: ShareCreateRequest,
  idempotencyKey?: string,
): Promise<WriteResponse<ShareWriteResource>> {
  return apiRequest<WriteResponse<ShareWriteResource>>('/shares', {
    method: 'POST',
    body: request,
    idempotencyKey,
  })
}

export async function deleteShare(shareId: string, idempotencyKey?: string): Promise<WriteResponse<ShareWriteResource>> {
  return apiRequest<WriteResponse<ShareWriteResource>>(`/shares/${encodeURIComponent(shareId)}`, {
    method: 'DELETE',
    idempotencyKey,
  })
}
