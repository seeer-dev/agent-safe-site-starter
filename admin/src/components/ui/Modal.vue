<script setup lang="ts">
import { onBeforeUnmount, onMounted, watch, useId } from 'vue'

const props = withDefaults(defineProps<{
  open: boolean
  title?: string
  maxWidth?: string
}>(), {
  title: '',
  maxWidth: 'min(620px, 94vw)',
})

const emit = defineEmits<{
  (e: 'close'): void
}>()

const titleId = useId()

/** Selector for focusable elements inside the dialog. */
const FOCUSABLE = [
  'a[href]',
  'button:not([disabled])',
  'input:not([disabled])',
  'textarea:not([disabled])',
  'select:not([disabled])',
  '[tabindex]:not([tabindex="-1"])',
].join(',')

/** Check if an element is visible. offsetParent is null for display:none
 *  and position:fixed elements. In jsdom (no layout engine), offsetParent
 *  is always null, so we fall back to computed style. */
function isVisible(el: HTMLElement): boolean {
  if (el.offsetParent !== null) return true
  if (el === document.activeElement) return true
  const style = window.getComputedStyle(el)
  return style.display !== 'none' && style.visibility !== 'hidden'
}

function onKeydown(ev: KeyboardEvent) {
  if (!props.open) return
  if (ev.key === 'Escape') {
    emit('close')
    return
  }
  if (ev.key !== 'Tab') return
  const dialog = document.querySelector('.modal[role="dialog"]')
  if (!dialog) return
  const focusable = Array.from(
    dialog.querySelectorAll<HTMLElement>(FOCUSABLE),
  ).filter(isVisible)
  if (focusable.length === 0) {
    // No focusable children - keep focus on the dialog container itself.
    ev.preventDefault()
    ;(dialog as HTMLElement).focus()
    return
  }
  const first = focusable[0]
  const last = focusable[focusable.length - 1]
  const active = document.activeElement as HTMLElement
  if (ev.shiftKey) {
    // Shift+Tab from first -> wrap to last
    if (active === first || !dialog.contains(active)) {
      ev.preventDefault()
      last.focus()
    }
  } else {
    // Tab from last -> wrap to first
    if (active === last || !dialog.contains(active)) {
      ev.preventDefault()
      first.focus()
    }
  }
}

onMounted(() => window.addEventListener('keydown', onKeydown))
onBeforeUnmount(() => window.removeEventListener('keydown', onKeydown))

watch(
  () => props.open,
  (open) => {
    if (open) {
      document.body.style.overflow = 'hidden'
    } else {
      document.body.style.overflow = ''
    }
  },
)

function onBackdrop(ev: MouseEvent) {
  if (ev.target === ev.currentTarget) emit('close')
}
</script>

<template>
  <Teleport to="body">
    <Transition name="modal">
      <div
        v-if="open"
        class="modalback open"
        @mousedown="onBackdrop"
      >
        <div
          class="modal"
          role="dialog"
          aria-modal="true"
          :aria-labelledby="title ? titleId : undefined"
          tabindex="-1"
          :style="{ width: maxWidth }"
        >
          <div class="mh"><b :id="title ? titleId : undefined">{{ title }}</b></div>
          <div class="mb"><slot /></div>
          <div v-if="$slots.footer" class="mf"><slot name="footer" /></div>
        </div>
      </div>
    </Transition>
  </Teleport>
</template>
