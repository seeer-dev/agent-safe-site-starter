import { defineStore } from 'pinia'
import { ref } from 'vue'
import { ANNOUNCEMENTS } from '@/shared/lib/mock-data'

export const useAnnouncementStore = defineStore('announcement', () => {
  const index = ref(0)
  const isPopupOpen = ref(false)

  // Guard against empty announcements (fail-closed: no fabricated content).
  const count = ANNOUNCEMENTS.length

  function setIndex(i: number) {
    if (count === 0) return
    index.value = ((i % count) + count) % count
  }

  function next() {
    if (count === 0) return
    index.value = (index.value + 1) % count
  }

  function prev() {
    if (count === 0) return
    index.value = (index.value - 1 + count) % count
  }

  function setPopupOpen(open: boolean) {
    isPopupOpen.value = open
  }

  return { index, isPopupOpen, setIndex, next, prev, setPopupOpen }
})
