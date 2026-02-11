/**
 * SPDX-License-Identifier: Apache-2.0
 * Copyright (c) 2026 Goya
 * Author: Goya
 * Created: 2026-02-11
 * Version: v1.0.0
 * Description: Goyais source file.
 */

import { apiRequest } from '@/api/http'
import type { ContextBundleDTO, ListResponse, WorkflowListParams } from '@/api/types'

export async function listContextBundles(params: WorkflowListParams = {}): Promise<ListResponse<ContextBundleDTO>> {
  return apiRequest<ListResponse<ContextBundleDTO>>('/context-bundles', {
    method: 'GET',
    query: {
      cursor: params.cursor,
      page: params.page,
      pageSize: params.pageSize,
    },
  })
}

export async function getContextBundle(bundleId: string): Promise<ContextBundleDTO> {
  return apiRequest<ContextBundleDTO>(`/context-bundles/${encodeURIComponent(bundleId)}`, {
    method: 'GET',
  })
}
