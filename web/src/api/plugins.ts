import { apiRequest } from '@/api/http'
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
