<script setup lang="ts">
import { watch, onUnmounted, nextTick, ref, useId, computed } from 'vue'
import { X } from 'lucide-vue-next'
import { cn } from '@/shared/lib/utils'

const props = withDefaults(defineProps<{
  open: boolean
  title?: string
  description?: string
  class?: string
  showClose?: boolean
  ariaLabel?: string
  labelledBy?: string
}>(), {
  showClose: true,
})

const emit = defineEmits<{
  'update:open': [value: boolean]
  close: []
}>()

const panelRef = ref<HTMLElement | null>(null)
const titleId = useId()
let triggerEl: HTMLElement | null = null
let keydownAttached = false
let focusGeneration = 0
let rafId = 0

const FOCUSABLE = [
  'a[href]',
  'button:not([disabled])',
  'input:not([disabled])',
  'textarea:not([disabled])',
  'select:not([disabled])',
  '[tabindex]:not([tabindex="-1"])',
].join(',')

const labelledById = computed(() => {
  if (props.labelledBy) return props.labelledBy
  if (props.title) return titleId
  return ''
})

function close() {
  emit('update:open', false)
  emit('close')
}

function isVisible(el: HTMLElement): boolean {
  if (el.hasAttribute('disabled') || el.getAttribute('aria-hidden') === 'true') return false
  const withBox = el as HTMLElement & {
    checkVisibility?: (opts?: { checkOpacity?: boolean; checkVisibilityCSS?: boolean }) => boolean
  }
  if (typeof withBox.checkVisibility === 'function') {
    // Ignore opacity so enter-from (opacity: 0) controls stay eligible.
    return withBox.checkVisibility({ checkOpacity: false, checkVisibilityCSS: true })
  }
  let node: HTMLElement | null = el
  while (node && node !== document.documentElement) {
    const style = window.getComputedStyle(node)
    if (style.display === 'none' || style.visibility === 'hidden') return false
    node = node.parentElement
  }
  return true
}

function focusables(): HTMLElement[] {
  const root = panelRef.value
  if (!root) return []
  return Array.from(root.querySelectorAll<HTMLElement>(FOCUSABLE)).filter(isVisible)
}

function focusMovedIntoPanel(): boolean {
  const panel = panelRef.value
  return !!panel && panel.contains(document.activeElement)
}

function focusInitial() {
  const list = focusables()
  if (list.length > 0) list[0].focus()
  const panel = panelRef.value
  if (panel && !panel.contains(document.activeElement)) {
    panel.focus()
  }
}

function cancelPendingFocus() {
  focusGeneration += 1
  if (rafId !== 0) {
    cancelAnimationFrame(rafId)
    rafId = 0
  }
}

function scheduleInitialFocus() {
  const generation = ++focusGeneration
  let waits = 0
  const attempt = () => {
    if (generation !== focusGeneration || !props.open) return
    if (!panelRef.value) {
      // Nested Transition/Teleport often inserts the panel on the next frame.
      if (waits++ >= 2) return
      rafId = requestAnimationFrame(() => {
        rafId = 0
        if (generation !== focusGeneration || !props.open) return
        void nextTick(attempt)
      })
      return
    }
    focusInitial()
    if (!focusMovedIntoPanel() && waits++ < 2) {
      rafId = requestAnimationFrame(() => {
        rafId = 0
        if (generation !== focusGeneration || !props.open) return
        focusInitial()
      })
    }
  }
  void nextTick(attempt)
}

function restoreTrigger() {
  if (triggerEl && document.contains(triggerEl) && typeof triggerEl.focus === 'function') {
    triggerEl.focus()
  }
  triggerEl = null
}

function attach() {
  if (keydownAttached) return
  document.addEventListener('keydown', onKeydown)
  keydownAttached = true
  document.body.style.overflow = 'hidden'
}

function detach() {
  if (keydownAttached) {
    document.removeEventListener('keydown', onKeydown)
    keydownAttached = false
  }
  document.body.style.overflow = ''
}

function onKeydown(e: KeyboardEvent) {
  if (!props.open) return
  if (e.key === 'Escape') {
    e.preventDefault()
    close()
    return
  }
  if (e.key !== 'Tab') return
  const list = focusables()
  const root = panelRef.value
  if (!root) return
  if (list.length === 0) {
    e.preventDefault()
    root.focus()
    return
  }
  const first = list[0]
  const last = list[list.length - 1]
  const active = document.activeElement as HTMLElement | null
  if (e.shiftKey) {
    if (active === first || !root.contains(active)) {
      e.preventDefault()
      last.focus()
    }
  } else if (active === last || !root.contains(active)) {
    e.preventDefault()
    first.focus()
  }
}

watch(() => props.open, (val) => {
  if (val) {
    const active = document.activeElement
    triggerEl = active instanceof HTMLElement ? active : null
    attach()
    scheduleInitialFocus()
  } else {
    cancelPendingFocus()
    detach()
    restoreTrigger()
  }
})

watch(panelRef, (panel) => {
  if (!panel || !props.open) return
  const generation = focusGeneration
  void nextTick(() => {
    if (generation !== focusGeneration || !props.open || panelRef.value !== panel) return
    focusInitial()
  })
}, { flush: 'post' })

onUnmounted(() => {
  cancelPendingFocus()
  detach()
})
</script>

<template>
  <Teleport to="body">
    <Transition name="fade">
      <div v-if="open" class="fixed inset-0 z-50 flex items-center justify-center p-4">
        <div
          class="fixed inset-0 bg-black/60 backdrop-blur-sm"
          @click="close"
        />
        <Transition name="v" appear>
          <div
            v-if="open"
            ref="panelRef"
            role="dialog"
            aria-modal="true"
            :aria-labelledby="labelledById || undefined"
            :aria-label="labelledById ? undefined : (ariaLabel || undefined)"
            tabindex="-1"
            :class="cn(
              'relative z-50 w-full max-h-[calc(100dvh-2rem)] border bg-background shadow-2xl rounded-2xl flex flex-col overflow-hidden',
              props.class || 'max-w-lg p-6',
            )"
          >
            <div v-if="title || description || showClose" class="flex items-start justify-between gap-4 border-b border-border/60 px-6 py-4 shrink-0">
              <div v-if="title || description" class="flex flex-col gap-1">
                <h2 v-if="title" :id="titleId" class="text-lg font-semibold leading-tight tracking-tight">{{ title }}</h2>
                <p v-if="description" class="text-sm text-muted-foreground">{{ description }}</p>
              </div>
              <button
                v-if="showClose"
                type="button"
                class="grid h-8 w-8 shrink-0 place-items-center rounded-full text-muted-foreground transition-colors hover:bg-muted hover:text-foreground"
                aria-label="關閉"
                @click="close"
              >
                <X class="h-4 w-4" />
              </button>
            </div>
            <slot />
          </div>
        </Transition>
      </div>
    </Transition>
  </Teleport>
</template>
