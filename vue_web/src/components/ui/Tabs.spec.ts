/**
 * SPDX-License-Identifier: Apache-2.0
 * Copyright (c) 2026 Goya
 * Author: Goya
 * Created: 2026-02-11
 * Version: v1.0.0
 * Description: Goyais source file.
 */

import { mount } from '@vue/test-utils'
import Tabs, { type TabItem } from '@/components/ui/Tabs.vue'

const tabs: TabItem[] = [
  { id: 'one', label: 'One' },
  { id: 'two', label: 'Two', disabled: true },
  { id: 'three', label: 'Three' },
]

describe('Tabs', () => {
  it('uses roving tabindex and keyboard navigation for horizontal tablists', async () => {
    const wrapper = mount(Tabs, {
      props: {
        modelValue: 'one',
        tabs,
        ariaLabel: 'tabs',
      },
    })

    const buttons = wrapper.findAll('button')
    expect(buttons[0]?.attributes('tabindex')).toBe('0')
    expect(buttons[1]?.attributes('tabindex')).toBe('-1')
    expect(buttons[2]?.attributes('tabindex')).toBe('-1')

    await buttons[0]?.trigger('keydown', { key: 'ArrowRight' })
    await buttons[0]?.trigger('keydown', { key: 'End' })
    await buttons[0]?.trigger('keydown', { key: 'Home' })

    const emissions = wrapper.emitted('update:modelValue')
    expect(emissions?.[0]?.[0]).toBe('three')
    expect(emissions?.[1]?.[0]).toBe('three')
    expect(emissions?.[2]?.[0]).toBe('one')
  })
})
