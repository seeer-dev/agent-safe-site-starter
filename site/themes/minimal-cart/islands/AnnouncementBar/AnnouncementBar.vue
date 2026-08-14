<script setup lang="ts">
import { ref, onMounted, onUnmounted, computed, watch } from 'vue'
import { X, ChevronLeft, ChevronRight } from 'lucide-vue-next'
import { useAnnouncementStore } from '@/shared/stores/announcement'
import { ANNOUNCEMENTS } from '@/shared/lib/mock-data'
import { getIcon } from '@/shared/lib/icon-map'
import { cn } from '@/shared/lib/utils'

const announcement = useAnnouncementStore()

const ROTATE_INTERVAL = 5000

const dismissed = ref(false)
let rotateTimer: ReturnType<typeof setInterval> | null = null

onMounted(() => {
  try {
    if (sessionStorage.getItem('announcement-dismissed') === '1') {
      dismissed.value = true
    }
  } catch { /* ignore */ }
})

watch([dismissed, () => announcement.isPopupOpen], () => {
  if (rotateTimer) { clearInterval(rotateTimer); rotateTimer = null }
  if (!dismissed.value && !announcement.isPopupOpen) {
    rotateTimer = setInterval(() => announcement.next(), ROTATE_INTERVAL)
  }
}, { immediate: true })

onUnmounted(() => {
  if (rotateTimer) clearInterval(rotateTimer)
})

const currentAnnouncement = computed(() => ANNOUNCEMENTS.length > 0 ? ANNOUNCEMENTS[announcement.index] : null)

function dismiss() {
  dismissed.value = true
  try { sessionStorage.setItem('announcement-dismissed', '1') } catch { /* ignore */ }
}

function openPopup() {
  announcement.setPopupOpen(true)
}
</script>

<template>
  <div v-if="!dismissed && currentAnnouncement" class="relative z-30 w-full bg-cta text-cta-foreground">
    <div class="mx-auto flex h-9 max-w-7xl items-center justify-between px-4 sm:px-6 lg:px-8">
      <!-- Left: prev arrow (desktop) -->
      <button
        @click="announcement.prev"
        class="hidden h-6 w-6 shrink-0 items-center justify-center rounded-full transition-colors hover:bg-cta-foreground/15 sm:flex"
        aria-label="上一則公告"
      >
        <ChevronLeft class="h-3.5 w-3.5" />
      </button>

      <!-- Center: rotating message — click to open popup -->
      <button
        @click="openPopup"
        class="relative flex flex-1 items-center justify-center overflow-hidden px-2 py-1 outline-none"
        aria-label="查看公告詳情"
      >
        <Transition name="ann-slide" mode="out-in">
          <div :key="currentAnnouncement.id" class="flex items-center gap-2 text-xs font-medium sm:text-[13px]">
            <component :is="getIcon(currentAnnouncement.icon)" class="h-3.5 w-3.5 shrink-0" />
            <span class="whitespace-nowrap">
              {{ currentAnnouncement.text }}
              <template v-if="currentAnnouncement.highlight">
                ·
                <span class="font-semibold underline underline-offset-2">{{ currentAnnouncement.highlight }}</span>
              </template>
            </span>
          </div>
        </Transition>
      </button>

      <!-- Right: dots + next + close -->
      <div class="flex shrink-0 items-center gap-1.5">
        <div class="hidden items-center gap-1 sm:flex">
          <button
            v-for="(a, i) in ANNOUNCEMENTS"
            :key="a.id"
            @click="announcement.setIndex(i)"
            :aria-label="`第 ${i + 1} 則公告`"
            :class="cn(
              'h-1.5 rounded-full transition-all',
              i === announcement.index
                ? 'w-4 bg-cta-foreground'
                : 'w-1.5 bg-cta-foreground/40 hover:bg-cta-foreground/60'
            )"
          />
        </div>
        <button
          @click="announcement.next"
          class="hidden h-6 w-6 items-center justify-center rounded-full transition-colors hover:bg-cta-foreground/15 sm:flex"
          aria-label="下一則公告"
        >
          <ChevronRight class="h-3.5 w-3.5" />
        </button>
        <div class="mx-0.5 hidden h-4 w-px bg-cta-foreground/25 sm:block" />
        <button
          @click="dismiss"
          class="grid h-6 w-6 place-items-center rounded-full transition-colors hover:bg-cta-foreground/15"
          aria-label="關閉公告列"
        >
          <X class="h-3.5 w-3.5" />
        </button>
      </div>
    </div>
  </div>
</template>

<style scoped>
.ann-slide-enter-active,
.ann-slide-leave-active {
  transition: opacity 0.3s ease, transform 0.3s ease;
}
.ann-slide-enter-from {
  opacity: 0;
  transform: translateY(8px);
}
.ann-slide-leave-to {
  opacity: 0;
  transform: translateY(-8px);
}
</style>
