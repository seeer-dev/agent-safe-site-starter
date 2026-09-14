<script setup lang="ts">
// 質選所 CURATORY 品牌識別 — 細線雙環 + 中央「質」字
// animate=true 時播放一次性描繪動畫（CSS stroke-dashoffset，對應
// reference 的 framer-motion pathLength）。

withDefaults(defineProps<{ size?: number; animate?: boolean }>(), {
  size: 40,
  animate: false,
})
</script>

<template>
  <svg
    :width="size"
    :height="size"
    viewBox="0 0 72 72"
    class="shrink-0"
    :class="{ 'logo-animate': animate }"
    aria-label="質選所標誌"
    role="img"
  >
    <!-- 外環：完整細環 -->
    <circle
      class="logo-outer"
      cx="36"
      cy="36"
      r="33"
      fill="none"
      stroke="currentColor"
      stroke-width="1.4"
      stroke-linecap="round"
      pathLength="1"
    />
    <!-- 內弧：約 300° 斷口（從 12 點鐘方向順時針） -->
    <path
      class="logo-inner"
      d="M 36 15.5 A 20.5 20.5 0 1 1 23.5 20.5"
      fill="none"
      stroke="currentColor"
      stroke-width="1.2"
      stroke-linecap="round"
      pathLength="1"
    />
    <!-- 內弧端點小點 -->
    <circle class="logo-dot" cx="23.5" cy="20.5" r="1.5" fill="currentColor" />
    <!-- 中央「質」字 -->
    <text
      class="logo-char"
      x="36"
      y="38"
      text-anchor="middle"
      dominant-baseline="central"
      font-size="24"
      :style="{
        fontFamily: 'var(--font-noto-serif), serif',
        fontWeight: 500,
        letterSpacing: '0.02em',
      }"
      fill="currentColor"
    >
      質
    </text>
  </svg>
</template>

<style scoped>
.logo-animate .logo-outer {
  stroke-dasharray: 1;
  stroke-dashoffset: 1;
  animation: logo-draw 0.62s cubic-bezier(0.22, 1, 0.36, 1) forwards;
}
.logo-animate .logo-inner {
  stroke-dasharray: 1;
  stroke-dashoffset: 1;
  animation: logo-draw 0.55s cubic-bezier(0.22, 1, 0.36, 1) 0.14s forwards;
}
.logo-animate .logo-dot {
  opacity: 0;
  transform-origin: 23.5px 20.5px;
  animation: logo-pop 0.24s cubic-bezier(0.22, 1, 0.36, 1) 0.6s forwards;
}
.logo-animate .logo-char {
  opacity: 0;
  animation: logo-fade 0.42s cubic-bezier(0.22, 1, 0.36, 1) 0.2s forwards;
}
@keyframes logo-draw {
  to {
    stroke-dashoffset: 0;
  }
}
@keyframes logo-pop {
  from {
    opacity: 0;
    transform: scale(0);
  }
  to {
    opacity: 1;
    transform: scale(1);
  }
}
@keyframes logo-fade {
  from {
    opacity: 0;
    transform: translateY(3px);
  }
  to {
    opacity: 1;
    transform: translateY(0);
  }
}
@media (prefers-reduced-motion: reduce) {
  .logo-animate .logo-outer,
  .logo-animate .logo-inner,
  .logo-animate .logo-dot,
  .logo-animate .logo-char {
    animation: none;
    opacity: 1;
    stroke-dashoffset: 0;
  }
}
</style>
