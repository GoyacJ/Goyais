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

  it('scrolls the nearest container when dragging near the bottom edge', async () => {
    const host = document.createElement('div')
    host.style.position = 'relative'
    host.style.overflowY = 'auto'
    host.style.overflowX = 'hidden'
    document.body.appendChild(host)

    let scrollTop = 0
    let scrollLeft = 0
    Object.defineProperty(host, 'clientHeight', { configurable: true, value: 120 })
    Object.defineProperty(host, 'scrollHeight', { configurable: true, value: 560 })
    Object.defineProperty(host, 'clientWidth', { configurable: true, value: 320 })
    Object.defineProperty(host, 'scrollWidth', { configurable: true, value: 320 })
    Object.defineProperty(host, 'scrollTop', {
      configurable: true,
      get: () => scrollTop,
      set: (value: number) => {
        scrollTop = value
      },
    })
    Object.defineProperty(host, 'scrollLeft', {
      configurable: true,
      get: () => scrollLeft,
      set: (value: number) => {
        scrollLeft = value
      },
    })
    host.getBoundingClientRect = () =>
      ({
        x: 0,
        y: 0,
        top: 0,
        left: 0,
        right: 320,
        bottom: 120,
        width: 320,
        height: 120,
        toJSON: () => ({}),
      }) as DOMRect

    const wrapper = mount(WindowPane, {
      attachTo: host,
      props: baseProps(),
      slots: { default: '<div>content</div>' },
      global: {
        plugins: [i18n],
      },
    })

    await wrapper.find('.ui-window-pane-header').trigger('pointerdown', {
      button: 0,
      clientX: 120,
      clientY: 40,
    })

    window.dispatchEvent(new PointerEvent('pointermove', { clientX: 120, clientY: 118 }))
    window.dispatchEvent(new PointerEvent('pointerup', { clientX: 120, clientY: 118 }))

    expect(scrollTop).toBeGreaterThan(0)

    const updates = wrapper.emitted('update:rect')
    expect(updates?.length).toBeGreaterThan(0)
    const lastPayload = updates?.[updates.length - 1]?.[0] as { rect: { y: number } }
    expect(lastPayload.rect.y).toBeGreaterThan(108)

    wrapper.unmount()
    host.remove()
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
