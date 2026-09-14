<script setup lang="ts">
// 置中 Dialog（對應 radix Dialog）：遮罩 + 縮放淡入 + Esc/遮罩關閉
import { watch, onBeforeUnmount } from 'vue'
import { cn } from '@/shared/lib/utils'

const props = withDefaults(defineProps<{ open: boolean; panelClass?: string }>(), { panelClass: '' })
const emit = defineEmits<{ 'update:open': [value: boolean] }>()

function close() {
  emit('update:open', false)
}
function onKeydown(e: KeyboardEvent) {
  if (e.key === 'Escape') close()
}

watch(
  () => props.open,
  (open) => {
    if (open) {
      document.addEventListener('keydown', onKeydown)
      document.body.style.overflow = 'hidden'
    } else {
      document.removeEventListener('keydown', onKeydown)
      document.body.style.overflow = ''
    }
  },
)
onBeforeUnmount(() => {
  document.removeEventListener('keydown', onKeydown)
  document.body.style.overflow = ''
})
</script>

<template>
  <Teleport to="body">
    <Transition name="dialog-fade">
      <div v-if="open" class="fixed inset-0 z-[140] flex items-start justify-center p-4 sm:items-center">
        <div class="absolute inset-0 bg-ink/45 backdrop-blur-[2px]" aria-hidden="true" @click="close" />
        <div
          role="dialog"
          aria-modal="true"
          :class="cn('dialog-panel relative z-10 w-full max-w-lg rounded-2xl border bg-popover p-6 shadow-2xl', panelClass)"
        >
          <slot />
        </div>
      </div>
    </Transition>
  </Teleport>
</template>

<style scoped>
.dialog-fade-enter-active,
.dialog-fade-leave-active {
  transition: opacity 0.25s ease;
}
.dialog-fade-enter-from,
.dialog-fade-leave-to {
  opacity: 0;
}
.dialog-fade-enter-active .dialog-panel {
  transition: transform 0.3s cubic-bezier(0.22, 1, 0.36, 1), opacity 0.25s ease;
}
.dialog-fade-leave-active .dialog-panel {
  transition: transform 0.18s ease-in, opacity 0.18s ease-in;
}
.dialog-fade-enter-from .dialog-panel {
  transform: translateY(10px) scale(0.98);
  opacity: 0;
}
.dialog-fade-leave-to .dialog-panel {
  transform: translateY(6px) scale(0.98);
  opacity: 0;
}
@media (prefers-reduced-motion: reduce) {
  .dialog-fade-enter-active,
  .dialog-fade-leave-active,
  .dialog-fade-enter-active .dialog-panel,
  .dialog-fade-leave-active .dialog-panel {
    transition: none;
  }
}
</style>
