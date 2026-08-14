<script setup lang="ts">
import { ref, onMounted, onUnmounted, computed } from 'vue'
import {
  ChevronLeft, ChevronRight, ArrowDown, ShoppingBag,
} from 'lucide-vue-next'
import { HERO_SLIDES, HERO_STATS } from '@/shared/lib/mock-data'
import { getIcon } from '@/shared/lib/icon-map'
import CountUp from '@/shared/components/site/CountUp.vue'

const AUTOPLAY_MS = 6000

const index = ref(0)
const isPaused = ref(false)
let timer: ReturnType<typeof setInterval> | null = null

const slides = HERO_SLIDES
const stats = HERO_STATS

const activeSlide = computed(() => slides[index.value])

function goNext() {
  index.value = (index.value + 1) % slides.length
}

function goPrev() {
  index.value = (index.value - 1 + slides.length) % slides.length
}

function goTo(i: number) {
  index.value = i
}

onMounted(() => {
  if (slides.length === 0) return
  timer = setInterval(() => {
    if (!isPaused.value) goNext()
  }, AUTOPLAY_MS)
})

onUnmounted(() => {
  if (timer) clearInterval(timer)
})
</script>

<template>
  <section
    v-if="slides.length > 0"
    id="top"
    class="relative overflow-hidden"
    @mouseenter="isPaused = true"
    @mouseleave="isPaused = false"
  >
    <!-- 大橫幅輪播背景 -->
    <div class="absolute inset-0">
      <Transition name="hero-fade">
        <img
          :key="activeSlide.id"
          :src="activeSlide.image"
          :alt="activeSlide.title"
          class="absolute inset-0 h-full w-full object-cover"
        />
      </Transition>
      <!-- 左側漸層遮罩 -->
      <div
        aria-hidden
        class="absolute inset-0"
        style="background: linear-gradient(95deg, rgba(28,25,23,0.88) 0%, rgba(28,25,23,0.65) 40%, rgba(28,25,23,0.3) 70%, rgba(28,25,23,0.15) 100%)"
      />
      <!-- 底部漸層 -->
      <div
        aria-hidden
        class="absolute inset-x-0 bottom-0 h-48"
        style="background: linear-gradient(to bottom, transparent 0%, rgba(28,25,23,0.85) 100%)"
      />
    </div>

    <!-- 左右切換箭頭（桌面） -->
    <button
      @click="goPrev"
      class="absolute left-3 top-1/2 z-20 hidden h-11 w-11 -translate-y-1/2 place-items-center rounded-full bg-white/20 text-white backdrop-blur transition-colors hover:bg-white/35 md:grid"
      aria-label="上一張"
    >
      <ChevronLeft class="h-5 w-5" />
    </button>
    <button
      @click="goNext"
      class="absolute right-3 top-1/2 z-20 hidden h-11 w-11 -translate-y-1/2 place-items-center rounded-full bg-white/20 text-white backdrop-blur transition-colors hover:bg-white/35 md:grid"
      aria-label="下一張"
    >
      <ChevronRight class="h-5 w-5" />
    </button>

    <div class="relative mx-auto max-w-7xl px-4 sm:px-6 lg:px-8">
      <!-- 上半部：標題與 CTA -->
      <div class="max-w-2xl py-24 text-white sm:py-32 lg:py-40">
        <Transition name="hero-content" mode="out-in">
          <div :key="activeSlide.id">
            <span class="inline-flex items-center gap-2 rounded-full border border-white/30 bg-white/10 px-3 py-1 text-xs font-medium backdrop-blur">
              <span class="h-1.5 w-1.5 rounded-full bg-cta" />
              {{ activeSlide.eyebrow }}
            </span>

            <h1 class="mt-6 text-balance text-4xl font-semibold leading-[1.05] tracking-tight drop-shadow-sm sm:text-5xl lg:text-6xl">
              {{ activeSlide.title }}
            </h1>

            <p class="mt-6 max-w-xl text-pretty text-base leading-relaxed text-white/85 drop-shadow-sm sm:text-lg">
              {{ activeSlide.description }}
            </p>
          </div>
        </Transition>

        <!-- CTA 按鈕（固定） -->
        <div class="mt-10 flex flex-wrap items-center gap-3 card-enter" style="animation-delay: 0.2s">
          <a
            href="#shop"
            class="inline-flex h-11 items-center justify-center rounded-full bg-cta px-7 text-sm font-medium text-cta-foreground shadow-lg transition-transform hover:scale-[1.02] active:scale-[0.98]"
          >
            <ShoppingBag class="mr-1.5 h-4 w-4" />
            立即選購
          </a>
          <a
            href="#shop"
            class="inline-flex h-11 items-center gap-2 rounded-full border border-white/40 bg-white/10 px-7 text-sm font-medium text-white backdrop-blur transition-colors hover:bg-white/20"
          >
            瀏覽分類
            <ArrowDown class="h-4 w-4" />
          </a>
        </div>

        <!-- 輪播指示器 dots -->
        <div class="mt-10 flex items-center gap-2">
          <button
            v-for="(s, i) in slides"
            :key="s.id"
            @click="goTo(i)"
            :class="[
              'h-1.5 rounded-full transition-all',
              i === index ? 'w-8 bg-cta' : 'w-1.5 bg-white/40 hover:bg-white/60',
            ]"
            :aria-label="`第 ${i + 1} 張`"
          />
          <span v-if="isPaused" class="ml-2 text-[10px] text-white/60">已暫停</span>
        </div>
      </div>

      <!-- Stat strip -->
      <div class="card-enter relative grid grid-cols-2 gap-px overflow-hidden rounded-t-2xl border border-white/15 bg-white/10 backdrop-blur-md sm:grid-cols-4" style="animation-delay: 0.4s">
        <div
          v-for="(s, i) in stats"
          :key="s.label"
          class="group relative flex flex-col items-center justify-center gap-1 px-3 py-6 text-center text-white sm:py-7 card-enter"
          :style="{ animationDelay: `${0.5 + i * 0.1}s` }"
        >
          <component :is="getIcon(s.icon)" class="mb-1 h-4 w-4 text-cta/90 transition-transform duration-300 group-hover:scale-110" />
          <div class="text-3xl font-semibold tracking-tight tabular-nums drop-shadow-sm sm:text-4xl">
            <CountUp :to="s.value" :suffix="s.suffix" :fallback="s.fallback" />
          </div>
          <div class="text-[11px] font-medium uppercase tracking-wider text-white/70 sm:text-xs">
            {{ s.label }}
          </div>
        </div>
      </div>
    </div>
  </section>
</template>

<style scoped>
.hero-fade-enter-active {
  transition: opacity 0.8s ease-in-out, transform 0.8s ease-in-out;
}
.hero-fade-leave-active {
  transition: opacity 0.8s ease-in-out;
}
.hero-fade-enter-from {
  opacity: 0;
  transform: scale(1.04);
}
.hero-fade-leave-to {
  opacity: 0;
}
.hero-content-enter-active {
  transition: all 0.5s ease-out;
}
.hero-content-leave-active {
  transition: all 0.3s ease-in;
}
.hero-content-enter-from {
  opacity: 0;
  transform: translateY(16px);
}
.hero-content-leave-to {
  opacity: 0;
  transform: translateY(-16px);
}
</style>
