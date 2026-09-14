<script setup lang="ts">
// 極簡 toast 呈現層（對應 sonner 的 top-center 浮出 toast）
import { CheckCircle2, AlertCircle, Info, X } from 'lucide-vue-next'
import { toasts, dismissToast } from '@/shared/lib/toast'

const ICONS = {
  success: CheckCircle2,
  error: AlertCircle,
  message: Info,
}
</script>

<template>
  <Teleport to="body">
    <div
      class="pointer-events-none fixed inset-x-0 top-4 z-[400] flex w-full flex-col items-center gap-2 px-4"
      role="status"
      aria-live="polite"
    >
      <TransitionGroup name="toast">
        <div
          v-for="t in toasts"
          :key="t.id"
          class="pointer-events-auto flex w-full max-w-sm items-start gap-3 rounded-xl border bg-popover px-4 py-3 shadow-lg"
        >
          <component
            :is="ICONS[t.kind]"
            class="mt-0.5 size-4 shrink-0"
            :class="t.kind === 'error' ? 'text-destructive' : t.kind === 'success' ? 'text-primary' : 'text-muted-foreground'"
          />
          <div class="min-w-0 flex-1">
            <p class="text-sm font-medium leading-snug">{{ t.title }}</p>
            <p v-if="t.description" class="mt-0.5 text-xs text-muted-foreground">{{ t.description }}</p>
          </div>
          <button
            type="button"
            class="-mr-1 -mt-0.5 shrink-0 rounded-full p-1 text-muted-foreground/60 transition-colors hover:bg-muted hover:text-foreground"
            aria-label="關閉提示"
            @click="dismissToast(t.id)"
          >
            <X class="size-3.5" />
          </button>
        </div>
      </TransitionGroup>
    </div>
  </Teleport>
</template>

<style scoped>
.toast-enter-active {
  transition: transform 0.3s cubic-bezier(0.22, 1, 0.36, 1), opacity 0.3s ease;
}
.toast-leave-active {
  transition: transform 0.2s ease-in, opacity 0.2s ease-in;
}
.toast-enter-from {
  transform: translateY(-12px);
  opacity: 0;
}
.toast-leave-to {
  transform: translateY(-8px);
  opacity: 0;
}
@media (prefers-reduced-motion: reduce) {
  .toast-enter-active,
  .toast-leave-active {
    transition: none;
  }
}
</style>
