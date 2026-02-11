import { apiRequest } from '@/api/http'
import type { AssetDTO, AssetListParams, ListResponse, ResourceSnapshot, WriteResponse } from '@/api/types'
import type { Visibility } from '@/design-system/types'

export type AssetWriteResource = AssetDTO | ResourceSnapshot

export interface AssetCreateRequest {
  file: File
  name?: string
  type?: string
  visibility?: Visibility
}

export async function listAssets(params: AssetListParams = {}): Promise<ListResponse<AssetDTO>> {
  return apiRequest<ListResponse<AssetDTO>>('/assets', {
    method: 'GET',
    query: {
      cursor: params.cursor,
      page: params.page,
      pageSize: params.pageSize,
    },
  })
}

export async function getAsset(assetId: string): Promise<AssetDTO> {
  return apiRequest<AssetDTO>(`/assets/${encodeURIComponent(assetId)}`, {
    method: 'GET',
  })
}

export async function createAsset(
  request: AssetCreateRequest,
  idempotencyKey?: string,
): Promise<WriteResponse<AssetWriteResource>> {
  const form = new FormData()
  form.append('file', request.file)
  if (request.name) {
    form.append('name', request.name)
  }
  if (request.type) {
    form.append('type', request.type)
  }
  if (request.visibility) {
    form.append('visibility', request.visibility)
  }

  return apiRequest<WriteResponse<AssetWriteResource>>('/assets', {
    method: 'POST',
    body: form,
    idempotencyKey,
  })
}
