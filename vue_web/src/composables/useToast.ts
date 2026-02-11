import { reactive } from 'vue'

export type ToastTone = 'info' | 'success' | 'warn' | 'error'

export interface ToastItem {
  id: number
  title: string
  message: string
  tone: ToastTone
}

const items = reactive<ToastItem[]>([])
let seed = 0

function removeToast(id: number): void {
  const index = items.findIndex((item) => item.id === id)
  if (index >= 0) {
    items.splice(index, 1)
  }
}

function pushToast(payload: Omit<ToastItem, 'id'>, ttlMs = 3000): void {
  seed += 1
  const toast: ToastItem = { id: seed, ...payload }
  items.push(toast)

  if (ttlMs > 0) {
    window.setTimeout(() => removeToast(toast.id), ttlMs)
  }
}

export function useToast() {
  return {
    items,
    pushToast,
    removeToast,
  }
}
