/**
 * SPDX-License-Identifier: Apache-2.0
 * Copyright (c) 2026 Goya
 * Author: Goya
 * Created: 2026-02-11
 * Version: v1.0.0
 * Description: Goyais source file.
 */

import { mount } from '@vue/test-utils'
import { nextTick } from 'vue'
import { beforeEach, describe, expect, it, vi } from 'vitest'

import i18n from '@/i18n'
import PermissionManagementView from '@/views/PermissionManagementView.vue'

const listCommandsMock = vi.fn()
const listSharesMock = vi.fn()
const createShareMock = vi.fn()
const deleteShareMock = vi.fn()
const pushToastMock = vi.fn()

vi.mock('@/api/http', () => ({
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

vi.mock('@/api/commands', () => ({
  listCommands: (...args: unknown[]) => listCommandsMock(...args),
}))

vi.mock('@/api/shares', () => ({
  listShares: (...args: unknown[]) => listSharesMock(...args),
  createShare: (...args: unknown[]) => createShareMock(...args),
  deleteShare: (...args: unknown[]) => deleteShareMock(...args),
}))

vi.mock('@/composables/useToast', () => ({
  useToast: () => ({
    pushToast: (...args: unknown[]) => pushToastMock(...args),
  }),
}))

const stubs = {
  WindowBoard: {
    props: ['routeKey', 'panes'],
    template: '<div><slot name="permission-overview" /><slot name="permission-detail" /></div>',
  },
  PageHeader: {
    props: ['title', 'subtitle'],
    template: '<header><h1>{{ title }}</h1><p>{{ subtitle }}</p><slot name="actions" /></header>',
  },
  SectionCard: {
    props: ['title', 'subtitle'],
    template: '<section><h2>{{ title }}</h2><p>{{ subtitle }}</p><slot /></section>',
  },
  Button: {
    emits: ['click'],
    template: '<button @click="$emit(\'click\')"><slot /></button>',
  },
  Icon: { template: '<span />' },
  Table: {
    props: ['rows', 'interactiveRows'],
    emits: ['row-click'],
    template:
      '<div><button v-for="row in rows" :key="row.id" :data-row-key="row.id" @click="interactiveRows && $emit(\'row-click\', { rowKey: row.id })">{{ row.id }}</button></div>',
  },
  Input: {
    props: ['modelValue', 'placeholder'],
    emits: ['update:modelValue'],
    template:
      '<input :value="modelValue" :placeholder="placeholder" @input="$emit(\'update:modelValue\', $event.target.value)" />',
  },
  Select: {
    props: ['modelValue', 'options'],
    emits: ['update:modelValue'],
    template:
      '<select :value="modelValue" @change="$emit(\'update:modelValue\', $event.target.value)"><option v-for="item in options" :key="item.value" :value="item.value">{{ item.label }}</option></select>',
  },
}

async function flushAll(): Promise<void> {
  await Promise.resolve()
  await nextTick()
  await Promise.resolve()
  await nextTick()
}

describe('PermissionManagementView', () => {
  beforeEach(() => {
    listCommandsMock.mockReset()
    listSharesMock.mockReset()
    createShareMock.mockReset()
    deleteShareMock.mockReset()
    pushToastMock.mockReset()

    listCommandsMock.mockResolvedValue({
      items: [
        {
          id: 'cmd_share_1',
          tenantId: 't1',
          workspaceId: 'w1',
          ownerId: 'u1',
          visibility: 'PRIVATE',
          acl: [],
          status: 'succeeded',
          createdAt: '2026-02-11T00:00:00Z',
          updatedAt: '2026-02-11T00:00:01Z',
          commandType: 'share.create',
          payload: { resourceType: 'asset', resourceId: 'asset_1' },
          result: { share: { id: 'share_1' } },
          acceptedAt: '2026-02-11T00:00:01Z',
          traceId: 'trace_cmd_1',
        },
      ],
      pageInfo: { page: 1, pageSize: 20, total: 1 },
    })

    listSharesMock.mockResolvedValue({
      items: [
        {
          id: 'share_1',
          tenantId: 't1',
          workspaceId: 'w1',
          resourceType: 'asset',
          resourceId: 'asset_1',
          subjectType: 'user',
          subjectId: 'user_demo',
          permissions: ['READ'],
          createdBy: 'u1',
          createdAt: '2026-02-11T00:00:01Z',
        },
      ],
      pageInfo: { page: 1, pageSize: 20, total: 1 },
    })

    createShareMock.mockResolvedValue({
      resource: { id: 'share_2', status: 'accepted' },
      commandRef: {
        commandId: 'cmd_share_2',
        status: 'accepted',
        acceptedAt: '2026-02-11T00:00:02Z',
      },
    })

    deleteShareMock.mockResolvedValue({
      resource: { id: 'share_1', status: 'deleted' },
      commandRef: {
        commandId: 'cmd_share_3',
        status: 'accepted',
        acceptedAt: '2026-02-11T00:00:03Z',
      },
    })
  })

  it('grants and revokes share policies via form actions', async () => {
    const wrapper = mount(PermissionManagementView, {
      global: {
        plugins: [i18n],
        stubs,
      },
    })

    await flushAll()

    const resourceIdInput = wrapper.find('input[placeholder="例如 asset_123 或 cmd_123"], input[placeholder="e.g. asset_123 or cmd_123"]')
    const subjectIdInput = wrapper.find('input[placeholder="例如 user_a 或 role_editor"], input[placeholder="e.g. user_a or role_editor"]')
    const permissionsInput = wrapper.find('input[placeholder="READ,WRITE"]')

    expect(resourceIdInput.exists()).toBe(true)
    expect(subjectIdInput.exists()).toBe(true)
    expect(permissionsInput.exists()).toBe(true)

    await resourceIdInput.setValue('asset_2')
    await subjectIdInput.setValue('role_editor')
    await permissionsInput.setValue('read,write')

    const grantButton = wrapper
      .findAll('button')
      .find((item) => item.text().includes('授予策略') || item.text().includes('Grant Policy'))
    expect(grantButton).toBeDefined()
    await grantButton!.trigger('click')
    await flushAll()

    expect(createShareMock).toHaveBeenCalledTimes(1)
    expect(createShareMock.mock.calls[0]?.[0]).toMatchObject({
      resourceType: 'asset',
      resourceId: 'asset_2',
      subjectType: 'user',
      subjectId: 'role_editor',
      permissions: ['READ', 'WRITE'],
    })

    const revokeButton = wrapper
      .findAll('button')
      .find((item) => item.text().includes('撤销选中策略') || item.text().includes('Revoke Selected Policy'))
    expect(revokeButton).toBeDefined()
    await revokeButton!.trigger('click')
    await flushAll()

    expect(deleteShareMock).toHaveBeenCalledTimes(1)
    expect(deleteShareMock.mock.calls[0]?.[0]).toBe('share_2')
  })
})
