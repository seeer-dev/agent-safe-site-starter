<script setup lang="ts">
import { ref, computed, watch, onMounted } from 'vue'
import { X, ChevronLeft, ChevronRight, ArrowRight } from 'lucide-vue-next'
import { useAnnouncementStore } from '@/shared/stores/announcement'
import { ANNOUNCEMENTS } from '@/shared/lib/mock-data'
import { getIcon } from '@/shared/lib/icon-map'

const announcement = useAnnouncementStore()

const DISMISS_KEY = 'welcome-popup-dismissed'
const DISMISS_DURATION = 7 * 24 * 60 * 60 * 1000

const isVisible = ref(false)

onMounted(() => {
  if (ANNOUNCEMENTS.length === 0) return
  const raw = localStorage.getItem(DISMISS_KEY)
  if (raw) {
    const dismissedAt = parseInt(raw, 10)
    if (Date.now() - dismissedAt < DISMISS_DURATION) return
  }
  setTimeout(() => { isVisible.value = true }, 1200)
})

const currentAnnouncement = computed(() => ANNOUNCEMENTS.length > 0 ? ANNOUNCEMENTS[announcement.index] : null)

function close() {
  isVisible.value = false
  localStorage.setItem(DISMISS_KEY, Date.now().toString())
}

function cta() {
  close()
}

watch(() => announcement.isPopupOpen, (val) => {
  if (val) isVisible.value = true
  else close()
})

watch(isVisible, (val) => {
  if (val) document.body.style.overflow = 'hidden'
  else document.body.style.overflow = ''
})
</script>

<template>
  <Teleport to="body">
    <Transition name="popup">
      <div
        v-if="isVisible && currentAnnouncement"
        class="fixed inset-0 z-[70] flex items-center justify-center p-4"
        @click.self="close"
      >
        <div class="absolute inset-0 bg-foreground/40 backdrop-blur-sm" />

        <div class="relative z-10 w-full max-w-md overflow-hidden rounded-2xl bg-background shadow-2xl">
          <!-- Close -->
          <button
            @click="close"
            class="absolute right-3 top-3 z-20 grid h-8 w-8 place-items-center rounded-full bg-background/90 text-foreground shadow-sm backdrop-blur transition-colors hover:bg-background"
            aria-label="關閉"
          >
            <X class="h-4 w-4" />
          </button>

          <!-- Image -->
          <div class="relative aspect-[16/10] overflow-hidden bg-gradient-to-br from-stone-700 via-stone-800 to-stone-900">
            <div class="absolute inset-0 bg-gradient-to-t from-black/60 to-transparent" />
            <div class="absolute bottom-3 left-4 flex items-center gap-2 text-white">
              <component :is="getIcon(currentAnnouncement.icon)" class="h-4 w-4" />
              <span class="text-sm font-medium">{{ currentAnnouncement.text }}</span>
            </div>
          </div>

          <!-- Content -->
          <div class="p-6">
            <Transition name="fade" mode="out-in">
              <div :key="announcement.index">
                <h2 class="text-lg font-semibold tracking-tight">{{ currentAnnouncement.popupTitle }}</h2>
                <p class="mt-2 text-sm leading-relaxed text-muted-foreground">
                  {{ currentAnnouncement.popupDescription }}
                </p>
              </div>
            </Transition>

            <!-- Nav -->
            <div class="mt-5 flex items-center justify-between">
              <div class="flex items-center gap-1.5">
                <button
                  @click="announcement.prev"
                  class="grid h-7 w-7 place-items-center rounded-full border border-border text-muted-foreground transition-colors hover:bg-muted"
                  aria-label="上一則"
                >
                  <ChevronLeft class="h-3.5 w-3.5" />
                </button>
                <div class="flex gap-1">
                  <span
                    v-for="(_, i) in ANNOUNCEMENTS"
                    :key="i"
                    :class="[
                      'h-1.5 rounded-full transition-all',
                      i === announcement.index ? 'w-4 bg-cta' : 'w-1.5 bg-muted-foreground/30',
                    ]"
                  />
                </div>
                <button
                  @click="announcement.next"
                  class="grid h-7 w-7 place-items-center rounded-full border border-border text-muted-foreground transition-colors hover:bg-muted"
                  aria-label="下一則"
                >
                  <ChevronRight class="h-3.5 w-3.5" />
                </button>
              </div>

              <button
                @click="cta"
                class="inline-flex h-9 items-center gap-1.5 rounded-full bg-cta px-4 text-sm font-medium text-cta-foreground transition-colors hover:brightness-95"
              >
                立即查看
                <ArrowRight class="h-3.5 w-3.5" />
              </button>
            </div>
          </div>
        </div>
      </div>
    </Transition>
  </Teleport>
</template>

<style scoped>
.popup-enter-active, .popup-leave-active {
  transition: opacity 0.3s ease;
}
.popup-enter-from, .popup-leave-to {
  opacity: 0;
}
.fade-enter-active, .fade-leave-active {
  transition: opacity 0.2s ease;
}
.fade-enter-from, .fade-leave-to {
  opacity: 0;
}
</style>
