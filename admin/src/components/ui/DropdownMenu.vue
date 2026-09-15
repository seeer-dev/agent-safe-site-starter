<script setup lang="ts">
import { ref, onBeforeUnmount, nextTick } from 'vue'
import { ChevronDown, Check } from 'lucide-vue-next'
import type { MenuItem } from '@/lib/types'

const props = withDefaults(defineProps<{
  items: MenuItem[]
  /** Trigger text when the trigger slot is not provided. */
  label?: string
  /** Menu alignment against the trigger. */
  align?: 'start' | 'end'
  disabled?: boolean
  ariaLabel?: string
  /** Extra class on the trigger button (e.g. "flt" for filter style). */
  triggerClass?: string
}>(), {
  items: () => [],
  label: '',
  align: 'end',
  disabled: false,
  ariaLabel: '',
  triggerClass: '',
})

const emit = defineEmits<{
  select: [key: string]
}>()

const open = ref(false)
const activeIdx = ref(-1)
const root = ref<HTMLElement | null>(null)
const menuEl = ref<HTMLElement | null>(null)
const menuStyle = ref<{ top: string; left: string }>({ top: '0px', left: '0px' })

function onDocClick(e: MouseEvent) {
  const t = e.target as Node
  if (root.value?.contains(t) || menuEl.value?.contains(t)) return
  close()
}

function onDocKeydown(e: KeyboardEvent) {
  if (e.key !== 'Escape') return
  e.preventDefault()
  close()
  root.value?.querySelector<HTMLElement>('.ddown-trigger')?.focus()
}

// The menu is teleported to <body> and fixed-positioned off the trigger's
// bounding rect so it escapes overflow clipping (.rtable-wrap scroll
// container, .panel overflow:hidden) and the stacking contexts created by
// sticky pinned table cells. Reposition on any scroll (capture reaches
// nested scrollers like .rtable-wrap) or resize.
function reposition() {
  const trigger = root.value?.querySelector<HTMLElement>('.ddown-trigger')
  const menu = menuEl.value
  if (!trigger || !menu) return
  const r = trigger.getBoundingClientRect()
  const mw = menu.offsetWidth
  const mh = menu.offsetHeight
  let top = r.bottom + 4
  if (top + mh > window.innerHeight - 8 && r.top - 4 - mh > 8) {
    top = r.top - 4 - mh
  }
  const left = props.align === 'end' ? r.right - mw : r.left
  menuStyle.value = {
    top: `${top}px`,
    left: `${Math.max(8, Math.min(left, window.innerWidth - mw - 8))}px`,
  }
}

async function openMenu() {
  if (props.disabled || props.items.length === 0) return
  open.value = true
  activeIdx.value = props.items.findIndex((i) => !i.disabled)
  document.addEventListener('mousedown', onDocClick, true)
  document.addEventListener('keydown', onDocKeydown, true)
  window.addEventListener('resize', reposition)
  window.addEventListener('scroll', reposition, true)
  await nextTick()
  reposition()
}

function close() {
  if (!open.value) return
  open.value = false
  document.removeEventListener('mousedown', onDocClick, true)
  document.removeEventListener('keydown', onDocKeydown, true)
  window.removeEventListener('resize', reposition)
  window.removeEventListener('scroll', reposition, true)
}

function toggle() {
  if (open.value) close()
  else openMenu()
}

function choose(item: MenuItem) {
  if (item.disabled) return
  emit('select', item.key)
  close()
  nextTick(() => {
    root.value?.querySelector<HTMLElement>('.ddown-trigger')?.focus()
  })
}

function moveActive(dir: 1 | -1) {
  const n = props.items.length
  if (!n) return
  let i = activeIdx.value
  for (let step = 0; step < n; step++) {
    i = (i + dir + n) % n
    if (!props.items[i].disabled) break
  }
  activeIdx.value = i
}

function onKeydown(e: KeyboardEvent) {
  if (!open.value) {
    if (e.key === 'ArrowDown' || e.key === 'Enter' || e.key === ' ') {
      e.preventDefault()
      openMenu()
    }
    return
  }
  if (e.key === 'Escape') {
    e.preventDefault()
    close()
    root.value?.querySelector<HTMLElement>('.ddown-trigger')?.focus()
    return
  }
  if (e.key === 'ArrowDown') { e.preventDefault(); moveActive(1); return }
  if (e.key === 'ArrowUp') { e.preventDefault(); moveActive(-1); return }
  if (e.key === 'Enter' || e.key === ' ') {
    e.preventDefault()
    const it = props.items[activeIdx.value]
    if (it) choose(it)
  }
}

onBeforeUnmount(() => {
  document.removeEventListener('mousedown', onDocClick, true)
  document.removeEventListener('keydown', onDocKeydown, true)
  window.removeEventListener('resize', reposition)
  window.removeEventListener('scroll', reposition, true)
})
</script>

<template>
  <div ref="root" class="ddown" @keydown="onKeydown">
    <button
      type="button"
      class="ddown-trigger"
      :class="triggerClass"
      :disabled="disabled"
      :aria-expanded="open"
      aria-haspopup="menu"
      :aria-label="ariaLabel || undefined"
      @click="toggle"
    >
      <slot name="trigger">{{ label }}</slot>
      <ChevronDown class="ddown-caret" aria-hidden="true" />
    </button>
    <Teleport to="body">
    <div
      v-if="open"
      ref="menuEl"
      class="ddown-menu"
      role="menu"
      :style="menuStyle"
      @keydown="onKeydown"
    >
      <button
        v-for="(it, i) in items"
        :key="it.key"
        type="button"
        role="menuitem"
        class="ddown-item"
        :class="{ danger: it.danger, active: i === activeIdx, checked: it.checked }"
        :disabled="it.disabled"
        :title="it.hint"
        @click="choose(it)"
        @mouseenter="activeIdx = i"
      >
        <Check v-if="it.checked" class="ddown-check" aria-hidden="true" />
        <span class="ddown-item-body">
          <span>{{ it.label }}</span>
          <small v-if="it.hint" class="ddown-hint">{{ it.hint }}</small>
        </span>
      </button>
    </div>
    </Teleport>
  </div>
</template>
