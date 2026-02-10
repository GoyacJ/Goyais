import { nextTick } from 'vue'
import { __resetLayoutSystemForTests, initLayoutSystem, layoutStorageKey, useLayoutStore } from '@/design-system/layout'

describe('layout system', () => {
  beforeEach(() => {
    __resetLayoutSystemForTests()
    localStorage.clear()
  })

  it('uses auto preference by default and follows route default', async () => {
    initLayoutSystem()
    const store = useLayoutStore()

    expect(store.layoutPreference.value).toBe('auto')
    expect(store.effectiveLayout.value).toBe('console')

    store.setRouteLayoutDefault('focus')
    await nextTick()
    expect(store.effectiveLayout.value).toBe('focus')
    expect(document.documentElement.getAttribute('data-layout')).toBe('focus')
  })

  it('manual selection overrides route defaults and persists', async () => {
    initLayoutSystem()
    const store = useLayoutStore()

    store.setRouteLayoutDefault('focus')
    store.setLayoutPreference('topnav')
    await nextTick()

    expect(store.effectiveLayout.value).toBe('topnav')
    expect(localStorage.getItem(layoutStorageKey())).toBe('topnav')

    store.setLayoutPreference('auto')
    await nextTick()

    expect(store.effectiveLayout.value).toBe('focus')
  })

  it('restores stored layout preference', async () => {
    localStorage.setItem(layoutStorageKey(), 'focus')
    initLayoutSystem()

    const store = useLayoutStore()
    await nextTick()

    expect(store.layoutPreference.value).toBe('focus')
    expect(store.effectiveLayout.value).toBe('focus')
    expect(document.documentElement.getAttribute('data-layout-pref')).toBe('focus')
  })
})
