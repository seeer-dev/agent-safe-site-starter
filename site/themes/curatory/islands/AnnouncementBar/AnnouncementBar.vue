<script setup lang="ts">
// 頂部墨色跑馬燈（促銷 banner + 置頂公告輪播）— 對應 reference AnnouncementBar
import { computed, onMounted, ref } from 'vue'
import { Sparkles } from 'lucide-vue-next'
import { bootstrap, loadBootstrap } from '@/shared/lib/store'

const ready = ref(false)
onMounted(async () => {
  await loadBootstrap().catch(() => undefined)
  ready.value = true
})

const messages = computed<string[]>(() => {
  const data = bootstrap.data
  if (!data) return []
  const msgs: string[] = []
  const banner = data.settings?.promoBanner
  if (data.settings?.promoBannerEnabled && typeof banner === 'string' && banner.trim()) {
    msgs.push(banner.trim())
  }
  for (const a of data.announcements ?? []) {
    if (a.pinned) msgs.push(a.title)
  }
  return msgs
})

// 重複填滿跑馬燈寬度（2 的倍數以支援 -50% 平滑循環）
const doubled = computed(() => {
  const msgs = messages.value
  if (msgs.length === 0) return []
  const seq: string[] = []
  const times = Math.max(2, Math.ceil(6 / msgs.length))
  for (let i = 0; i < times; i++) seq.push(...msgs)
  return [...seq, ...seq]
})
</script>

<template>
  <a
    v-if="ready && doubled.length > 0"
    href="/news/"
    class="group relative block h-8 w-full shrink-0 cursor-pointer overflow-hidden bg-ink text-cream"
    :aria-label="`最新消息：${messages[0]}（點擊查看公告）`"
  >
    <div class="absolute inset-0 flex items-center">
      <div class="animate-marquee flex w-max group-hover:[animation-play-state:paused]">
        <span
          v-for="(msg, i) in doubled"
          :key="i"
          class="flex items-center gap-2.5 whitespace-nowrap pr-10 text-[11px] font-light tracking-[0.18em]"
        >
          <Sparkles class="size-3 shrink-0 text-gold/80" />
          {{ msg }}
        </span>
      </div>
    </div>
  </a>
</template>
