/**
 * SPDX-License-Identifier: Apache-2.0
 * Copyright (c) 2026 Goya
 * Author: Goya
 * Created: 2026-02-11
 * Version: v1.0.0
 * Description: Goyais source file.
 */

import { mount } from '@vue/test-utils'
import i18n from '@/i18n'
import AIWorkbenchView from '@/views/AIWorkbenchView.vue'
import { nextTick } from 'vue'
import { beforeEach, expect, it, vi } from 'vitest'

const listAISessionsMock = vi.fn()
const getAISessionMock = vi.fn()
const getAISessionEventsMock = vi.fn()
const createAISessionMock = vi.fn()
const createAISessionTurnMock = vi.fn()
const archiveAISessionMock = vi.fn()
const previewAIPlanMock = vi.fn()

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

vi.mock('@/api/ai', () => ({
  listAISessions: (...args: unknown[]) => listAISessionsMock(...args),
  getAISession: (...args: unknown[]) => getAISessionMock(...args),
  getAISessionEvents: (...args: unknown[]) => getAISessionEventsMock(...args),
  previewAIPlan: (...args: unknown[]) => previewAIPlanMock(...args),
  createAISession: (...args: unknown[]) => createAISessionMock(...args),
  createAISessionTurn: (...args: unknown[]) => createAISessionTurnMock(...args),
  archiveAISession: (...args: unknown[]) => archiveAISessionMock(...args),
}))

const stubs = {
  WindowBoard: {
    props: ['routeKey', 'panes'],
    template: '<div><slot name="ai-sessions" /><slot name="ai-composer" /><slot name="ai-events" /></div>',
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
  LogPanel: { props: ['lines'], template: '<div data-testid="log-lines">{{ lines.join("\\n") }}</div>' },
  Badge: { template: '<span><slot /></span>' },
  Icon: { template: '<span />' },
  Button: { template: '<button @click="$emit(\'click\')"><slot /></button>' },
  Input: {
    props: ['modelValue'],
    emits: ['update:modelValue'],
    template: '<input :value="modelValue" @input="$emit(\'update:modelValue\', $event.target.value)" />',
  },
  Textarea: {
    props: ['modelValue'],
    emits: ['update:modelValue'],
    template: '<textarea :value="modelValue" @input="$emit(\'update:modelValue\', $event.target.value)" />',
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
}

describe('AIWorkbenchView', () => {
  beforeEach(() => {
    listAISessionsMock.mockReset()
    getAISessionMock.mockReset()
    getAISessionEventsMock.mockReset()
    createAISessionMock.mockReset()
    createAISessionTurnMock.mockReset()
    archiveAISessionMock.mockReset()
    previewAIPlanMock.mockReset()

    listAISessionsMock.mockResolvedValue({
      items: [
        {
          id: 'sess_1',
          tenantId: 't1',
          workspaceId: 'w1',
          ownerId: 'u1',
          visibility: 'PRIVATE',
          acl: [],
          status: 'active',
          title: 'demo session',
          goal: 'smoke',
          inputs: {},
          constraints: {},
          preferences: {},
          createdAt: '2026-02-11T00:00:00Z',
          updatedAt: '2026-02-11T00:00:00Z',
        },
      ],
      pageInfo: { page: 1, pageSize: 20, total: 1 },
    })
    getAISessionMock.mockResolvedValue({
      id: 'sess_1',
      tenantId: 't1',
      workspaceId: 'w1',
      ownerId: 'u1',
      visibility: 'PRIVATE',
      acl: [],
      status: 'active',
      title: 'demo session',
      goal: 'smoke',
      inputs: {},
      constraints: {},
      preferences: {},
      createdAt: '2026-02-11T00:00:00Z',
      updatedAt: '2026-02-11T00:00:00Z',
    })
    getAISessionEventsMock.mockResolvedValue([
      { id: 'turn_1', event: 'ai.turn.user', data: { role: 'user', content: 'hello' } },
      { id: 'turn_2', event: 'ai.turn.assistant', data: { role: 'assistant', content: 'Plan drafted' } },
      {
        id: 'cmd_1',
        event: 'command.succeeded',
        data: { commandId: 'cmd_1', commandType: 'workflow.run', status: 'succeeded', updatedAt: '2026-02-11T00:00:11Z' },
      },
      {
        id: 'cmd_2',
        event: 'command.failed',
        data: {
          commandId: 'cmd_2',
          commandType: 'workflow.run',
          status: 'failed',
          errorCode: 'AUTHZ_DENIED',
          messageKey: 'error.authz.forbidden',
          updatedAt: '2026-02-11T00:00:12Z',
        },
      },
      {
        id: 'run_1',
        event: 'workflow.run.succeeded',
        data: { runId: 'run_1', status: 'succeeded', commandId: 'cmd_1', commandType: 'workflow.run' },
      },
    ])
    createAISessionMock.mockResolvedValue({
      resource: { id: 'sess_2', status: 'active' },
      commandRef: { commandId: 'cmd_ai_create', status: 'succeeded', acceptedAt: '2026-02-11T00:00:10Z' },
    })
    createAISessionTurnMock.mockResolvedValue({
      resource: { id: 'turn_3', status: 'succeeded' },
      commandRef: { commandId: 'cmd_ai_turn', status: 'succeeded', acceptedAt: '2026-02-11T00:00:11Z' },
    })
    previewAIPlanMock.mockResolvedValue({
      plan: {
        commandType: 'workflow.run',
        payload: {
          templateId: 'tpl_1',
          mode: 'sync',
          inputs: {},
        },
        planner: 'workflow.run',
        reason: 'matched_workflow_run',
        suggestions: [],
        score: 0.91,
        steps: [
          {
            order: 1,
            segment: 'run workflow tpl_1',
            commandType: 'workflow.run',
            payload: { templateId: 'tpl_1' },
            planner: 'workflow.run',
            reason: 'matched_workflow_run',
            score: 0.91,
            executable: true,
          },
        ],
        strategyScores: [
          {
            strategy: 'single_step',
            score: 0.91,
            selected: true,
            reason: 'high_confidence',
          },
        ],
      },
    })
    archiveAISessionMock.mockResolvedValue({
      resource: { id: 'sess_1', status: 'archived' },
      commandRef: { commandId: 'cmd_ai_archive', status: 'succeeded', acceptedAt: '2026-02-11T00:00:12Z' },
    })
  })

  it('loads sessions and submits turn', async () => {
    const wrapper = mount(AIWorkbenchView, {
      global: {
        plugins: [i18n],
        stubs,
      },
    })

    await flushAll()

    expect(wrapper.text()).toContain('AI 工作台')
    expect(wrapper.text()).toContain('demo session')
    expect(wrapper.text()).toContain('AUTHZ_DENIED')

    const textarea = wrapper.find('textarea')
    await textarea.setValue('run workflow tpl_1')
    const sendButton = wrapper
      .findAll('button')
      .find((item) => item.text().includes('发送回合') || item.text().includes('Send Turn'))
    await sendButton?.trigger('click')
    await flushAll()

    expect(previewAIPlanMock).toHaveBeenCalled()
    expect(createAISessionTurnMock).toHaveBeenCalledTimes(1)
    expect(createAISessionTurnMock.mock.calls[0]?.[0]).toBe('sess_1')
    expect(createAISessionTurnMock.mock.calls[0]?.[1]).toMatchObject({
      message: 'run workflow tpl_1',
      execute: false,
      intentCommandType: 'workflow.run',
      intentPayload: {
        templateId: 'tpl_1',
      },
    })
  })

  it('omits explicit intent payload when planner returns executable multi-step chain', async () => {
    previewAIPlanMock.mockResolvedValueOnce({
      plan: {
        commandType: 'workflow.run',
        payload: {
          templateId: 'tpl_1',
        },
        planner: 'multi_step.chain',
        reason: 'matched_multi_step_intent',
        suggestions: [],
        score: 0.88,
        steps: [
          {
            order: 1,
            segment: 'run workflow tpl_1',
            commandType: 'workflow.run',
            payload: { templateId: 'tpl_1' },
            planner: 'workflow.run',
            reason: 'matched_workflow_run',
            score: 0.91,
            executable: true,
          },
          {
            order: 2,
            segment: 'run algorithm algo_1',
            commandType: 'algorithm.run',
            payload: { algorithmId: 'algo_1' },
            planner: 'algorithm.run',
            reason: 'matched_algorithm_run',
            score: 0.85,
            executable: true,
          },
        ],
        strategyScores: [
          {
            strategy: 'multi_step_chain',
            score: 0.88,
            selected: true,
            reason: 'contains_two_executable_segments',
          },
        ],
      },
    })

    const wrapper = mount(AIWorkbenchView, {
      global: {
        plugins: [i18n],
        stubs,
      },
    })

    await flushAll()

    const textarea = wrapper.find('textarea')
    await textarea.setValue('run workflow tpl_1; run algorithm algo_1')
    const sendButton = wrapper
      .findAll('button')
      .find((item) => item.text().includes('发送回合') || item.text().includes('Send Turn'))
    await sendButton?.trigger('click')
    await flushAll()

    expect(createAISessionTurnMock).toHaveBeenCalledTimes(1)
    expect(createAISessionTurnMock.mock.calls[0]?.[1]).toMatchObject({
      message: 'run workflow tpl_1; run algorithm algo_1',
      execute: false,
    })
    expect(createAISessionTurnMock.mock.calls[0]?.[1]).not.toHaveProperty('intentCommandType')
    expect(createAISessionTurnMock.mock.calls[0]?.[1]).not.toHaveProperty('intentPayload')
  })
})
