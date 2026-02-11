import { afterEach, vi } from 'vitest'

function createMatchMediaResult(matches: boolean): MediaQueryList {
  return {
    matches,
    media: '',
    onchange: null,
    addListener: vi.fn(),
    removeListener: vi.fn(),
    addEventListener: vi.fn(),
    removeEventListener: vi.fn(),
    dispatchEvent: vi.fn(),
  }
}

if (!window.matchMedia) {
  Object.defineProperty(window, 'matchMedia', {
    writable: true,
    value: vi.fn().mockImplementation((query: string) => createMatchMediaResult(!query.includes('max-width'))),
  })
}

if (!window.PointerEvent) {
  // JSDOM fallback for pointer handlers used by window panes.
  // eslint-disable-next-line @typescript-eslint/no-explicit-any
  ;(window as any).PointerEvent = MouseEvent
}

if (typeof window.ResizeObserver === 'undefined') {
  class ResizeObserverStub implements ResizeObserver {
    observe(): void {}
    unobserve(): void {}
    disconnect(): void {}
  }

  // eslint-disable-next-line @typescript-eslint/no-explicit-any
  ;(window as any).ResizeObserver = ResizeObserverStub
}

afterEach(() => {
  localStorage.clear()
  document.documentElement.removeAttribute('data-layout')
  document.documentElement.removeAttribute('data-layout-pref')
})
