<script setup lang="ts">
import { onMounted, ref } from 'vue'
import { Mail, ArrowRight } from 'lucide-vue-next'
import { useToast } from '@/shared/composables/use-toast'
import type { FooterPageKey } from '@/shared/lib/types'
import { useUiStore } from '@/shared/stores/ui'

const ui = useUiStore()
const { toast } = useToast()
const email = ref('')

// The Go renderer emits #footer-static as the no-JavaScript fallback. Hide it
// only after this interactive Footer island has mounted, so a failed island
// load leaves the static policy links available.
onMounted(() => {
  document.getElementById('footer-static')?.setAttribute('hidden', '')
})

function subscribe() {
  if (!email.value.trim()) return
  toast({ title: '訂閱成功', description: `確認信已寄至 ${email.value}` })
  email.value = ''
}

const NAV_LINKS: { label: string; key: FooterPageKey }[] = [
  { label: '關於我們', key: 'about' },
  { label: '配送資訊', key: 'shipping' },
  { label: '隱私權政策', key: 'privacy' },
  { label: '服務條款', key: 'terms' },
  { label: '聯絡我們', key: 'contact' },
  { label: '常見問答', key: 'faq' },
]
</script>

<template>
  <footer class="border-t border-border/60 bg-muted/20">
    <div class="mx-auto max-w-7xl px-4 py-12 sm:px-6 lg:px-8">
      <div class="grid grid-cols-1 gap-8 md:grid-cols-3">
        <!-- Brand + newsletter -->
        <div class="space-y-4">
          <div class="flex items-center gap-2">
            <span class="grid h-8 w-8 place-items-center rounded-full bg-cta text-sm font-bold text-cta-foreground">質</span>
            <span class="text-sm font-semibold tracking-tight">質物選物</span>
          </div>
          <p class="max-w-xs text-xs leading-relaxed text-muted-foreground">
            少一點，更好一點。
          </p>
          <div class="flex items-stretch gap-2">
            <div class="relative flex-1">
              <Mail class="pointer-events-none absolute left-3.5 top-1/2 h-4 w-4 -translate-y-1/2 text-muted-foreground" />
              <input
                v-model="email"
                type="email"
                placeholder="輸入 Email 訂閱電子報"
                class="h-10 w-full rounded-full border border-border bg-background pl-10 pr-4 text-sm outline-none transition-colors focus:border-cta"
                @keydown.enter="subscribe"
              />
            </div>
            <button
              @click="subscribe"
              class="inline-flex h-10 shrink-0 items-center gap-1 rounded-full bg-cta px-4 text-sm font-medium text-cta-foreground transition-colors hover:brightness-95"
              aria-label="訂閱"
            >
              訂閱
              <ArrowRight class="h-4 w-4" />
            </button>
          </div>
        </div>

        <!-- Nav links -->
        <div class="space-y-3">
          <h4 class="text-xs font-medium uppercase tracking-wider text-muted-foreground">資訊</h4>
          <ul class="space-y-2">
            <li v-for="link in NAV_LINKS" :key="link.key">
              <button
                @click="ui.openFooterPage(link.key)"
                class="text-sm text-foreground/80 transition-colors hover:text-foreground"
              >
                {{ link.label }}
              </button>
            </li>
          </ul>
        </div>

        <!-- Contact -->
        <div class="space-y-3">
          <h4 class="text-xs font-medium uppercase tracking-wider text-muted-foreground">聯絡</h4>
          <ul class="space-y-2 text-sm text-foreground/80">
            <li>客服資訊將於正式上線前公告。</li>
          </ul>
        </div>
      </div>

      <div class="mt-10 border-t border-border/60 pt-6 text-center text-xs text-muted-foreground">
        © 2026 質物選物 Monolith. All rights reserved.
      </div>
    </div>
  </footer>
</template>
