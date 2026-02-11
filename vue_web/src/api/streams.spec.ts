/**
 * SPDX-License-Identifier: Apache-2.0
 * Copyright (c) 2026 Goya
 * Author: Goya
 * Created: 2026-02-11
 * Version: v1.0.0
 * Description: Goyais source file.
 */

import { beforeEach, describe, expect, it, vi } from 'vitest'
import {
  createStream,
  deleteStream,
  getStream,
  kickStream,
  listStreams,
  startStreamRecording,
  stopStreamRecording,
  updateStreamAuth,
} from '@/api/streams'

const apiRequestMock = vi.fn()

vi.mock('@/api/http', () => ({
  apiRequest: (...args: unknown[]) => apiRequestMock(...args),
}))

describe('streams api', () => {
  beforeEach(() => {
    apiRequestMock.mockReset()
    apiRequestMock.mockResolvedValue({})
  })

  it('calls stream endpoints', async () => {
    await listStreams({ page: 1, pageSize: 20 })
    await createStream(
      {
        path: 'live/cam-01',
        protocol: 'rtmp',
        source: 'push',
        metadata: { onPublishTemplateId: 'tpl_1' },
      },
      'idem-stream-create',
    )
    await getStream('str_1')
    await startStreamRecording('str_1', 'idem-start')
    await stopStreamRecording('str_1', 'idem-stop')
    await kickStream('str_1', 'idem-kick')
    await updateStreamAuth('str_1', { token: 'demo-token' }, 'idem-auth')
    await deleteStream('str_1', 'idem-delete')

    expect(apiRequestMock.mock.calls[0]?.[0]).toBe('/streams')
    expect(apiRequestMock.mock.calls[1]?.[0]).toBe('/streams')
    expect(apiRequestMock.mock.calls[2]?.[0]).toBe('/streams/str_1')
    expect(apiRequestMock.mock.calls[3]?.[0]).toBe('/streams/str_1:record-start')
    expect(apiRequestMock.mock.calls[4]?.[0]).toBe('/streams/str_1:record-stop')
    expect(apiRequestMock.mock.calls[5]?.[0]).toBe('/streams/str_1:kick')
    expect(apiRequestMock.mock.calls[6]?.[0]).toBe('/streams/str_1:update-auth')
    expect(apiRequestMock.mock.calls[7]?.[0]).toBe('/streams/str_1')
  })
})
