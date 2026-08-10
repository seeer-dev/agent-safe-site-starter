<script setup lang="ts">
import { inject, onMounted, onBeforeUnmount, type Ref } from 'vue'
import { X } from 'lucide-vue-next'

withDefaults(defineProps<{
  closeLabel?: string
  showCloseButton?: boolean
  class?: string
}>(), {
  closeLabel: 'Close',
  showCloseButton: true,
})

const open = inject<Ref<boolean>>('dialog-open')!
const setOpen = inject<(v: boolean) => void>('dialog-set-open')!

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
    <Transition name="dialog-overlay">
      <div
        v-if="open"
        data-slot="dialog-overlay"
        class="dialog-overlay-surface fixed inset-0"
        style="z-index: 50"
        @click="onOverlayClick"
      />
    </Transition>
    <Transition name="dialog-content">
      <div
        v-if="open"
        data-slot="dialog-content"
        role="dialog"
        aria-modal="true"
        class="bg-[var(--admin-surface)] fixed top-[50%] left-[50%] flex max-h-[calc(100vh-2rem)] w-full max-w-[calc(100%-2rem)] translate-x-[-50%] translate-y-[-50%] flex-col gap-4 overflow-hidden rounded-lg border p-6 shadow-lg sm:max-w-lg"
        style="z-index: 51"
        :class="$props.class"
      >
        <slot />
        <button
          v-if="showCloseButton"
          type="button"
          data-slot="dialog-close-button"
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
[data-slot="dialog-overlay"] {
  background-color: var(--admin-overlay-bg);
}
.dialog-overlay-enter-active,
.dialog-overlay-leave-active {
  transition: opacity 0.2s ease;
}
.dialog-overlay-enter-from,
.dialog-overlay-leave-to {
  opacity: 0;
}

/* Use the `scale` property (not `transform`) so the Tailwind `translate`
 * property that centers the dialog is not overridden during the animation.
 * `translate` and `scale` are independent CSS properties that compose. */
.dialog-content-enter-active,
.dialog-content-leave-active {
  transition: opacity 0.2s ease, scale 0.2s ease;
}
.dialog-content-enter-from,
.dialog-content-leave-to {
  opacity: 0;
  scale: 0.95;
}
</style>
