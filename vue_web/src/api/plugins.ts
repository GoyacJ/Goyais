/**
 * SPDX-License-Identifier: Apache-2.0
 * Copyright (c) 2026 Goya
 * Author: Goya
 * Created: 2026-02-11
 * Version: v1.0.0
 * Description: Goyais source file.
 */

import { ApiHttpError, apiRequest, getApiRuntimeConfig } from '@/api/http'
import type {
  ListResponse,
  PluginInstallDTO,
  PluginInstallRequest,
  PluginPackageDTO,
  PluginPackageUploadRequest,
  ResourceSnapshot,
  WorkflowListParams,
  WriteResponse,
} from '@/api/types'

export type PluginInstallWriteResource = PluginInstallDTO | ResourceSnapshot

export async function listPluginPackages(params: WorkflowListParams = {}): Promise<ListResponse<PluginPackageDTO>> {
  return apiRequest<ListResponse<PluginPackageDTO>>('/plugin-market/packages', {
    method: 'GET',
    query: {
      cursor: params.cursor,
      page: params.page,
      pageSize: params.pageSize,
    },
  })
}

export async function uploadPluginPackage(
  request: PluginPackageUploadRequest,
  idempotencyKey?: string,
): Promise<WriteResponse<PluginPackageDTO>> {
  return apiRequest<WriteResponse<PluginPackageDTO>>('/plugin-market/packages', {
    method: 'POST',
    body: request,
    idempotencyKey,
  })
}

export async function installPlugin(
  request: PluginInstallRequest,
  idempotencyKey?: string,
): Promise<WriteResponse<PluginInstallWriteResource>> {
  return apiRequest<WriteResponse<PluginInstallWriteResource>>('/plugin-market/installs', {
    method: 'POST',
    body: request,
    idempotencyKey,
  })
}

export async function enablePluginInstall(
  installId: string,
  idempotencyKey?: string,
): Promise<WriteResponse<PluginInstallWriteResource>> {
  return apiRequest<WriteResponse<PluginInstallWriteResource>>(`/plugin-market/installs/${encodeURIComponent(installId)}:enable`, {
    method: 'POST',
    body: {},
    idempotencyKey,
  })
}

export async function disablePluginInstall(
  installId: string,
  idempotencyKey?: string,
): Promise<WriteResponse<PluginInstallWriteResource>> {
  return apiRequest<WriteResponse<PluginInstallWriteResource>>(`/plugin-market/installs/${encodeURIComponent(installId)}:disable`, {
    method: 'POST',
    body: {},
    idempotencyKey,
  })
}

export async function rollbackPluginInstall(
  installId: string,
  idempotencyKey?: string,
): Promise<WriteResponse<PluginInstallWriteResource>> {
  return apiRequest<WriteResponse<PluginInstallWriteResource>>(`/plugin-market/installs/${encodeURIComponent(installId)}:rollback`, {
    method: 'POST',
    body: {},
    idempotencyKey,
  })
}

export async function upgradePluginInstall(
  installId: string,
  idempotencyKey?: string,
): Promise<WriteResponse<PluginInstallWriteResource>> {
  return apiRequest<WriteResponse<PluginInstallWriteResource>>(`/plugin-market/installs/${encodeURIComponent(installId)}:upgrade`, {
    method: 'POST',
    body: {},
    idempotencyKey,
  })
}

export interface DownloadPluginPackageResult {
  filename: string
  content: string
}

export async function downloadPluginPackage(packageId: string): Promise<DownloadPluginPackageResult> {
  const runtime = getApiRuntimeConfig()
  const path = `/plugin-market/packages/${encodeURIComponent(packageId)}:download`
  const url = `${runtime.apiBaseUrl}${path}`

  const headers = new Headers()
  headers.set('X-Tenant-Id', runtime.tenantId)
  headers.set('X-Workspace-Id', runtime.workspaceId)
  headers.set('X-User-Id', runtime.userId)
  headers.set('X-Roles', runtime.roles)
  headers.set('X-Policy-Version', runtime.policyVersion)

  let response: Response
  try {
    response = await fetch(url, { method: 'GET', headers })
  } catch (error) {
    throw new ApiHttpError(0, {
      code: 'NETWORK_ERROR',
      messageKey: 'error.common.internal',
      details: { reason: error instanceof Error ? error.message : 'fetch_failed' },
    })
  }

  if (!response.ok) {
    let envelope: unknown
    try {
      envelope = await response.json()
    } catch {
      envelope = undefined
    }
    if (envelope && typeof envelope === 'object') {
      const errorPayload = (envelope as Record<string, unknown>).error
      if (errorPayload && typeof errorPayload === 'object') {
        const code = (errorPayload as Record<string, unknown>).code
        const messageKey = (errorPayload as Record<string, unknown>).messageKey
        const details = (errorPayload as Record<string, unknown>).details
        if (typeof code === 'string' && typeof messageKey === 'string') {
          throw new ApiHttpError(response.status, {
            code,
            messageKey,
            details: (details ?? undefined) as Record<string, unknown> | undefined,
          })
        }
      }
    }
    throw new ApiHttpError(response.status, {
      code: `HTTP_${response.status}`,
      messageKey: 'error.common.internal',
    })
  }

  const disposition = response.headers.get('content-disposition') ?? ''
  const match = disposition.match(/filename="([^"]+)"/i)
  const filename = match?.[1]?.trim() || `${packageId}.json`
  const content = await response.text()
  return { filename, content }
}
