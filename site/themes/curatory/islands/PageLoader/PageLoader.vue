<script setup lang="ts">
// 品牌開場 loader — 對應 reference InitialLoader：
// 墨色全屏 + Logo 描繪 + 細線進度，就緒後整屏上滑揭幕。
// SPA 只在首次掛載播一次；MPA 用 sessionStorage 讓同次造訪只播一次。
import { onMounted, ref } from 'vue'
import LogoLoading from '@/shared/components/LogoLoading.vue'
import { loadBootstrap } from '@/shared/lib/store'

const hidden = ref(false)
const leaving = ref(false)

const SESSION_KEY = 'curatory-loaded'

onMounted(() => {
  // 同次造訪已播過 → 直接隱藏（對應 SPA 只掛載一次）。
  try {
    if (sessionStorage.getItem(SESSION_KEY)) {
      hidden.value = true
      return
    }
    sessionStorage.setItem(SESSION_KEY, '1')
  } catch {
    /* private mode — still play once per page */
  }

  const reduced = window.matchMedia('(prefers-reduced-motion: reduce)').matches
  const minElapsed = new Promise<void>((r) => window.setTimeout(r, reduced ? 50 : 900))
  // bootstrap 結果不阻塞揭幕 —— 失敗時頁面仍能顯示靜態內容。
  const ready = loadBootstrap().catch(() => undefined)

  Promise.all([minElapsed, ready]).then(() => {
    leaving.value = true
    window.setTimeout(() => (hidden.value = true), reduced ? 60 : 850)
  })
})
</script>

<template>
  <Teleport to="body">
    <div
      v-if="!hidden"
      class="page-loader fixed inset-0 z-[300] flex items-center justify-center bg-ink"
      :class="{ 'loader-leaving': leaving }"
      aria-label="載入中"
    >
      <div class="loader-inner" :class="{ 'loader-inner-out': leaving }">
        <LogoLoading />
      </div>
    </div>
  </Teleport>
</template>

<style scoped>
.page-loader {
  transition: transform 0.78s cubic-bezier(0.76, 0, 0.24, 1);
}
.page-loader.loader-leaving {
  transform: translateY(-100%);
}
/* 對應 reference：揭幕時內容先淡出並上移，不跟著簾幕一起飛走 */
.loader-inner {
  transition:
    opacity 0.26s ease-out,
    transform 0.26s ease-out;
}
.loader-inner.loader-inner-out {
  opacity: 0;
  transform: translateY(-16px);
}
@media (prefers-reduced-motion: reduce) {
  .page-loader {
    transition-duration: 0.05s;
  }
  .loader-inner {
    transition-duration: 0.01s;
  }
}
</style>
