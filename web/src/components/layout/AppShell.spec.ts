import { mount, flushPromises } from '@vue/test-utils'
import { createRouter, createMemoryHistory } from 'vue-router'
import AppShell from '@/components/layout/AppShell.vue'
import TopNavBar from '@/components/layout/TopNavBar.vue'
import SideNav from '@/components/layout/SideNav.vue'
import i18n from '@/i18n'
import { __resetLayoutSystemForTests, initLayoutSystem, useLayoutStore } from '@/design-system/layout'

function createTestRouter() {
  const view = { template: '<div>view</div>' }
  return createRouter({
    history: createMemoryHistory(),
    routes: [
      {
        path: '/',
        component: view,
        meta: { layoutDefault: 'console' },
      },
      { path: '/canvas', component: view },
      { path: '/commands', component: view },
      { path: '/assets', component: view },
      { path: '/plugins', component: view },
      { path: '/streams', component: view },
      { path: '/settings', component: view },
    ],
  })
}

describe('AppShell layout switching', () => {
  beforeEach(() => {
    __resetLayoutSystemForTests()
    localStorage.clear()
  })

  it('renders SideNav in console mode', async () => {
    const router = createTestRouter()
    await router.push('/')
    await router.isReady()
    initLayoutSystem(router)

    const { setLayoutPreference } = useLayoutStore()
    setLayoutPreference('console')

    const wrapper = mount(AppShell, {
      global: {
        plugins: [router, i18n],
      },
    })

    await flushPromises()
    expect(wrapper.findComponent(SideNav).exists()).toBe(true)
    expect(wrapper.findComponent(TopNavBar).exists()).toBe(false)
  })

  it('renders TopNavBar in topnav mode', async () => {
    const router = createTestRouter()
    await router.push('/')
    await router.isReady()
    initLayoutSystem(router)

    const { setLayoutPreference } = useLayoutStore()
    setLayoutPreference('topnav')

    const wrapper = mount(AppShell, {
      global: {
        plugins: [router, i18n],
      },
    })

    await flushPromises()
    expect(wrapper.findComponent(SideNav).exists()).toBe(false)
    expect(wrapper.findComponent(TopNavBar).exists()).toBe(true)
  })

  it('hides both nav containers in focus mode', async () => {
    const router = createTestRouter()
    await router.push('/')
    await router.isReady()
    initLayoutSystem(router)

    const { setLayoutPreference } = useLayoutStore()
    setLayoutPreference('focus')

    const wrapper = mount(AppShell, {
      global: {
        plugins: [router, i18n],
      },
    })

    await flushPromises()
    expect(wrapper.findComponent(SideNav).exists()).toBe(false)
    expect(wrapper.findComponent(TopNavBar).exists()).toBe(false)
  })
})
