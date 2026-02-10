import { mount } from '@vue/test-utils'
import WindowPane from '@/components/layout/WindowPane.vue'
import i18n from '@/i18n'

function baseProps() {
  return {
    paneId: 'list',
    title: 'List',
    rect: { x: 20, y: 30, w: 420, h: 300, z: 1 },
    bounds: { width: 1000, height: 900 },
    minWidth: 260,
    minHeight: 180,
  }
}

describe('WindowPane', () => {
  it('emits rect updates when dragging header', async () => {
    const wrapper = mount(WindowPane, {
      props: baseProps(),
      slots: { default: '<div>content</div>' },
      global: {
        plugins: [i18n],
      },
    })

    await wrapper.find('.ui-window-pane-header').trigger('pointerdown', {
      button: 0,
      clientX: 100,
      clientY: 120,
    })

    window.dispatchEvent(new PointerEvent('pointermove', { clientX: 160, clientY: 180 }))
    window.dispatchEvent(new PointerEvent('pointerup', { clientX: 160, clientY: 180 }))

    const updates = wrapper.emitted('update:rect')
    expect(updates?.length).toBeGreaterThan(0)
    const lastPayload = updates?.[updates.length - 1]?.[0] as { paneId: string; rect: { x: number; y: number } }

    expect(lastPayload.paneId).toBe('list')
    expect(lastPayload.rect.x).toBe(80)
    expect(lastPayload.rect.y).toBe(90)
  })

  it('emits width and height updates when resizing from corner', async () => {
    const wrapper = mount(WindowPane, {
      props: baseProps(),
      slots: { default: '<div>content</div>' },
      global: {
        plugins: [i18n],
      },
    })

    await wrapper.find('[data-testid="resize-corner"]').trigger('pointerdown', {
      button: 0,
      clientX: 420,
      clientY: 330,
    })

    window.dispatchEvent(new PointerEvent('pointermove', { clientX: 500, clientY: 400 }))
    window.dispatchEvent(new PointerEvent('pointerup', { clientX: 500, clientY: 400 }))

    const updates = wrapper.emitted('update:rect')
    expect(updates?.length).toBeGreaterThan(0)

    const lastPayload = updates?.[updates.length - 1]?.[0] as { rect: { w: number; h: number } }
    expect(lastPayload.rect.w).toBeGreaterThan(420)
    expect(lastPayload.rect.h).toBeGreaterThan(300)
  })

  it('supports keyboard move and resize shortcuts from the pane header', async () => {
    const wrapper = mount(WindowPane, {
      props: baseProps(),
      slots: { default: '<div>content</div>' },
      global: {
        plugins: [i18n],
      },
    })

    const header = wrapper.find('.ui-window-pane-header')

    await header.trigger('keydown', {
      key: 'ArrowRight',
      altKey: true,
    })

    await header.trigger('keydown', {
      key: 'ArrowDown',
      altKey: true,
      shiftKey: true,
    })

    const updates = wrapper.emitted('update:rect')
    expect(updates?.length).toBe(2)

    const moved = updates?.[0]?.[0] as { rect: { x: number; y: number } }
    const resized = updates?.[1]?.[0] as { rect: { h: number } }

    expect(moved.rect.x).toBe(36)
    expect(moved.rect.y).toBe(30)
    expect(resized.rect.h).toBe(316)
  })

  it('clamps keyboard operations to board bounds and min size', async () => {
    const wrapper = mount(WindowPane, {
      props: {
        ...baseProps(),
        rect: { x: 0, y: 0, w: 264, h: 184, z: 1 },
      },
      slots: { default: '<div>content</div>' },
      global: {
        plugins: [i18n],
      },
    })

    const header = wrapper.find('.ui-window-pane-header')

    await header.trigger('keydown', {
      key: 'ArrowLeft',
      altKey: true,
    })
    await header.trigger('keydown', {
      key: 'ArrowLeft',
      altKey: true,
      shiftKey: true,
    })

    const updates = wrapper.emitted('update:rect')
    expect(updates?.length).toBe(2)

    const moved = updates?.[0]?.[0] as { rect: { x: number } }
    const resized = updates?.[1]?.[0] as { rect: { w: number } }

    expect(moved.rect.x).toBe(0)
    expect(resized.rect.w).toBe(260)
  })
})
