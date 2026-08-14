import { ref, onMounted, onUnmounted, type Ref } from 'vue'

export function useDragScroll() {
  const elRef: Ref<HTMLElement | null> = ref(null)
  const isDragging = ref(false)
  const startX = ref(0)
  const startScrollLeft = ref(0)
  const hasDragged = ref(false)

  function onPointerDown(e: PointerEvent) {
    const el = elRef.value
    if (!el) return
    isDragging.value = true
    hasDragged.value = false
    startX.value = e.pageX
    startScrollLeft.value = el.scrollLeft
    el.setPointerCapture(e.pointerId)
  }

  function onPointerMove(e: PointerEvent) {
    if (!isDragging.value) return
    const el = elRef.value
    if (!el) return
    const delta = e.pageX - startX.value
    if (Math.abs(delta) > 5) hasDragged.value = true
    el.scrollLeft = startScrollLeft.value - delta
  }

  function onPointerUp(e: PointerEvent) {
    const el = elRef.value
    if (el) el.releasePointerCapture(e.pointerId)
    isDragging.value = false
  }

  function onClickCapture(e: Event) {
    if (hasDragged.value) {
      e.preventDefault()
      e.stopPropagation()
      hasDragged.value = false
    }
  }

  onMounted(() => {
    const el = elRef.value
    if (!el) return
    el.addEventListener('pointerdown', onPointerDown)
    el.addEventListener('pointermove', onPointerMove)
    el.addEventListener('pointerup', onPointerUp)
    el.addEventListener('pointercancel', onPointerUp)
    el.addEventListener('click', onClickCapture, true)
  })

  onUnmounted(() => {
    const el = elRef.value
    if (!el) return
    el.removeEventListener('pointerdown', onPointerDown)
    el.removeEventListener('pointermove', onPointerMove)
    el.removeEventListener('pointerup', onPointerUp)
    el.removeEventListener('pointercancel', onPointerUp)
    el.removeEventListener('click', onClickCapture, true)
  })

  return { elRef, isDragging }
}
