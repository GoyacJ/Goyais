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
import AlgorithmLibraryView from '@/views/AlgorithmLibraryView.vue'

const listAlgorithmsMock = vi.fn()
const runAlgorithmMock = vi.fn()

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

vi.mock('@/api/registry', () => ({
  listAlgorithms: (...args: unknown[]) => listAlgorithmsMock(...args),
}))

vi.mock('@/api/algorithms', () => ({
  runAlgorithm: (...args: unknown[]) => runAlgorithmMock(...args),
}))

const stubs = {
  WindowBoard: {
    props: ['routeKey', 'panes'],
    template: '<div><slot name="algorithm-list" /><slot name="algorithm-detail" /></div>',
  },
  PageHeader: {
    props: ['title', 'subtitle'],
    template: '<header><h1>{{ title }}</h1><p>{{ subtitle }}</p><slot name="actions" /></header>',
  },
  SectionCard: {
    props: ['title', 'subtitle'],
    template: '<section><h2>{{ title }}</h2><p>{{ subtitle }}</p><slot /></section>',
  },
  Button: { template: '<button @click="$emit(\'click\')"><slot /></button>' },
  Icon: { template: '<span />' },
  Table: {
    props: ['rows'],
    template:
      '<div><button v-for="row in rows" :key="row.id" @click="$emit(\'row-click\', { rowKey: row.id })">{{ row.id }}</button></div>',
  },
  Select: {
    props: ['modelValue', 'options'],
    emits: ['update:modelValue'],
    template:
      '<select :value="modelValue" @change="$emit(\'update:modelValue\', $event.target.value)"><option v-for="item in options" :key="item.value" :value="item.value">{{ item.label }}</option></select>',
  },
  Textarea: {
    props: ['modelValue'],
    emits: ['update:modelValue'],
    template: '<textarea :value="modelValue" @input="$emit(\'update:modelValue\', $event.target.value)" />',
  },
}

async function flushAll(): Promise<void> {
  await Promise.resolve()
  await nextTick()
  await Promise.resolve()
  await nextTick()
}

describe('AlgorithmLibraryView', () => {
  beforeEach(() => {
    listAlgorithmsMock.mockReset()
    runAlgorithmMock.mockReset()
    listAlgorithmsMock.mockResolvedValue({
      items: [
        {
          id: 'algo_1',
          tenantId: 't1',
          workspaceId: 'w1',
          ownerId: 'u1',
          visibility: 'PRIVATE',
          acl: [],
          status: 'enabled',
          createdAt: '2026-02-11T00:00:00Z',
          updatedAt: '2026-02-11T00:00:00Z',
          name: 'detector',
          version: 'v1',
          templateRef: 'tpl_1',
          defaults: {},
          constraints: {},
          dependencies: {},
        },
      ],
      pageInfo: { page: 1, pageSize: 20, total: 1 },
    })
    runAlgorithmMock.mockResolvedValue({
      resource: {
        id: 'algo_run_1',
        algorithmId: 'algo_1',
        workflowRunId: 'run_1',
        status: 'succeeded',
        outputs: {},
        assetIds: ['asset_1'],
        createdAt: '2026-02-11T00:00:00Z',
        updatedAt: '2026-02-11T00:00:01Z',
      },
      commandRef: {
        commandId: 'cmd_algo_1',
        status: 'succeeded',
        acceptedAt: '2026-02-11T00:00:01Z',
      },
    })
  })

  it('runs algorithm with json input and renders result', async () => {
    const wrapper = mount(AlgorithmLibraryView, {
      global: {
        plugins: [i18n],
        stubs,
      },
    })
    await flushAll()

    const textarea = wrapper.find('textarea')
    await textarea.setValue('{"imageId":"asset_1"}')
    const runButton = wrapper
      .findAll('button')
      .find((item) => item.text().includes('运行算法') || item.text().includes('Run Algorithm'))
    expect(runButton).toBeDefined()
    await runButton!.trigger('click')
    await flushAll()

    expect(runAlgorithmMock).toHaveBeenCalledTimes(1)
    expect(runAlgorithmMock.mock.calls[0]?.[0]).toBe('algo_1')
    expect(runAlgorithmMock.mock.calls[0]?.[1]).toMatchObject({
      mode: 'sync',
      inputs: { imageId: 'asset_1' },
    })
    expect(wrapper.text()).toContain('run_1')
  })
})
