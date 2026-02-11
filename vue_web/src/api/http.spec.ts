/**
 * SPDX-License-Identifier: Apache-2.0
 * Copyright (c) 2026 Goya
 * Author: Goya
 * Created: 2026-02-11
 * Version: v1.0.0
 * Description: Goyais source file.
 */

import { afterEach, describe, expect, it, vi } from 'vitest'
import { ApiHttpError, apiRequest, setApiRuntimeContext } from '@/api/http'

const DEFAULT_CONTEXT = {
  tenantId: 't1',
  workspaceId: 'w1',
  userId: 'u1',
  roles: 'member',
  policyVersion: 'v0.1',
}

describe('apiRequest', () => {
  afterEach(() => {
    vi.restoreAllMocks()
    setApiRuntimeContext(DEFAULT_CONTEXT)
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

  it('updates context headers after runtime context switch', async () => {
    setApiRuntimeContext({
      tenantId: 'tenant-next',
      workspaceId: 'workspace-next',
      userId: 'user-next',
      roles: 'admin',
      policyVersion: 'v9.9',
    })

    const fetchMock = vi.spyOn(globalThis, 'fetch').mockResolvedValue(
      new Response(JSON.stringify({ items: [] }), {
        status: 200,
        headers: {
          'Content-Type': 'application/json',
        },
      }),
    )

    await apiRequest('/commands', {
      method: 'GET',
    })

    const [, init] = fetchMock.mock.calls[0] ?? []
    const headers = new Headers((init as RequestInit).headers)
    expect(headers.get('X-Tenant-Id')).toBe('tenant-next')
    expect(headers.get('X-Workspace-Id')).toBe('workspace-next')
    expect(headers.get('X-User-Id')).toBe('user-next')
    expect(headers.get('X-Roles')).toBe('admin')
    expect(headers.get('X-Policy-Version')).toBe('v9.9')
  })
})
