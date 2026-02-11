/**
 * SPDX-License-Identifier: Apache-2.0
 * Copyright (c) 2026 Goya
 * Author: Goya
 * Created: 2026-02-11
 * Version: v1.0.0
 * Description: Goyais source file.
 */

import { beforeEach, describe, expect, it, vi } from 'vitest'
import { getContextBundle, listContextBundles } from '@/api/context-bundles'

const apiRequestMock = vi.fn()

vi.mock('@/api/http', () => ({
  apiRequest: (...args: unknown[]) => apiRequestMock(...args),
}))

describe('context bundle api', () => {
  beforeEach(() => {
    apiRequestMock.mockReset()
    apiRequestMock.mockResolvedValue({})
  })

  it('calls context bundle endpoints', async () => {
    await listContextBundles({ page: 1, pageSize: 20 })
    await getContextBundle('cb_1')

    expect(apiRequestMock.mock.calls[0]?.[0]).toBe('/context-bundles')
    expect(apiRequestMock.mock.calls[1]?.[0]).toBe('/context-bundles/cb_1')
  })
})
