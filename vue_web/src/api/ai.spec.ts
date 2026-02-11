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
  archiveAISession,
  createAISession,
  createAISessionTurn,
  getAISession,
  getAISessionEvents,
  listAISessions,
  previewAIPlan,
} from '@/api/ai'

const apiRequestMock = vi.fn()

vi.mock('@/api/http', () => ({
  apiRequest: (...args: unknown[]) => apiRequestMock(...args),
  getApiRuntimeConfig: () => ({
    apiBaseUrl: '/api/v1',
    tenantId: 't1',
    workspaceId: 'w1',
    userId: 'u1',
    roles: 'member',
    policyVersion: 'v0.1',
    mockEnabled: false,
  }),
  ApiHttpError: class ApiHttpError extends Error {
    readonly status: number
    readonly error: { code: string; messageKey: string }

    constructor(status: number, error: { code: string; messageKey: string }) {
      super(error.messageKey)
      this.status = status
      this.error = error
    }
  },
}))

describe('ai api', () => {
  beforeEach(() => {
    apiRequestMock.mockReset()
    apiRequestMock.mockResolvedValue({})
  })

  it('calls ai session endpoints', async () => {
    await listAISessions({ page: 1, pageSize: 20 })
    await createAISession({ title: 't', goal: 'g' }, 'idem-ai-create')
    await getAISession('sess_1')
    await previewAIPlan({ message: 'run workflow tpl_1' })
    await createAISessionTurn('sess_1', { message: 'hello', execute: false }, 'idem-ai-turn')
    await archiveAISession('sess_1', 'idem-ai-archive')

    expect(apiRequestMock.mock.calls[0]?.[0]).toBe('/ai/sessions')
    expect(apiRequestMock.mock.calls[1]?.[0]).toBe('/ai/sessions')
    expect(apiRequestMock.mock.calls[2]?.[0]).toBe('/ai/sessions/sess_1')
    expect(apiRequestMock.mock.calls[3]?.[0]).toBe('/ai/plans:preview')
    expect(apiRequestMock.mock.calls[4]?.[0]).toBe('/ai/sessions/sess_1/turns')
    expect(apiRequestMock.mock.calls[5]?.[0]).toBe('/ai/sessions/sess_1:archive')
  })

  it('parses sse events from ai session events endpoint', async () => {
    const fetchMock = vi.fn().mockResolvedValue(
      new Response(
        [
          'id: turn_1',
          'event: ai.turn.user',
          'data: {"id":"turn_1","role":"user","content":"hello"}',
          '',
          'id: turn_2',
          'event: ai.turn.assistant',
          'data: {"id":"turn_2","role":"assistant","content":"Plan drafted"}',
          '',
        ].join('\n'),
        {
          status: 200,
          headers: { 'Content-Type': 'text/event-stream' },
        },
      ),
    )
    vi.stubGlobal('fetch', fetchMock)

    const events = await getAISessionEvents('sess_1')

    expect(fetchMock).toHaveBeenCalledTimes(1)
    expect(fetchMock.mock.calls[0]?.[0]).toBe('/api/v1/ai/sessions/sess_1/events')
    expect(events).toHaveLength(2)
    expect(events[0]?.event).toBe('ai.turn.user')
    expect(events[1]?.event).toBe('ai.turn.assistant')
  })
})
