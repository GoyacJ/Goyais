/**
 * SPDX-License-Identifier: Apache-2.0
 * Copyright (c) 2026 Goya
 * Author: Goya
 * Created: 2026-02-11
 * Version: v1.0.0
 * Description: Goyais source file.
 */

import { mount } from '@vue/test-utils'
import type { AssetDTO } from '@/api/types'
import AssetsView from '@/views/AssetsView.vue'
import i18n from '@/i18n'
import { nextTick } from 'vue'
import { beforeEach, vi } from 'vitest'

const listAssetsMock = vi.fn()
const createAssetMock = vi.fn()

vi.mock('@/api/http', () => ({
  isMockEnabled: () => false,
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

vi.mock('@/api/assets', () => ({
  listAssets: (...args: unknown[]) => listAssetsMock(...args),
  createAsset: (...args: unknown[]) => createAssetMock(...args),
}))

const stubs = {
  WindowBoard: {
    props: ['routeKey', 'panes'],
    template: '<div><slot name="filters" /><slot name="list" /><slot name="detail" /></div>',
  },
  PageHeader: {
    props: ['title', 'subtitle'],
    template: '<header><h1>{{ title }}</h1><p>{{ subtitle }}</p><slot name="actions" /></header>',
  },
  SectionCard: {
    props: ['title', 'subtitle'],
    template: '<section><h2>{{ title }}</h2><p>{{ subtitle }}</p><slot /></section>',
  },
  EmptyState: { template: '<div data-testid="empty-state"><slot /></div>' },
  Button: { template: '<button><slot /></button>' },
  Icon: { template: '<span />' },
  Select: { template: '<div data-testid="select-stub" />' },
  Input: { template: '<input />' },
}

async function flushAll(): Promise<void> {
  await Promise.resolve()
  await nextTick()
  await Promise.resolve()
}

function buildAsset(id: string, name: string, ownerId: string, visibility: AssetDTO['visibility']): AssetDTO {
  return {
    id,
    tenantId: 't1',
    workspaceId: 'w1',
    ownerId,
    visibility,
    acl: [],
    status: 'succeeded',
    createdAt: '2026-02-10T08:00:00Z',
    updatedAt: '2026-02-10T08:00:01Z',
    name,
    type: 'image/png',
    mime: 'image/png',
    size: 4096,
    uri: `local:///${name}`,
    hash: `sha256:${id}`,
    metadata: {},
  }
}

describe('AssetsView', () => {
  beforeEach(() => {
    listAssetsMock.mockReset()
    createAssetMock.mockReset()

    listAssetsMock.mockResolvedValue({
      items: [
        buildAsset('ast_1', 'warehouse-cam-01.mp4', 'u_alice', 'WORKSPACE'),
        buildAsset('ast_2', 'dashboard-snapshot.png', 'u_bob', 'PRIVATE'),
        buildAsset('ast_3', 'daily-report.json', 'u_ops', 'TENANT'),
      ],
      pageInfo: {
        page: 1,
        pageSize: 20,
        total: 3,
      },
    })

    createAssetMock.mockResolvedValue({
      resource: {
        id: 'ast_new',
        status: 'accepted',
      },
      commandRef: {
        commandId: 'cmd_asset_upload',
        status: 'accepted',
        acceptedAt: '2026-02-10T09:00:00Z',
      },
    })
  })

  it('updates detail content when selecting rows via mouse and keyboard', async () => {
    const wrapper = mount(AssetsView, {
      global: {
        plugins: [i18n],
        stubs,
      },
    })

    await flushAll()

    expect(wrapper.text()).toContain('warehouse-cam-01.mp4')

    const rows = wrapper.findAll('tbody tr')
    await rows[1]?.trigger('click')
    expect(wrapper.text()).toContain('dashboard-snapshot.png')

    await rows[2]?.trigger('keydown.enter')
    expect(wrapper.text()).toContain('daily-report.json')
  })
})
