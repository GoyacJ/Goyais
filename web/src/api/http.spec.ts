import { afterEach, describe, expect, it, vi } from 'vitest'
import { ApiHttpError, apiRequest } from '@/api/http'

describe('apiRequest', () => {
  afterEach(() => {
    vi.restoreAllMocks()
  })

  it('injects context headers and idempotency key', async () => {
    const fetchMock = vi.spyOn(globalThis, 'fetch').mockResolvedValue(
      new Response(JSON.stringify({ items: [] }), {
        status: 200,
        headers: {
          'Content-Type': 'application/json',
        },
      }),
    )

    await apiRequest('/commands', {
      method: 'POST',
      body: {
        commandType: 'ui.ping',
        payload: { ping: true },
      },
      idempotencyKey: 'idem-1',
    })

    expect(fetchMock).toHaveBeenCalledTimes(1)
    const [url, init] = fetchMock.mock.calls[0] ?? []
    expect(url).toBe('/api/v1/commands')

    const headers = new Headers((init as RequestInit).headers)
    expect(headers.get('X-Tenant-Id')).toBe('t1')
    expect(headers.get('X-Workspace-Id')).toBe('w1')
    expect(headers.get('X-User-Id')).toBe('u1')
    expect(headers.get('X-Roles')).toBe('member')
    expect(headers.get('X-Policy-Version')).toBe('v0.1')
    expect(headers.get('Idempotency-Key')).toBe('idem-1')
  })

  it('throws ApiHttpError with backend error payload', async () => {
    vi.spyOn(globalThis, 'fetch').mockResolvedValue(
      new Response(
        JSON.stringify({
          error: {
            code: 'FORBIDDEN',
            messageKey: 'error.authz.forbidden',
          },
        }),
        {
          status: 403,
          headers: {
            'Content-Type': 'application/json',
          },
        },
      ),
    )

    const apiErr = await apiRequest('/commands', { method: 'GET' }).catch((error) => error)
    expect(apiErr).toBeInstanceOf(ApiHttpError)
    if (!(apiErr instanceof ApiHttpError)) {
      throw new Error('expected ApiHttpError')
    }
    expect(apiErr.status).toBe(403)
    expect(apiErr.error.code).toBe('FORBIDDEN')
    expect(apiErr.error.messageKey).toBe('error.authz.forbidden')
  })
})
