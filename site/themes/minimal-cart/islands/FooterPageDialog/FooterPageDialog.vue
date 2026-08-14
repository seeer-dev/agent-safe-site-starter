<script setup lang="ts">
import { computed } from 'vue'
import {
  Shield, FileText, Truck, Mail, Info, HelpCircle, X,
} from 'lucide-vue-next'
import type { FooterPageKey } from '@/shared/lib/types'
import { useUiStore } from '@/shared/stores/ui'
import Dialog from '@/shared/components/ui/Dialog.vue'
import Separator from '@/shared/components/ui/Separator.vue'

const ui = useUiStore()

const PAGE_META: Record<FooterPageKey, { title: string; icon: any }> = {
  privacy: { title: '隱私權政策', icon: Shield },
  terms: { title: '服務條款', icon: FileText },
  shipping: { title: '配送資訊', icon: Truck },
  contact: { title: '聯絡我們', icon: Mail },
  about: { title: '關於我們', icon: Info },
  faq: { title: '常見問答', icon: HelpCircle },
}

const currentMeta = computed(() => ui.footerPage ? PAGE_META[ui.footerPage] : null)
</script>

<template>
  <Dialog
    :open="ui.footerPageOpen"
    :show-close="false"
    :aria-label="currentMeta?.title ?? '頁面說明'"
    class="max-w-2xl p-0"
    @update:open="ui.closeFooterPage()"
  >
    <div v-if="currentMeta" class="flex items-center justify-between border-b border-border/60 px-6 py-4">
      <div class="flex items-center gap-2">
        <component :is="currentMeta.icon" class="h-4 w-4 text-cta" />
        <h2 class="text-lg font-semibold tracking-tight">{{ currentMeta.title }}</h2>
      </div>
      <button
        type="button"
        class="grid h-8 w-8 place-items-center rounded-full text-muted-foreground transition-colors hover:bg-muted hover:text-foreground"
        aria-label="關閉"
        @click="ui.closeFooterPage()"
      >
        <X class="h-4 w-4" />
      </button>
    </div>
    <div class="max-h-[70vh] overflow-y-auto px-6 py-5">
      <Transition name="fade" mode="out-in">
        <div :key="ui.footerPage ?? 'empty'" class="space-y-5 text-sm leading-relaxed text-muted-foreground">
          <!-- Privacy -->
          <template v-if="ui.footerPage === 'privacy'">
            <section>
              <h3 class="font-semibold text-foreground">一、資料收集</h3>
              <p class="mt-1.5">我們僅收集與訂單履行、客服聯繫相關之必要資料，包括姓名、電話、電子郵件、收件地址。具體資料處理與保護措施將於正式上線前公告。</p>
            </section>
            <section>
              <h3 class="font-semibold text-foreground">二、資料使用</h3>
              <p class="mt-1.5">您的個人資料用於：訂單處理與配送、客服回覆、電子報寄送（需訂閱）。詳細資料使用政策將於正式上線前公告。</p>
            </section>
            <section>
              <h3 class="font-semibold text-foreground">三、Cookie 使用</h3>
              <p class="mt-1.5">本網站使用 Cookie 來保存購物車內容與偏好設定，以提升您的購物體驗。您可隨時清除 Cookie。</p>
            </section>
            <section>
              <h3 class="font-semibold text-foreground">四、資料安全</h3>
              <p class="mt-1.5">我們採取合理措施保護您的個人資料。具體安全措施將於正式上線前公告。</p>
            </section>
            <section>
              <h3 class="font-semibold text-foreground">五、您的權利</h3>
              <p class="mt-1.5">您有權隨時要求查詢、更正或刪除您的個人資料。客服聯絡方式將於正式上線前公告。</p>
            </section>
          </template>

          <!-- Terms -->
          <template v-else-if="ui.footerPage === 'terms'">
            <section>
              <h3 class="font-semibold text-foreground">一、服務內容</h3>
              <p class="mt-1.5">質物選物提供線上購物服務。我們保留修改商品資訊、價格及服務內容之權利。</p>
            </section>
            <section>
              <h3 class="font-semibold text-foreground">二、訂單成立</h3>
              <p class="mt-1.5">訂單於系統確認後成立。缺貨或異常訂單的處理方式將於正式上線前公告。</p>
            </section>
            <section>
              <h3 class="font-semibold text-foreground">三、退換貨政策</h3>
              <p class="mt-1.5">退換貨政策將於正式上線前依消費者保護法規公告。</p>
            </section>
            <section>
              <h3 class="font-semibold text-foreground">四、保固服務</h3>
              <p class="mt-1.5">保固服務將於正式上線前公告。</p>
            </section>
            <section>
              <h3 class="font-semibold text-foreground">五、免責聲明</h3>
              <p class="mt-1.5">本網站商品圖片僅供參考，實際顏色可能因螢幕設定略有差異。我們盡力確保資訊正確，但不保證完全無誤。</p>
            </section>
          </template>

          <!-- Shipping -->
          <template v-else-if="ui.footerPage === 'shipping'">
            <section>
              <h3 class="font-semibold text-foreground">配送方式</h3>
              <p class="mt-1.5">配送方式、運費及預計送達時間將於正式上線前公告。實際運費與送達時間以結帳時系統顯示為準。</p>
            </section>
            <Separator />
            <section>
              <h3 class="font-semibold text-foreground">出貨時間</h3>
              <p class="mt-1.5">出貨處理時間將於正式上線前公告。</p>
            </section>
            <section>
              <h3 class="font-semibold text-foreground">物流追蹤</h3>
              <p class="mt-1.5">訂單出貨後，您可至「訂單查詢」查看進度。</p>
            </section>
          </template>

          <!-- Contact -->
          <template v-else-if="ui.footerPage === 'contact'">
            <section>
              <h3 class="font-semibold text-foreground">客服資訊</h3>
              <p class="mt-1.5">客服信箱、專線及營業時間將於正式上線前公告。</p>
            </section>
            <Separator />
            <section>
              <h3 class="font-semibold text-foreground">常見聯絡原因</h3>
              <ul class="mt-2 list-inside list-disc space-y-1">
                <li>訂單修改或取消</li>
                <li>退換貨申請</li>
                <li>商品諮詢</li>
                <li>物流追蹤</li>
              </ul>
            </section>
          </template>

          <!-- About -->
          <template v-else-if="ui.footerPage === 'about'">
            <section>
              <h3 class="font-semibold text-foreground">品牌故事</h3>
              <p class="mt-1.5">質物選物（Monolith）的品牌故事與選品理念將於正式上線前公告。</p>
            </section>
            <section>
              <h3 class="font-semibold text-foreground">永續承諾</h3>
              <p class="mt-1.5">永續承諾與包裝政策將於正式上線前公告。</p>
            </section>
          </template>

          <!-- FAQ -->
          <template v-else-if="ui.footerPage === 'faq'">
            <section>
              <h3 class="font-semibold text-foreground">Q：如何追蹤我的訂單？</h3>
              <p class="mt-1.5">至首頁點擊「訂單查詢」，輸入訂單編號與結帳時產生的存取碼即可查看進度。</p>
            </section>
            <Separator />
            <section>
              <h3 class="font-semibold text-foreground">Q：退換貨流程為何？</h3>
              <p class="mt-1.5">退換貨流程將於正式上線前依消費者保護法規公告。</p>
            </section>
            <Separator />
            <section>
              <h3 class="font-semibold text-foreground">Q：優惠碼如何使用？</h3>
              <p class="mt-1.5">在購物車頁面輸入優惠碼並點擊「套用」即可。目前可用的優惠碼請以結帳時系統顯示為準。</p>
            </section>
            <Separator />
            <section>
              <h3 class="font-semibold text-foreground">Q：可以修改訂單嗎？</h3>
              <p class="mt-1.5">訂單修改政策將於正式上線前公告。</p>
            </section>
          </template>
        </div>
      </Transition>
    </div>
  </Dialog>
</template>

<style scoped>
.fade-enter-active, .fade-leave-active { transition: opacity 0.2s ease; }
.fade-enter-from, .fade-leave-to { opacity: 0; }
</style>
