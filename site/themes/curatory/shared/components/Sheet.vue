<script setup lang="ts">
// 右側滑出 Drawer（對應 radix Sheet side="right"）
// 含遮罩、Esc 關閉、滑入滑出動畫、focus 回到觸發前的處理由呼叫端管理。
import { watch, onBeforeUnmount } from 'vue'
import { cn } from '@/shared/lib/utils'

const props = withDefaults(
  defineProps<{ open: boolean; side?: 'right' | 'left'; panelClass?: string }>(),
  { side: 'right', panelClass: '' },
)
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
    <Transition name="sheet-fade">
      <div
        v-if="open"
        class="fixed inset-0 z-[120] bg-ink/45 backdrop-blur-[2px]"
        aria-hidden="true"
        @click="close"
      />
    </Transition>
    <Transition :name="side === 'right' ? 'sheet-right' : 'sheet-left'">
      <div
        v-if="open"
        role="dialog"
        aria-modal="true"
        :class="
          cn(
            'fixed inset-y-0 z-[130] flex w-full flex-col border-border bg-background shadow-2xl sm:max-w-md',
            side === 'right' ? 'right-0 border-l' : 'left-0 border-r',
            panelClass,
          )
        "
      >
        <slot />
      </div>
    </Transition>
  </Teleport>
</template>

<style scoped>
.sheet-fade-enter-active,
.sheet-fade-leave-active {
  transition: opacity 0.3s ease;
}
.sheet-fade-enter-from,
.sheet-fade-leave-to {
  opacity: 0;
}
.sheet-right-enter-active,
.sheet-right-leave-active,
.sheet-left-enter-active,
.sheet-left-leave-active {
  transition: transform 0.4s cubic-bezier(0.22, 1, 0.36, 1);
}
.sheet-right-enter-from,
.sheet-right-leave-to {
  transform: translateX(100%);
}
.sheet-left-enter-from,
.sheet-left-leave-to {
  transform: translateX(-100%);
}
@media (prefers-reduced-motion: reduce) {
  .sheet-fade-enter-active,
  .sheet-fade-leave-active,
  .sheet-right-enter-active,
  .sheet-right-leave-active,
  .sheet-left-enter-active,
  .sheet-left-leave-active {
    transition: none;
  }
}
</style>
