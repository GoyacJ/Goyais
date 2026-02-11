/**
 * SPDX-License-Identifier: Apache-2.0
 * Copyright (c) 2026 Goya
 * Author: Goya
 * Created: 2026-02-11
 * Version: v1.0.0
 * Description: Goyais source file.
 */

import { apiRequest } from '@/api/http'
import type { AlgorithmRunRequest, AlgorithmRunResourceDTO, WriteResponse } from '@/api/types'

export async function runAlgorithm(
  algorithmId: string,
  request: AlgorithmRunRequest,
  idempotencyKey?: string,
): Promise<WriteResponse<AlgorithmRunResourceDTO>> {
  return apiRequest<WriteResponse<AlgorithmRunResourceDTO>>(`/algorithms/${encodeURIComponent(algorithmId)}:run`, {
    method: 'POST',
    body: request,
    idempotencyKey,
  })
}
