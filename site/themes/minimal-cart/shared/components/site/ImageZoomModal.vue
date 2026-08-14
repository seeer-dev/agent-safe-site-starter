<script setup lang="ts">
import { ref, watch, onUnmounted } from 'vue'
import {
  X, ChevronLeft, ChevronRight, ZoomIn, ZoomOut, RotateCcw,
} from 'lucide-vue-next'
import { cn } from '@/shared/lib/utils'

const props = defineProps<{
  images: string[]
  alt: string
  open: boolean
  initialIndex: number
}>()

const emit = defineEmits<{
  'update:open': [value: boolean]
}>()

const MIN_SCALE = 1
const MAX_SCALE = 4
const DOUBLE_CLICK_SCALE = 2.5

const index = ref(0)
const scale = ref(1)
const position = ref({ x: 0, y: 0 })
const isDragging = ref(false)

const dragStart = ref<{ x: number; y: number; posX: number; posY: number } | null>(null)
const lastClickTime = ref(0)

watch(() => props.initialIndex, (val) => {
  index.value = val
  scale.value = 1
  position.value = { x: 0, y: 0 }
})

let prevOverflow = ''

watch(() => props.open, (open) => {
  if (open) {
    index.value = props.initialIndex
    scale.value = 1
    position.value = { x: 0, y: 0 }
    prevOverflow = document.body.style.overflow
    document.body.style.overflow = 'hidden'
    document.addEventListener('keydown', handleKeydown, true)
  } else {
    document.body.style.overflow = prevOverflow
    document.removeEventListener('keydown', handleKeydown, true)
  }
})

function reset() {
  scale.value = 1
  position.value = { x: 0, y: 0 }
}

function goNext() {
  index.value = index.value === props.images.length - 1 ? 0 : index.value + 1
  reset()
}

function goPrev() {
  index.value = index.value === 0 ? props.images.length - 1 : index.value - 1
  reset()
}

function zoomIn() {
  scale.value = Math.min(MAX_SCALE, +(scale.value + 0.5).toFixed(2))
}

function zoomOut() {
  const next = Math.max(MIN_SCALE, +(scale.value - 0.5).toFixed(2))
  scale.value = next
  if (next === 1) position.value = { x: 0, y: 0 }
}

function handleWheel(e: WheelEvent) {
  if (!e.ctrlKey && !e.metaKey) return
  e.preventDefault()
  const delta = e.deltaY > 0 ? -0.2 : 0.2
  const next = Math.max(MIN_SCALE, Math.min(MAX_SCALE, +(scale.value + delta).toFixed(2)))
  scale.value = next
  if (next === 1) position.value = { x: 0, y: 0 }
}

function handlePointerDown(e: PointerEvent) {
  const target = e.currentTarget as HTMLElement
  target.setPointerCapture(e.pointerId)
  dragStart.value = { x: e.clientX, y: e.clientY, posX: position.value.x, posY: position.value.y }
  isDragging.value = true
}

function handlePointerMove(e: PointerEvent) {
  if (!isDragging.value || !dragStart.value) return
  if (scale.value <= 1) return
  position.value = {
    x: dragStart.value.posX + (e.clientX - dragStart.value.x),
    y: dragStart.value.posY + (e.clientY - dragStart.value.y),
  }
}

function handlePointerUp(e: PointerEvent) {
  const target = e.currentTarget as HTMLElement
  if (target.hasPointerCapture(e.pointerId)) target.releasePointerCapture(e.pointerId)
  dragStart.value = null
  isDragging.value = false
}

function handleClick() {
  const now = Date.now()
  if (now - lastClickTime.value < 300) {
    if (scale.value === 1) scale.value = DOUBLE_CLICK_SCALE
    else reset()
    lastClickTime.value = 0
    return
  }
  lastClickTime.value = now
  if (scale.value === 1) {
    setTimeout(() => {
      if (Date.now() - lastClickTime.value >= 280) emit('update:open', false)
    }, 290)
  }
}

function handleKeydown(e: KeyboardEvent) {
  if (!props.open) return
  if (e.key === 'Escape') {
    e.stopPropagation()
    e.stopImmediatePropagation()
    emit('update:open', false)
  } else if (e.key === 'ArrowRight') {
    e.stopPropagation()
    goNext()
  } else if (e.key === 'ArrowLeft') {
    e.stopPropagation()
    goPrev()
  } else if (e.key === '+' || e.key === '=') {
    zoomIn()
  } else if (e.key === '-') {
    zoomOut()
  } else if (e.key === '0') {
    reset()
  }
}

onUnmounted(() => {
  document.removeEventListener('keydown', handleKeydown, true)
  document.body.style.overflow = prevOverflow
})
</script>

<template>
  <Teleport to="body">
    <Transition name="zoom-modal">
      <div
        v-if="open"
        class="fixed inset-0 z-[60] flex items-center justify-center bg-background/95 backdrop-blur-md"
        role="dialog"
        :aria-label="`${alt} — 全螢幕檢視`"
      >
        <!-- Top bar -->
        <div class="absolute left-0 right-0 top-0 z-10 flex items-center justify-between px-4 py-4">
          <div class="flex items-center gap-2">
            <span class="rounded-full bg-foreground/80 px-3 py-1 text-xs font-medium text-background backdrop-blur">
              {{ index + 1 }} / {{ images.length }}
            </span>
            <span class="rounded-full bg-foreground/80 px-3 py-1 text-xs font-medium text-background backdrop-blur">
              {{ Math.round(scale * 100) }}%
            </span>
          </div>
          <div class="flex items-center gap-2">
            <button @click="zoomOut" :disabled="scale <= MIN_SCALE" class="grid h-9 w-9 place-items-center rounded-full bg-foreground/80 text-background backdrop-blur transition-opacity hover:bg-foreground disabled:opacity-40" aria-label="Zoom out">
              <ZoomOut class="h-4 w-4" />
            </button>
            <button @click="zoomIn" :disabled="scale >= MAX_SCALE" class="grid h-9 w-9 place-items-center rounded-full bg-foreground/80 text-background backdrop-blur transition-opacity hover:bg-foreground disabled:opacity-40" aria-label="Zoom in">
              <ZoomIn class="h-4 w-4" />
            </button>
            <button @click="reset" :disabled="scale === 1 && position.x === 0 && position.y === 0" class="grid h-9 w-9 place-items-center rounded-full bg-foreground/80 text-background backdrop-blur transition-opacity hover:bg-foreground disabled:opacity-40" aria-label="Reset zoom">
              <RotateCcw class="h-4 w-4" />
            </button>
            <div class="mx-1 h-5 w-px bg-foreground/30" />
            <button @click="emit('update:open', false)" class="grid h-9 w-9 place-items-center rounded-full bg-foreground/80 text-background backdrop-blur transition-opacity hover:bg-foreground" aria-label="Close">
              <X class="h-4 w-4" />
            </button>
          </div>
        </div>

        <!-- Image stage -->
        <div
          class="relative flex h-full w-full items-center justify-center overflow-hidden"
          @wheel="handleWheel"
          @pointerdown="handlePointerDown"
          @pointermove="handlePointerMove"
          @pointerup="handlePointerUp"
          @pointercancel="handlePointerUp"
          @click="handleClick"
          :style="{ cursor: scale > 1 ? (isDragging ? 'grabbing' : 'grab') : 'zoom-in' }"
        >
          <img
            :key="index"
            :src="images[index]"
            :alt="`${alt} — view ${index + 1}`"
            class="pointer-events-none max-h-[90vh] max-w-[90vw] select-none object-contain transition-transform duration-200"
            :style="{ transform: `scale(${scale}) translate(${position.x}px, ${position.y}px)` }"
            draggable="false"
          />

          <div v-if="scale === 1" class="pointer-events-none absolute bottom-20 left-1/2 -translate-x-1/2 rounded-full bg-foreground/70 px-3 py-1.5 text-[11px] text-background backdrop-blur">
            點擊或滾輪縮放 · 拖曳平移 · ← → 切換圖片
          </div>
        </div>

        <!-- Arrows -->
        <template v-if="images.length > 1">
          <button @click.stop="goPrev" class="absolute left-3 top-1/2 z-10 grid h-12 w-12 -translate-y-1/2 place-items-center rounded-full bg-foreground/70 text-background backdrop-blur transition-colors hover:bg-foreground" aria-label="Previous image">
            <ChevronLeft class="h-5 w-5" />
          </button>
          <button @click.stop="goNext" class="absolute right-3 top-1/2 z-10 grid h-12 w-12 -translate-y-1/2 place-items-center rounded-full bg-foreground/70 text-background backdrop-blur transition-colors hover:bg-foreground" aria-label="Next image">
            <ChevronRight class="h-5 w-5" />
          </button>
        </template>

        <!-- Thumbnails -->
        <div v-if="images.length > 1" class="absolute bottom-0 left-0 right-0 z-10 flex justify-center px-4 pb-4 pt-8" style="background: linear-gradient(to top, rgba(0,0,0,0.4), transparent)">
          <div class="flex max-w-full gap-2 overflow-x-auto rounded-full bg-foreground/10 p-1.5 backdrop-blur [scrollbar-width:none] [&::-webkit-scrollbar]:hidden">
            <button
              v-for="(img, i) in images"
              :key="i"
              @click.stop="index = i; reset()"
              :class="cn('relative h-12 w-12 shrink-0 overflow-hidden rounded-full border-2 transition-all', index === i ? 'border-background opacity-100' : 'border-transparent opacity-60 hover:opacity-100')"
              :aria-label="`View image ${i + 1}`"
            >
              <img :src="img" alt="" class="h-full w-full object-cover" draggable="false" />
            </button>
          </div>
        </div>
      </div>
    </Transition>
  </Teleport>
</template>

<style scoped>
.zoom-modal-enter-active, .zoom-modal-leave-active {
  transition: opacity 0.2s ease;
}
.zoom-modal-enter-from, .zoom-modal-leave-to {
  opacity: 0;
}
</style>
