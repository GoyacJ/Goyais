/**
 * SPDX-License-Identifier: Apache-2.0
 * Copyright (c) 2026 Goya
 * Author: Goya
 * Created: 2026-02-11
 * Version: v1.0.0
 * Description: Goyais source file.
 */

import { mount } from '@vue/test-utils'
import type { CommandDTO, PluginPackageDTO } from '@/api/types'
import i18n from '@/i18n'
import PluginsView from '@/views/PluginsView.vue'
import { nextTick } from 'vue'
import { beforeEach, vi } from 'vitest'

const listPluginPackagesMock = vi.fn()
const uploadPluginPackageMock = vi.fn()
const installPluginMock = vi.fn()
const enablePluginInstallMock = vi.fn()
const disablePluginInstallMock = vi.fn()
const rollbackPluginInstallMock = vi.fn()
const listCommandsMock = vi.fn()

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

vi.mock('@/api/plugins', () => ({
  listPluginPackages: (...args: unknown[]) => listPluginPackagesMock(...args),
  uploadPluginPackage: (...args: unknown[]) => uploadPluginPackageMock(...args),
  installPlugin: (...args: unknown[]) => installPluginMock(...args),
  enablePluginInstall: (...args: unknown[]) => enablePluginInstallMock(...args),
  disablePluginInstall: (...args: unknown[]) => disablePluginInstallMock(...args),
  rollbackPluginInstall: (...args: unknown[]) => rollbackPluginInstallMock(...args),
}))

vi.mock('@/api/commands', () => ({
  listCommands: (...args: unknown[]) => listCommandsMock(...args),
}))

const stubs = {
  WindowBoard: {
    props: ['routeKey', 'panes'],
    template: '<div><slot name="plugin-catalog" /><slot name="plugin-activity" /></div>',
  },
  PageHeader: {
    props: ['title', 'subtitle'],
    template: '<header><h1>{{ title }}</h1><p>{{ subtitle }}</p><slot name="actions" /></header>',
  },
  SectionCard: {
    props: ['title', 'subtitle'],
    template: '<section><h2>{{ title }}</h2><p>{{ subtitle }}</p><slot /></section>',
  },
  PluginPackageCard: {
    props: ['item', 'busy'],
    template: `
      <article>
        <p>{{ item.name }}</p>
        <button data-testid="install" @click="$emit('install', item.id)">install</button>
      </article>
    `,
  },
  PluginCommandTimeline: {
    props: ['events'],
    template: '<div data-testid="plugin-timeline">{{ events.length }}</div>',
  },
  EmptyState: { template: '<div data-testid="empty-state" />' },
  Button: { template: '<button><slot /></button>' },
  Icon: { template: '<span />' },
  Input: { template: '<input />' },
  Textarea: { template: '<textarea />' },
  Select: {
    props: ['modelValue', 'options'],
    template: '<select><option v-for="item in options" :key="item.value">{{ item.label }}</option></select>',
  },
}

async function flushAll(): Promise<void> {
  await Promise.resolve()
  await nextTick()
  await Promise.resolve()
}

function buildPluginPackage(id: string, name: string): PluginPackageDTO {
  return {
    id,
    tenantId: 't1',
    workspaceId: 'w1',
    ownerId: 'u1',
    visibility: 'PRIVATE',
    acl: [],
    status: 'uploaded',
    createdAt: '2026-02-10T10:00:00Z',
    updatedAt: '2026-02-10T10:00:01Z',
    name,
    version: '1.0.0',
    packageType: 'tool-provider',
    manifest: {},
  }
}

function buildPluginCommand(): CommandDTO {
  return {
    id: 'cmd_plugin_install',
    tenantId: 't1',
    workspaceId: 'w1',
    ownerId: 'u1',
    visibility: 'PRIVATE',
    acl: [],
    status: 'succeeded',
    createdAt: '2026-02-10T10:00:00Z',
    updatedAt: '2026-02-10T10:00:01Z',
    acceptedAt: '2026-02-10T10:00:00Z',
    traceId: 'trace-plugin',
    commandType: 'plugin.install',
    payload: { packageId: 'pkg_1' },
    result: {
      install: {
        id: 'ins_1',
        packageId: 'pkg_1',
        status: 'enabled',
      },
    },
  }
}

describe('PluginsView', () => {
  beforeEach(() => {
    listPluginPackagesMock.mockReset()
    uploadPluginPackageMock.mockReset()
    installPluginMock.mockReset()
    enablePluginInstallMock.mockReset()
    disablePluginInstallMock.mockReset()
    rollbackPluginInstallMock.mockReset()
    listCommandsMock.mockReset()

    listPluginPackagesMock.mockResolvedValue({
      items: [buildPluginPackage('pkg_1', 'Demo Plugin')],
      pageInfo: { page: 1, pageSize: 20, total: 1 },
    })
    listCommandsMock.mockResolvedValue({
      items: [buildPluginCommand()],
      pageInfo: { page: 1, pageSize: 20, total: 1 },
    })
    installPluginMock.mockResolvedValue({
      resource: { id: 'ins_1', status: 'enabled' },
      commandRef: { commandId: 'cmd_install', status: 'accepted', acceptedAt: '2026-02-10T10:00:00Z' },
    })
  })

  it('loads package list and runs install action', async () => {
    const wrapper = mount(PluginsView, {
      global: {
        plugins: [i18n],
        stubs,
      },
    })

    await flushAll()

    expect(wrapper.text()).toContain('Demo Plugin')
    expect(wrapper.get('[data-testid="plugin-timeline"]').text()).toContain('1')

    await wrapper.get('[data-testid="install"]').trigger('click')
    expect(installPluginMock).toHaveBeenCalledTimes(1)
  })
})
