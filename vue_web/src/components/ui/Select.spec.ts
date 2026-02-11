/**
 * SPDX-License-Identifier: Apache-2.0
 * Copyright (c) 2026 Goya
 * Author: Goya
 * Created: 2026-02-11
 * Version: v1.0.0
 * Description: Goyais source file.
 */

import { mount } from '@vue/test-utils'
import Select from '@/components/ui/Select.vue'
import i18n from '@/i18n'

describe('Select', () => {
  it('emits update:modelValue when selecting an option from listbox', async () => {
    const wrapper = mount(Select, {
      props: {
        modelValue: 'all',
        options: [
          { label: 'All', value: 'all' },
          { label: 'Running', value: 'running' },
        ],
      },
      global: {
        plugins: [i18n],
      },
      attachTo: document.body,
    })

    await wrapper.find('button').trigger('click')
    await wrapper.findAll('li')[1]?.trigger('click')

    const emissions = wrapper.emitted('update:modelValue')
    expect(emissions?.[0]?.[0]).toBe('running')

    wrapper.unmount()
  })
})
