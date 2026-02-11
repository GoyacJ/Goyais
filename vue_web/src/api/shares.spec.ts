/**
 * SPDX-License-Identifier: Apache-2.0
 * Copyright (c) 2026 Goya
 * Author: Goya
 * Created: 2026-02-11
 * Version: v1.0.0
 * Description: Goyais source file.
 */

import { beforeEach, describe, expect, it, vi } from 'vitest'
import { createShare, deleteShare, listShares } from '@/api/shares'

const apiRequestMock = vi.fn()

vi.mock('@/api/http', () => ({
  apiRequest: (...args: unknown[]) => apiRequestMock(...args),
}))

describe('shares api', () => {
  beforeEach(() => {
    apiRequestMock.mockReset()
    apiRequestMock.mockResolvedValue({})
  })

  it('calls share endpoints', async () => {
    await listShares({ page: 1, pageSize: 20 })
    await createShare(
      {
        resourceType: 'asset',
        resourceId: 'asset_1',
        subjectType: 'user',
        subjectId: 'user_demo',
        permissions: ['READ'],
      },
      'idem-share-create',
    )
    await deleteShare('share_1', 'idem-share-delete')

    expect(apiRequestMock.mock.calls[0]?.[0]).toBe('/shares')
    expect(apiRequestMock.mock.calls[1]?.[0]).toBe('/shares')
    expect(apiRequestMock.mock.calls[2]?.[0]).toBe('/shares/share_1')
  })
})
