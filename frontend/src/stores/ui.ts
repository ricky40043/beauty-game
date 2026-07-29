import { defineStore } from 'pinia'
import { ref } from 'vue'

export type ToastKind = 'success' | 'error' | 'info' | 'warning'

export interface Toast {
  id: number
  kind: ToastKind
  message: string
}

let nextId = 1

export const useUIStore = defineStore('ui', () => {
  const toasts = ref<Toast[]>([])
  const busy = ref(false)

  const push = (kind: ToastKind, message: string, ttl = 2600) => {
    const toast: Toast = { id: nextId++, kind, message }
    toasts.value.push(toast)
    setTimeout(() => dismiss(toast.id), ttl)
  }

  const dismiss = (id: number) => {
    toasts.value = toasts.value.filter((t) => t.id !== id)
  }

  const clear = () => {
    toasts.value = []
  }

  return {
    toasts,
    busy,
    dismiss,
    clear,
    showSuccess: (m: string) => push('success', m),
    showError: (m: string) => push('error', m, 3600),
    showInfo: (m: string) => push('info', m),
    showWarning: (m: string) => push('warning', m, 3200),
  }
})
