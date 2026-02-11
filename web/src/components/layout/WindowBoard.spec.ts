import { flushPromises, mount } from '@vue/test-utils'
import { createMemoryHistory, createRouter } from 'vue-router'
import { vi } from 'vitest'

import i18n from '@/i18n'
import { __resetLayoutSystemForTests, initLayoutSystem } from '@/design-system/layout'
import WindowBoard from '@/components/layout/WindowBoard.vue'

function createTestRouter() {
  const view = { template: '<div>view</div>' }
  return createRouter({
    history: createMemoryHistory(),
    routes: [
      {
        path: '/commands',
        component: view,
        meta: { layoutDefault: 'console', windowManifestKey: 'commands' },
      },
    ],
  })
}

const panes = [
  { id: 'list', title: 'List' },
  { id: 'detail', title: 'Detail' },
]

describe('WindowBoard', () => {
  beforeEach(() => {
    __resetLayoutSystemForTests()
    localStorage.clear()
  })

  it('renders only target pane in immersive mode', async () => {
    const router = createTestRouter()
    await router.push('/commands?wbMode=immersive&wbPane=list')
    await router.isReady()
    initLayoutSystem(router)

    const wrapper = mount(WindowBoard, {
      props: {
        routeKey: 'commands',
        panes,
      },
      slots: {
        list: '<div data-testid="list-pane">List Pane</div>',
        detail: '<div data-testid="detail-pane">Detail Pane</div>',
      },
      global: {
        plugins: [router, i18n],
      },
    })

    await flushPromises()
    expect(wrapper.find('[data-testid="list-pane"]').exists()).toBe(true)
    expect(wrapper.find('[data-testid="detail-pane"]').exists()).toBe(false)
  })

  it('falls back to standard windowed board for invalid immersive pane', async () => {
    const router = createTestRouter()
    await router.push('/commands?wbMode=immersive&wbPane=unknown')
    await router.isReady()
    initLayoutSystem(router)

    const wrapper = mount(WindowBoard, {
      props: {
        routeKey: 'commands',
        panes,
      },
      slots: {
        list: '<div data-testid="list-pane">List Pane</div>',
        detail: '<div data-testid="detail-pane">Detail Pane</div>',
      },
      global: {
        plugins: [router, i18n],
        stubs: {
          WindowPane: {
            props: ['paneId'],
            template: '<section><slot /></section>',
          },
        },
      },
    })

    await flushPromises()
    expect(wrapper.find('[data-testid="list-pane"]').exists()).toBe(true)
    expect(wrapper.find('[data-testid="detail-pane"]').exists()).toBe(true)
  })

  it('opens immersive url in new tab when pane requests pop-out', async () => {
    const router = createTestRouter()
    await router.push('/commands')
    await router.isReady()
    initLayoutSystem(router)

    const openSpy = vi.spyOn(window, 'open').mockImplementation(() => null)

    const wrapper = mount(WindowBoard, {
      props: {
        routeKey: 'commands',
        panes,
      },
      slots: {
        list: '<div>List Pane</div>',
        detail: '<div>Detail Pane</div>',
      },
      global: {
        plugins: [router, i18n],
        stubs: {
          WindowPane: {
            props: ['paneId'],
            emits: ['open-new-page'],
            template:
              '<section><button class="emit-open-btn" @click="$emit(\'open-new-page\', paneId)">open</button><slot /></section>',
          },
        },
      },
    })

    await flushPromises()
    await wrapper.find('.emit-open-btn').trigger('click')

    expect(openSpy).toHaveBeenCalledTimes(1)
    const [href] = openSpy.mock.calls[0] ?? []
    expect(String(href)).toContain('wbMode=immersive')
    expect(String(href)).toContain('wbPane=list')

    openSpy.mockRestore()
  })
})
