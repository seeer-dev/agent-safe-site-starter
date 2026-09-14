<script setup lang="ts">
// 頁尾 — 對應 reference footer.tsx：墨色品牌欄 + 導覽 + 分類 + 聯絡 + 版權條。
// MPA：所有連結為真實 <a href>；島掛上前模板會輸出等價靜態頁尾，
// 掛上後由此元件接管（資料來自 bootstrap：設定/分類）。
import { computed, onMounted } from 'vue'
import { CreditCard, Mail, MapPin, Phone, Truck } from 'lucide-vue-next'
import LogoLockup from '@/shared/components/LogoLockup.vue'
import { bootstrap, loadBootstrap } from '@/shared/lib/store'

const settings = computed(() => bootstrap.data?.settings)
const categories = computed(() => bootstrap.data?.categories ?? [])
const year = new Date().getFullYear()

const NAV = [
  { label: '首頁', href: '/' },
  { label: '全部商品', href: '/shop/' },
  { label: '最新公告', href: '/news/' },
  { label: '品牌故事', href: '/about/' },
  { label: '購物車', href: '/cart/' },
  { label: '訂單查詢', href: '/track/' },
]

onMounted(() => {
  void loadBootstrap().catch(() => undefined)
})
</script>

<template>
  <footer class="mt-auto bg-ink text-cream/70">
    <div class="mx-auto w-full max-w-7xl px-4 py-12 sm:px-6 sm:py-16 lg:px-8">
      <div class="grid gap-10 md:grid-cols-2 lg:grid-cols-[1.6fr_1fr_1fr_1.2fr]">
        <!-- 品牌欄 -->
        <div>
          <LogoLockup :store-name="settings?.storeName ?? '質選所'" :size="40" light />
          <p class="mt-4 max-w-xs text-sm leading-relaxed text-cream/55">
            {{ settings?.tagline || '為日常，嚴選美好' }}——我們走訪窯場、工坊與工作室，只把親手摸過、真心喜歡的物件帶回來。
          </p>
        </div>

        <!-- 導覽 -->
        <nav aria-label="頁尾導覽">
          <h3 class="text-[11px] font-medium uppercase tracking-[0.32em] text-cream/40">Navigate</h3>
          <ul class="mt-4 space-y-2.5 text-sm">
            <li v-for="l in NAV" :key="l.label">
              <a :href="l.href" class="transition-colors hover:text-cream">{{ l.label }}</a>
            </li>
          </ul>
        </nav>

        <!-- 分類 -->
        <nav aria-label="商品分類">
          <h3 class="text-[11px] font-medium uppercase tracking-[0.32em] text-cream/40">Categories</h3>
          <ul class="mt-4 space-y-2.5 text-sm">
            <li v-for="c in categories.slice(0, 5)" :key="c.id">
              <a :href="`/categories/${c.slug}/`" class="transition-colors hover:text-cream">{{ c.name }}</a>
            </li>
            <li v-if="categories.length === 0" class="text-cream/40">尚無分類</li>
          </ul>
        </nav>

        <!-- 聯絡資訊 -->
        <div>
          <h3 class="text-[11px] font-medium uppercase tracking-[0.32em] text-cream/40">Contact</h3>
          <ul class="mt-4 space-y-3 text-sm">
            <li v-if="settings?.contactEmail" class="flex items-center gap-2.5">
              <Mail class="size-3.5 shrink-0 text-gold/70" />
              <a :href="`mailto:${settings.contactEmail}`" class="transition-colors hover:text-cream">
                {{ settings.contactEmail }}
              </a>
            </li>
            <li v-if="settings?.contactPhone" class="flex items-center gap-2.5">
              <Phone class="size-3.5 shrink-0 text-gold/70" />
              <span>{{ settings.contactPhone }}</span>
            </li>
            <li class="flex items-start gap-2.5">
              <MapPin class="mt-0.5 size-3.5 shrink-0 text-gold/70" />
              <span>
                {{ settings?.serviceHours || '週一至週五 10:00–18:00' }}
                <br />
                <span class="text-cream/45">{{ settings?.addressNote || '台北市大安區（線上選物，無實體門市）' }}</span>
              </span>
            </li>
          </ul>
        </div>
      </div>

      <!-- 支付與物流 -->
      <div class="mt-10 flex flex-wrap items-center gap-x-8 gap-y-3 border-t border-cream/10 pt-6 text-xs text-cream/45">
        <span class="flex items-center gap-1.5">
          <CreditCard class="size-3.5" /> 信用卡・LINE Pay・貨到付款
        </span>
        <span class="flex items-center gap-1.5">
          <Truck class="size-3.5" /> 7-11／全家店到店・黑貓宅配・中華郵政
        </span>
        <span>7 天鑑賞期・安心的退換貨服務</span>
      </div>

      <!-- 版權條 -->
      <div class="mt-6 flex flex-col items-start justify-between gap-3 text-xs text-cream/40 sm:flex-row sm:items-center">
        <p>© {{ year }} {{ settings?.storeName ?? '質選所' }} {{ settings?.storeNameEn ?? 'CURATORY' }}. All rights reserved.</p>
        <div class="flex items-center gap-4">
          <a href="/news/" class="transition-colors hover:text-cream/70">服務條款</a>
          <a href="/news/" class="transition-colors hover:text-cream/70">隱私權政策</a>
          <a href="/admin/" class="border-b border-cream/20 pb-0.5 tracking-wider transition-colors hover:text-cream/70" data-no-transition>管理後台</a>
        </div>
      </div>
    </div>
  </footer>
</template>
