// 極簡 toast（對應 reference 的 sonner 用法：success/error/message）

import { reactive } from 'vue'

export interface Toast {
  id: number
  kind: 'success' | 'error' | 'message'
  title: string
  description?: string
}

let nextId = 1

export const toasts = reactive<Toast[]>([])

function push(kind: Toast['kind'], title: string, opts?: { description?: string }) {
  const id = nextId++
  toasts.push({ id, kind, title, description: opts?.description })
  window.setTimeout(() => dismissToast(id), 3600)
}

export function dismissToast(id: number) {
  const i = toasts.findIndex((t) => t.id === id)
  if (i >= 0) toasts.splice(i, 1)
}

export const toast = {
  success: (title: string, opts?: { description?: string }) => push('success', title, opts),
  error: (title: string, opts?: { description?: string }) => push('error', title, opts),
  message: (title: string, opts?: { description?: string }) => push('message', title, opts),
}
