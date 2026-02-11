import { beforeEach, describe, expect, it, vi } from 'vitest'
import {
  disablePluginInstall,
  enablePluginInstall,
  installPlugin,
  listPluginPackages,
  rollbackPluginInstall,
  uploadPluginPackage,
} from '@/api/plugins'

const apiRequestMock = vi.fn()

vi.mock('@/api/http', () => ({
  apiRequest: (...args: unknown[]) => apiRequestMock(...args),
}))

describe('plugins api', () => {
  beforeEach(() => {
    apiRequestMock.mockReset()
    apiRequestMock.mockResolvedValue({})
  })

  it('calls plugin package and install endpoints', async () => {
    await listPluginPackages({ page: 1, pageSize: 20 })
    await uploadPluginPackage(
      {
        name: 'demo-plugin',
        version: '1.0.0',
        packageType: 'tool-provider',
        manifest: { entry: 'main' },
      },
      'idem-upload',
    )
    await installPlugin({ packageId: 'pkg_1', scope: 'workspace' }, 'idem-install')
    await enablePluginInstall('ins_1', 'idem-enable')
    await disablePluginInstall('ins_1', 'idem-disable')
    await rollbackPluginInstall('ins_1', 'idem-rollback')

    expect(apiRequestMock.mock.calls[0]?.[0]).toBe('/plugin-market/packages')
    expect(apiRequestMock.mock.calls[1]?.[0]).toBe('/plugin-market/packages')
    expect(apiRequestMock.mock.calls[2]?.[0]).toBe('/plugin-market/installs')
    expect(apiRequestMock.mock.calls[3]?.[0]).toBe('/plugin-market/installs/ins_1:enable')
    expect(apiRequestMock.mock.calls[4]?.[0]).toBe('/plugin-market/installs/ins_1:disable')
    expect(apiRequestMock.mock.calls[5]?.[0]).toBe('/plugin-market/installs/ins_1:rollback')
  })
})
