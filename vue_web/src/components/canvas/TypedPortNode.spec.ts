/**
 * SPDX-License-Identifier: Apache-2.0
 * Copyright (c) 2026 Goya
 * Author: Goya
 * Created: 2026-02-11
 * Version: v1.0.0
 * Description: Goyais source file.
 */

import { mount } from '@vue/test-utils'
import { computed } from 'vue'
import { describe, expect, it } from 'vitest'

import { canvasStepRuntimeByKeyKey } from '@/components/canvas/runtime'
import TypedPortNode from '@/components/canvas/TypedPortNode.vue'

function mountNode(withRuntime: boolean) {
  const provide: Record<string | symbol, unknown> = {}
  if (withRuntime) {
    provide[canvasStepRuntimeByKeyKey as symbol] = computed(() => ({
      step_a: {
        status: 'running',
        durationMs: 38,
        artifactCount: 2,
      },
    }))
  }
  const nodeProps = {
    id: 'step_a',
    type: 'typed',
    selected: false,
    connectable: true,
    position: { x: 120, y: 80 },
    dimensions: { width: 180, height: 72 },
    dragging: false,
    resizing: false,
    zIndex: 1,
    events: {},
    data: {
      label: 'HTTP Tool',
      inputType: 'json',
      outputType: 'json',
      nodeType: 'tool.http',
    },
  }
  return mount(TypedPortNode, {
    props: nodeProps as any,
    global: {
      stubs: {
        Handle: { template: '<span class="handle-stub" />' },
      },
      provide,
    },
  })
}

describe('TypedPortNode', () => {
  it('renders runtime summary when step runtime exists', () => {
    const wrapper = mountNode(true)
    expect(wrapper.find('.canvas-node-runtime').exists()).toBe(true)
    expect(wrapper.text()).toContain('running')
    expect(wrapper.text()).toContain('38ms')
    expect(wrapper.text()).toContain('2')
  })

  it('hides runtime summary when no runtime mapping exists', () => {
    const wrapper = mountNode(false)
    expect(wrapper.find('.canvas-node-runtime').exists()).toBe(false)
  })
})
