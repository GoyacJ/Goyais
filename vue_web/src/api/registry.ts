/**
 * SPDX-License-Identifier: Apache-2.0
 * Copyright (c) 2026 Goya
 * Author: Goya
 * Created: 2026-02-11
 * Version: v1.0.0
 * Description: Goyais source file.
 */

import { apiRequest } from '@/api/http'
import type { AlgorithmDTO, CapabilityDTO, ListResponse, ProviderDTO, WorkflowListParams } from '@/api/types'

export async function listCapabilities(params: WorkflowListParams = {}): Promise<ListResponse<CapabilityDTO>> {
  return apiRequest<ListResponse<CapabilityDTO>>('/registry/capabilities', {
    method: 'GET',
    query: {
      cursor: params.cursor,
      page: params.page,
      pageSize: params.pageSize,
    },
  })
}

export async function getCapability(capabilityId: string): Promise<CapabilityDTO> {
  return apiRequest<CapabilityDTO>(`/registry/capabilities/${encodeURIComponent(capabilityId)}`, {
    method: 'GET',
  })
}

export async function listAlgorithms(params: WorkflowListParams = {}): Promise<ListResponse<AlgorithmDTO>> {
  return apiRequest<ListResponse<AlgorithmDTO>>('/registry/algorithms', {
    method: 'GET',
    query: {
      cursor: params.cursor,
      page: params.page,
      pageSize: params.pageSize,
    },
  })
}

export async function getAlgorithm(algorithmId: string): Promise<AlgorithmDTO> {
  return apiRequest<AlgorithmDTO>(`/registry/algorithms/${encodeURIComponent(algorithmId)}`, {
    method: 'GET',
  })
}

export async function listProviders(params: WorkflowListParams = {}): Promise<ListResponse<ProviderDTO>> {
  return apiRequest<ListResponse<ProviderDTO>>('/registry/providers', {
    method: 'GET',
    query: {
      cursor: params.cursor,
      page: params.page,
      pageSize: params.pageSize,
    },
  })
}
