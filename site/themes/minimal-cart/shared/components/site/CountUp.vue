<script setup lang="ts">
import { ref, onMounted, onUnmounted, watch } from 'vue'

const props = withDefaults(defineProps<{
  to: number
  duration?: number
  suffix?: string
  prefix?: string
  fallback?: string
}>(), {
  duration: 1200,
  suffix: '',
  prefix: '',
})

const value = ref(0)
const started = ref(false)
const elRef = ref<HTMLElement | null>(null)
let observer: IntersectionObserver | null = null
let frame: number | null = null

onMounted(() => {
  const el = elRef.value
  if (!el) return
  observer = new IntersectionObserver(
    (entries) => {
      if (entries[0].isIntersecting && !started.value) {
        started.value = true
        observer?.disconnect()
      }
    },
    { threshold: 0.3 },
  )
  observer.observe(el)
})

onUnmounted(() => {
  observer?.disconnect()
  if (frame) cancelAnimationFrame(frame)
})

watch(started, (val) => {
  if (!val) return
  const startTime = performance.now()
  const animate = (now: number) => {
    const elapsed = now - startTime
    const progress = Math.min(elapsed / props.duration, 1)
    const eased = progress === 1 ? 1 : 1 - Math.pow(2, -10 * progress)
    value.value = Math.round(eased * props.to)
    if (progress < 1) {
      frame = requestAnimationFrame(animate)
    }
  }
  frame = requestAnimationFrame(animate)
})
</script>

<template>
  <span ref="elRef">{{ fallback || `${prefix}${value}${suffix}` }}</span>
</template>
