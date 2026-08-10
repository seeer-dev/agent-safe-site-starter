<script setup lang="ts">
import { inject, computed, onMounted, onBeforeUnmount, type Ref } from 'vue'
import { X } from 'lucide-vue-next'

const props = withDefaults(defineProps<{
  side?: 'left' | 'right' | 'bottom'
  showCloseButton?: boolean
  closeLabel?: string
  class?: string
}>(), {
  side: 'right',
  showCloseButton: true,
  closeLabel: 'Close',
})

const open = inject<Ref<boolean>>('sheet-open')!
const setOpen = inject<(v: boolean) => void>('sheet-set-open')!

const sideClass = computed(() => {
  switch (props.side) {
    case 'left':
      return 'inset-y-0 left-0 h-full w-[85vw] max-w-xs border-r'
    case 'right':
      return 'inset-y-0 right-0 h-full w-[85vw] max-w-xs border-l'
    case 'bottom':
      return 'inset-x-0 bottom-0 max-h-[85vh] w-full rounded-t-lg border-t'
    default:
      return 'inset-y-0 right-0 h-full w-[85vw] max-w-xs border-l'
  }
})

const transitionName = computed(() => `sheet-${props.side}`)

function onOverlayClick() {
  setOpen(false)
}

function onKeydown(event: KeyboardEvent) {
  if (event.key === 'Escape') {
    event.preventDefault()
    setOpen(false)
  }
}

onMounted(() => {
  window.addEventListener('keydown', onKeydown)
})
onBeforeUnmount(() => {
  window.removeEventListener('keydown', onKeydown)
})
</script>

<template>
  <Teleport to="body">
    <Transition name="sheet-overlay">
      <div
        v-if="open"
        data-slot="sheet-overlay"
        class="sheet-overlay-surface fixed inset-0"
        style="z-index: 50"
        @click="onOverlayClick"
      />
    </Transition>
    <Transition :name="transitionName">
      <div
        v-if="open"
        data-slot="sheet-content"
        :data-side="side"
        role="dialog"
        aria-modal="true"
        class="bg-[var(--admin-surface)] fixed flex flex-col gap-4 shadow-lg p-6"
        :class="[sideClass, $props.class]"
        style="z-index: 51"
      >
        <slot />
        <button
          v-if="showCloseButton"
          type="button"
          data-slot="sheet-close-button"
          class="absolute top-4 right-4 rounded-xs opacity-70 transition-opacity hover:opacity-100"
          :aria-label="closeLabel"
          @click="setOpen(false)"
        >
          <X :size="16" aria-hidden="true" />
        </button>
      </div>
    </Transition>
  </Teleport>
</template>

<style>
[data-slot="sheet-overlay"] {
  background-color: var(--admin-overlay-bg);
}
/* Overlay fade */
.sheet-overlay-enter-active,
.sheet-overlay-leave-active {
  transition: opacity 0.25s ease;
}
.sheet-overlay-enter-from,
.sheet-overlay-leave-to {
  opacity: 0;
}

/* Left slide */
.sheet-left-enter-active,
.sheet-left-leave-active {
  transition: transform 0.25s ease;
}
.sheet-left-enter-from,
.sheet-left-leave-to {
  transform: translateX(-100%);
}

/* Right slide */
.sheet-right-enter-active,
.sheet-right-leave-active {
  transition: transform 0.25s ease;
}
.sheet-right-enter-from,
.sheet-right-leave-to {
  transform: translateX(100%);
}

/* Bottom slide */
.sheet-bottom-enter-active,
.sheet-bottom-leave-active {
  transition: transform 0.25s ease;
}
.sheet-bottom-enter-from,
.sheet-bottom-leave-to {
  transform: translateY(100%);
}
</style>
