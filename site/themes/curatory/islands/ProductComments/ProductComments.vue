<script setup lang="ts">
// 顧客評價區 — 對應 reference product-comments.tsx：
//   評分摘要卡（avg + 5→1 分佈條）、留言卡（頭像字母/星列/相對時間/店家回覆）、
//   留言表單（暱稱/星評/內容，送出後待審核）。
import { computed, onMounted, ref } from 'vue'
import { Loader2, MessageSquareQuote, Send, Star } from 'lucide-vue-next'
import Skeleton from '@/shared/components/Skeleton.vue'
import { apiGet, apiPost } from '@/shared/lib/api'
import { relativeTime } from '@/shared/lib/format'
import { toast } from '@/shared/lib/toast'
import { useTurnstile } from '@/shared/lib/turnstile'
import { cn } from '@/shared/lib/utils'
import type { CommentDTO } from '@/shared/lib/types'

const props = defineProps<{ slug: string }>()
const turnstile = useTurnstile('comment')

interface Summary {
  count: number
  avg: number
  dist: Record<string, number>
}

const comments = ref<CommentDTO[]>([])
const summary = ref<Summary | null>(null)
const loading = ref(true)

const nickname = ref('')
const content = ref('')
const rating = ref<number | null>(null)
const hoverStar = ref<number | null>(null)
const submitting = ref(false)

const starActive = computed(() => hoverStar.value ?? rating.value ?? 0)
const distMax = computed(() => Math.max(1, ...Object.values(summary.value?.dist ?? {})))

async function load() {
  loading.value = true
  try {
    const res = await apiGet<{ comments: CommentDTO[]; summary?: Summary }>(
      `/api/products/${props.slug}/comments`,
    )
    comments.value = res.comments ?? []
    summary.value = res.summary ?? null
  } catch {
    comments.value = []
    summary.value = null
  } finally {
    loading.value = false
  }
}

async function submit() {
  if (nickname.value.trim().length < 2) {
    toast.error('暱稱至少 2 個字')
    return
  }
  if (content.value.trim().length < 5) {
    toast.error('內容至少 5 個字')
    return
  }
  const turnstileToken = turnstile.tokenForSubmit()
  if (turnstileToken === null) {
    toast.error('請先完成驗證')
    return
  }
  submitting.value = true
  try {
    await apiPost(`/api/products/${props.slug}/comments`, {
      nickname: nickname.value.trim(),
      content: content.value.trim(),
      rating: rating.value ?? undefined,
      turnstile_token: turnstileToken || undefined,
    })
    toast.success('已送出，待店家審核後顯示', {
      description: '感謝分享你的想法，留言將於審核後出現在此頁面',
    })
    nickname.value = ''
    content.value = ''
    rating.value = null
    void load()
  } catch (e) {
    toast.error(e instanceof Error ? e.message : '留言送出失敗，請稍後再試')
  } finally {
    // A submitted token is single-use whether it passed or failed; mint a
    // fresh one for the next attempt.
    if (turnstileToken) turnstile.reset()
    submitting.value = false
  }
}

onMounted(load)
</script>

<template>
  <section class="mt-14" data-reveal>
    <p class="eyebrow">Reviews</p>
    <h2 class="mt-3 font-serif text-2xl font-semibold tracking-wide sm:text-3xl">顧客評價</h2>
    <p class="mt-2 max-w-lg text-sm leading-relaxed text-muted-foreground">實際購買顧客分享的使用心得與評分</p>

    <!-- 骨架 -->
    <div v-if="loading" class="mt-6 space-y-5">
      <Skeleton class="h-36 w-full rounded-2xl" />
      <div v-for="i in 3" :key="i" class="rounded-2xl border p-5 sm:p-6">
        <div class="flex items-center gap-3.5">
          <Skeleton class="size-10 rounded-full" />
          <div class="space-y-1.5">
            <Skeleton class="h-4 w-24" />
            <Skeleton class="h-3 w-16" />
          </div>
        </div>
        <Skeleton class="mt-4 h-4 w-full" />
        <Skeleton class="mt-2 h-4 w-4/5" />
      </div>
    </div>

    <div v-else class="mt-6 space-y-5">
      <!-- 評分摘要 -->
      <div v-if="summary && summary.count > 0" class="rounded-2xl border bg-card p-6 sm:p-8">
        <div class="grid items-center gap-8 sm:grid-cols-[auto_1fr]">
          <div class="flex items-center gap-5 sm:border-r sm:pr-8">
            <p class="font-serif text-4xl font-semibold leading-none tabular-nums">
              {{ summary.avg.toFixed(1) }}
            </p>
            <div>
              <span class="inline-flex origin-left scale-110 items-center gap-0.5" :aria-label="`評分 ${Math.round(summary.avg)} 星`">
                <Star
                  v-for="i in 5"
                  :key="i"
                  :class="cn('size-3.5', i <= Math.round(summary.avg) ? 'fill-gold text-gold' : 'fill-transparent text-muted-foreground/35')"
                  :stroke-width="1.5"
                />
              </span>
              <p class="mt-2 text-xs text-muted-foreground">共 {{ summary.count }} 則評價</p>
            </div>
          </div>
          <div class="space-y-2">
            <div v-for="star in [5, 4, 3, 2, 1]" :key="star" class="flex items-center gap-3 text-xs text-muted-foreground">
              <span class="w-3 tabular-nums">{{ star }}</span>
              <Star class="size-3 fill-gold text-gold" :stroke-width="1.5" />
              <div class="h-1.5 flex-1 overflow-hidden rounded-full bg-muted">
                <div class="h-full rounded-full bg-gold/70 transition-all duration-700" :style="{ width: `${((summary.dist[String(star)] ?? 0) / distMax) * 100}%` }" />
              </div>
              <span class="w-6 text-right tabular-nums">{{ summary.dist[String(star)] ?? 0 }}</span>
            </div>
          </div>
        </div>
      </div>

      <!-- 空態 -->
      <div v-if="comments.length === 0" class="rounded-2xl border border-dashed bg-muted/20 px-6 py-14 text-center">
        <div class="mx-auto flex size-14 items-center justify-center rounded-full border bg-background">
          <MessageSquareQuote class="size-6 text-muted-foreground/60" :stroke-width="1.5" />
        </div>
        <p class="mt-4 font-serif text-lg font-medium">還沒有評價</p>
        <p class="mt-1 text-sm text-muted-foreground">歡迎分享你的想法</p>
      </div>

      <!-- 留言列表 -->
      <div v-else class="space-y-4">
        <article
          v-for="(c, i) in comments"
          :key="c.id"
          v-reveal="{ delay: Math.min(i * 0.06, 0.3) }"
          class="rounded-2xl border bg-card p-5 sm:p-6"
        >
          <div class="flex items-center gap-3.5">
            <span
              aria-hidden="true"
              class="flex size-10 shrink-0 items-center justify-center rounded-full bg-primary/10 font-serif text-base font-semibold text-primary"
            >{{ c.nickname.trim().charAt(0).toUpperCase() }}</span>
            <div class="min-w-0 flex-1">
              <div class="flex flex-wrap items-center gap-x-2.5 gap-y-1">
                <p class="font-medium">{{ c.nickname }}</p>
                <span v-if="c.rating" class="inline-flex items-center gap-0.5" :aria-label="`評分 ${c.rating} 星`">
                  <Star
                    v-for="i in 5"
                    :key="i"
                    :class="cn('size-3.5', i <= (c.rating ?? 0) ? 'fill-gold text-gold' : 'fill-transparent text-muted-foreground/35')"
                    :stroke-width="1.5"
                  />
                </span>
              </div>
              <p class="mt-0.5 text-xs text-muted-foreground">{{ relativeTime(c.created_unix) }}</p>
            </div>
          </div>

          <p class="mt-4 whitespace-pre-wrap text-[15px] leading-relaxed text-foreground/90">{{ c.content }}</p>

          <div v-if="c.admin_reply" class="mt-4 rounded-r-xl border-l-2 border-primary/40 bg-muted/40 px-4 py-3.5">
            <p class="text-xs font-medium tracking-wide text-primary">店家回覆</p>
            <p class="mt-1.5 whitespace-pre-wrap text-sm leading-relaxed text-muted-foreground">{{ c.admin_reply }}</p>
            <p v-if="c.replied_unix" class="mt-2 text-[11px] text-muted-foreground/70">{{ relativeTime(c.replied_unix) }}</p>
          </div>
        </article>
      </div>

      <!-- 留言表單 -->
      <div class="rounded-2xl border bg-card p-5 sm:p-6">
        <p class="flex items-center gap-2 font-medium">
          <MessageSquareQuote class="size-4 text-primary" aria-hidden="true" />
          分享你的想法
        </p>
        <p class="mt-1 text-xs text-muted-foreground">留言送出後將由店家審核，通過後才會公開顯示（評分為選填）</p>

        <div class="mt-5 space-y-4">
          <div class="grid gap-4 sm:grid-cols-2">
            <div class="space-y-1.5">
              <label :for="`c-nickname-${slug}`" class="text-sm font-medium">暱稱</label>
              <input
                :id="`c-nickname-${slug}`"
                v-model="nickname"
                type="text"
                placeholder="匿稱"
                maxlength="16"
                :disabled="submitting"
                class="h-10 w-full rounded-md border bg-background px-3 text-sm outline-none focus:border-primary"
              />
            </div>
            <div class="space-y-1.5">
              <p class="text-sm font-medium">評分</p>
              <div class="flex items-center gap-2" role="radiogroup" aria-label="評分">
                <div class="flex items-center gap-1">
                  <button
                    v-for="star in 5"
                    :key="star"
                    type="button"
                    role="radio"
                    :aria-checked="rating === star"
                    :aria-label="`${star} 星`"
                    class="rounded-sm p-0.5 transition-transform hover:scale-110 focus-visible:outline-2 focus-visible:outline-primary"
                    @mouseenter="hoverStar = star"
                    @mouseleave="hoverStar = null"
                    @click="rating = rating === star ? null : star"
                  >
                    <Star
                      :class="cn('size-5 transition-colors', star <= starActive ? 'fill-gold text-gold' : 'fill-transparent text-muted-foreground/40')"
                      :stroke-width="1.5"
                    />
                  </button>
                </div>
                <span class="text-xs text-muted-foreground">{{ rating ? `${rating} 星` : '選填' }}</span>
              </div>
            </div>
          </div>

          <div class="space-y-1.5">
            <label :for="`c-content-${slug}`" class="text-sm font-medium">使用心得</label>
            <textarea
              :id="`c-content-${slug}`"
              v-model="content"
              placeholder="聊聊這件商品的使用心得…"
              rows="3"
              maxlength="500"
              :disabled="submitting"
              class="min-h-24 w-full rounded-md border bg-background px-3 py-2 text-sm outline-none focus:border-primary"
            />
            <p class="text-right text-[11px] text-muted-foreground/70">{{ content.length }} / 500</p>
          </div>

          <div :ref="turnstile.setEl" />
          <button
            type="button"
            :disabled="submitting"
            class="inline-flex h-10 items-center gap-2 rounded-full bg-primary px-7 text-sm font-medium text-primary-foreground transition-colors hover:bg-primary/90 disabled:opacity-50"
            @click="submit"
          >
            <Loader2 v-if="submitting" class="size-4 animate-spin" aria-hidden="true" />
            <Send v-else class="size-4" aria-hidden="true" />
            {{ submitting ? '送出中…' : '送出留言' }}
          </button>
        </div>
      </div>
    </div>
  </section>
</template>
