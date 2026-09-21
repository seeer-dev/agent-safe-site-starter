<script setup lang="ts">
import LogoMark from './LogoMark.vue'
</script>

<template>
  <!-- 載入畫面用置中 Logo（描繪動畫 + 細線進度，靜謐不循環） -->
  <div class="flex flex-col items-center gap-6">
    <LogoMark :size="88" light />
    <div class="loading-text flex flex-col items-center gap-2.5">
      <span class="pl-[0.5em] font-serif text-xl font-medium tracking-[0.5em] text-cream/95">質選所</span>
      <span class="pl-[0.55em] font-display text-[11px] font-medium uppercase tracking-[0.55em] text-cream/45">Curatory</span>
      <div class="mt-1.5 h-px w-36 overflow-hidden bg-cream/12">
        <div class="loading-bar h-full w-1/4 bg-gradient-to-r from-transparent via-[#D69A6C]/80 to-transparent" />
      </div>
    </div>
  </div>
</template>

<style scoped>
.loading-text {
  opacity: 0;
  transform: translateY(8px);
  animation: loading-in 0.5s cubic-bezier(0.22, 1, 0.36, 1) 0.42s forwards;
}
.loading-bar {
  animation: loading-slide 1.1s ease-in-out 0.3s both;
}
@keyframes loading-in {
  to {
    opacity: 1;
    transform: translateY(0);
  }
}
@keyframes loading-slide {
  from {
    transform: translateX(-100%);
  }
  to {
    transform: translateX(400%);
  }
}
@media (prefers-reduced-motion: reduce) {
  .loading-text,
  .loading-bar {
    animation-duration: 0.01s;
  }
}
</style>
