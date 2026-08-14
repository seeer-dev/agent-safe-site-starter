import { ref } from 'vue'

export interface ToastOptions {
  title?: string
  description?: string
  variant?: 'default' | 'destructive'
  duration?: number
  image?: string
}

interface ToastItem extends ToastOptions {
  id: string
  visible: boolean
}

const toasts = ref<ToastItem[]>([])
let counter = 0

function toast(options: ToastOptions): string {
  const id = `toast-${++counter}`
  const duration = options.duration ?? 3000
  toasts.value.push({ ...options, id, visible: true })
  setTimeout(() => dismiss(id), duration)
  return id
}

function dismiss(id: string) {
  const t = toasts.value.find((t) => t.id === id)
  if (t) t.visible = false
  setTimeout(() => {
    toasts.value = toasts.value.filter((t) => t.id !== id)
  }, 300)
}

toast.dismiss = dismiss
toast.success = (msg: string, description?: string) => toast({ title: msg, description })
toast.error = (msg: string, description?: string) => toast({ title: msg, description, variant: 'destructive' })

export function useToast() {
  return { toasts, toast, dismiss }
}

export { toast }
