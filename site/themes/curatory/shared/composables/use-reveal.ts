// v-reveal 指令 — 對應 reference 的 framer-motion whileInView
// （淡入上浮 20px / 0.6s / ease [0.22,1,0.36,1] / once / -60px margin）。
//
// 用法：v-reveal 或 v-reveal="{ delay: 0.15, y: 24 }"

import type { Directive } from 'vue'

interface RevealOptions {
  delay?: number
  y?: number
}

let observer: IntersectionObserver | null = null

function getObserver(): IntersectionObserver {
  if (!observer) {
    observer = new IntersectionObserver(
      (entries) => {
        for (const entry of entries) {
          if (entry.isIntersecting) {
            entry.target.classList.add('reveal-in')
            observer?.unobserve(entry.target)
          }
        }
      },
      { rootMargin: '0px 0px -60px 0px', threshold: 0 },
    )
  }
  return observer
}

export const vReveal: Directive<HTMLElement, RevealOptions | undefined> = {
  mounted(el, binding) {
    if (window.matchMedia('(prefers-reduced-motion: reduce)').matches) return
    el.classList.add('reveal-init')
    const y = binding.value?.y ?? 20
    if (y !== 20) el.style.transform = `translateY(${y}px)`
    if (binding.value?.delay) el.style.transitionDelay = `${binding.value.delay}s`
    getObserver().observe(el)
  },
  unmounted(el) {
    observer?.unobserve(el)
  },
}
