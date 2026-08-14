<script setup lang="ts">
import { ref, computed, watch } from 'vue'
import {
  User as UserIcon, LogIn, LogOut, Package, Mail, Lock, UserPlus,
  ChevronRight, Calendar, ShoppingBag, X,
} from 'lucide-vue-next'
import { useUserStore, STATUS_META } from '@/shared/stores/user'
import { useUiStore } from '@/shared/stores/ui'
import Dialog from '@/shared/components/ui/Dialog.vue'
import Button from '@/shared/components/ui/Button.vue'
import Input from '@/shared/components/ui/Input.vue'
import Separator from '@/shared/components/ui/Separator.vue'
import Pagination from '@/shared/components/site/Pagination.vue'
import { useToast } from '@/shared/composables/use-toast'
import { formatNTD, cn } from '@/shared/lib/utils'
import { listMyOrders, ApiRequestError } from '@/shared/lib/api'
import { parseMemberOrderListEnvelope, type MemberOrder } from '@/shared/lib/member-orders'
import {
  signInWithPassword,
  signInWithOAuth,
  signUp,
  signOut,
  PublicAuthError,
  publicAuthErrorMessage,
} from '@/shared/lib/auth/session'
import { isPublicAuthEnabled, isGoogleOAuthEnabled, isLineOAuthEnabled, type OAuthProvider } from '@/shared/lib/auth/config'

const ui = useUiStore()
const userStore = useUserStore()
const { toast } = useToast()

type Mode = 'login' | 'register'
const mode = ref<Mode>('login')
const email = ref('')
const password = ref('')
const name = ref('')
const error = ref<string | null>(null)
const submitting = ref(false)
const checkEmail = ref(false)
const authEnabled = isPublicAuthEnabled()
const googleOAuthEnabled = isGoogleOAuthEnabled()
const lineOAuthEnabled = isLineOAuthEnabled()
// Orders fetched from the API for the authenticated member.
const apiOrders = ref<MemberOrder[]>([])
const ordersLoading = ref(false)
const ordersError = ref<string | null>(null)

const user = computed(() => userStore.user)
const userOrders = computed<MemberOrder[]>(() => {
  if (!user.value) return []
  return apiOrders.value
})

const ORDERS_PER_PAGE = 4
const ordersPage = ref(1)
const totalPages = computed(() => Math.max(1, Math.ceil(userOrders.value.length / ORDERS_PER_PAGE)))
const safeOrdersPage = computed(() => Math.min(Math.max(1, ordersPage.value), totalPages.value))
const pagedOrders = computed(() => {
  const start = (safeOrdersPage.value - 1) * ORDERS_PER_PAGE
  return userOrders.value.slice(start, start + ORDERS_PER_PAGE)
})

watch(() => ui.accountOpen, (open) => {
  if (!open) {
    ordersPage.value = 1
    error.value = null
    ordersError.value = null
    checkEmail.value = false
    submitting.value = false
  } else if (user.value && userStore.bearerToken) {
    // Fetch the member's orders from the API when the dialog opens.
    // PII is masked server-side. No local-order fallback.
    fetchMyOrders()
  }
})

async function fetchMyOrders() {
  if (!userStore.bearerToken) return
  ordersLoading.value = true
  ordersError.value = null
  try {
    apiOrders.value = parseMemberOrderListEnvelope(await listMyOrders(userStore.bearerToken))
  } catch (e: unknown) {
    if (e instanceof ApiRequestError && e.status === 401) {
      userStore.logout()
      await signOut()
      ordersError.value = '登入已過期，請重新登入。'
    } else {
      ordersError.value = '無法載入訂單，請稍後再試。'
    }
    apiOrders.value = []
  } finally {
    ordersLoading.value = false
  }
}

function formatDate(ts: number) {
  return new Date(ts).toLocaleDateString('zh-TW', { year: 'numeric', month: 'short', day: 'numeric' })
}

async function handleSubmit() {
  if (submitting.value) return
  error.value = null
  checkEmail.value = false
  if (!authEnabled) {
    error.value = '會員登入尚未開放。'
    return
  }
  submitting.value = true
  try {
    if (mode.value === 'login') {
      await signInWithPassword(email.value.trim(), password.value)
      password.value = ''
      await fetchMyOrders()
    } else {
      const result = await signUp(email.value.trim(), password.value, name.value)
      password.value = ''
      if (result.kind === 'check_email') {
        checkEmail.value = true
      } else {
        await fetchMyOrders()
      }
    }
  } catch (e: unknown) {
    if (e instanceof PublicAuthError && e.kind === 'signin') {
      error.value = '登入失敗，請確認資料後再試。'
    } else if (e instanceof PublicAuthError && e.kind === 'signup') {
      error.value = '無法建立帳號，請稍後再試。'
    } else {
      error.value = publicAuthErrorMessage(e)
    }
  } finally {
    submitting.value = false
  }
}

async function handleOAuth(provider: OAuthProvider) {
  if (submitting.value) return
  error.value = null
  if (!authEnabled) {
    error.value = '會員登入尚未開放。'
    return
  }
  submitting.value = true
  try {
    await signInWithOAuth(provider)
  } catch (e: unknown) {
    error.value = publicAuthErrorMessage(e)
    submitting.value = false
  }
}

async function handleLogout() {
  userStore.logout()
  await signOut()
  toast({ title: '已登出', description: '期待再見面。' })
}

function handleTrackFromOrders(orderId: string) {
  ui.closeAccount()
  ui.openTrackOrder(orderId)
}

const dialogClass = computed(() => user.value ? 'max-w-md' : 'max-w-3xl')
</script>

<template>
  <Dialog
    :open="ui.accountOpen"
    :show-close="false"
    :aria-label="user ? '我的帳號' : (mode === 'login' ? '會員登入' : '建立帳號')"
    :class="cn(dialogClass, 'p-0')"
    @update:open="ui.closeAccount()"
  >
    <!-- Signed-in view -->
    <div v-if="user" class="flex flex-1 flex-col min-h-0">
      <!-- Dialog header -->
      <div class="flex items-center justify-between border-b border-border/60 px-6 py-4">
        <h2 class="text-lg font-semibold tracking-tight">我的帳號</h2>
        <button
          type="button"
          class="grid h-8 w-8 place-items-center rounded-full text-muted-foreground transition-colors hover:bg-muted hover:text-foreground"
          aria-label="關閉帳號視窗"
          @click="ui.closeAccount()"
        >
          <X class="h-4 w-4" />
        </button>
      </div>
      <!-- Profile header -->
      <div class="border-b border-border/60 px-6 py-5">
        <div class="flex items-center gap-3">
          <div class="grid h-11 w-11 place-items-center rounded-full bg-cta text-cta-foreground">
            <span class="text-base font-semibold">{{ user.name.charAt(0).toUpperCase() }}</span>
          </div>
          <div class="min-w-0 flex-1">
            <p class="truncate text-sm font-semibold">{{ user.name }}</p>
            <p class="truncate text-xs text-muted-foreground">{{ user.email }}</p>
          </div>
          <Button variant="outline" size="sm" @click="handleLogout" class="h-8 rounded-full text-xs">
            <LogOut class="mr-1 h-3.5 w-3.5" />
            登出
          </Button>
        </div>
        <div v-if="user.joinedAt" class="mt-3 flex items-center gap-1.5 text-[11px] text-muted-foreground">
          <Calendar class="h-3 w-3" />
          加入時間：{{ formatDate(user.joinedAt) }}
        </div>
      </div>

      <!-- Orders -->
      <div class="flex flex-1 flex-col overflow-hidden">
        <div class="flex items-center justify-between px-6 pt-5">
          <h3 class="flex items-center gap-2 text-sm font-semibold">
            <Package class="h-4 w-4" />
            我的訂單
          </h3>
          <span class="rounded-full bg-muted px-2 py-0.5 text-[11px] text-muted-foreground">
            共 {{ userOrders.length }} 筆
          </span>
        </div>

        <!-- Loading state -->
        <div v-if="ordersLoading" class="mt-8 flex flex-col items-center justify-center gap-3 px-6 py-10 text-center">
          <div class="grid h-12 w-12 place-items-center rounded-full bg-muted">
            <Package class="h-5 w-5 text-muted-foreground animate-pulse" />
          </div>
          <p class="text-sm text-muted-foreground">載入訂單中…</p>
        </div>

        <!-- Error state -->
        <div v-else-if="ordersError" class="mt-8 flex flex-col items-center justify-center gap-3 px-6 py-10 text-center">
          <div class="grid h-12 w-12 place-items-center rounded-full bg-destructive/10">
            <X class="h-5 w-5 text-destructive" />
          </div>
          <div>
            <p class="text-sm font-medium text-destructive">{{ ordersError }}</p>
            <Button @click="fetchMyOrders" variant="outline" class="mt-3 h-8 rounded-full text-xs">重試</Button>
          </div>
        </div>

        <!-- Empty state -->
        <div v-else-if="userOrders.length === 0" class="mt-8 flex flex-col items-center justify-center gap-3 px-6 py-10 text-center">
          <div class="grid h-12 w-12 place-items-center rounded-full bg-muted">
            <ShoppingBag class="h-5 w-5 text-muted-foreground" />
          </div>
          <div>
            <p class="text-sm font-medium">尚無訂單</p>
            <p class="mt-1 text-xs text-muted-foreground">下單後訂單會顯示在這裡。</p>
          </div>
        </div>

        <template v-else>
          <div class="flex-1 overflow-y-auto px-6 py-4 min-h-0 [scrollbar-width:thin]">
            <ul class="space-y-2.5">
              <li v-for="order in pagedOrders" :key="order.id">
                <button
                  @click="handleTrackFromOrders(order.id)"
                  class="flex w-full items-center gap-3 rounded-xl border border-border/60 bg-background px-3 py-3 text-left transition-colors hover:border-border hover:bg-muted/30"
                >
                  <div class="h-12 w-10 shrink-0 overflow-hidden rounded-md bg-muted" />
                  <div class="min-w-0 flex-1">
                    <div class="flex items-center justify-between gap-2">
                      <p class="truncate text-xs font-semibold">{{ order.id }}</p>
                      <span class="shrink-0 text-[10px] text-muted-foreground">{{ formatDate(order.updatedUnix * 1000) }}</span>
                    </div>
                    <p class="mt-0.5 truncate text-[11px] text-muted-foreground">
                      {{ order.items.reduce((s, i) => s + i.quantity, 0) }} 件商品 · {{ formatNTD(order.total) }}
                    </p>
                    <div class="mt-1.5 flex items-center gap-1.5">
                      <span class="inline-flex h-4 items-center rounded-full bg-cta/10 px-1.5 text-[10px] font-medium text-cta">
                        {{ STATUS_META[order.status].label }}
                      </span>
                    </div>
                  </div>
                  <ChevronRight class="h-4 w-4 shrink-0 text-muted-foreground" />
                </button>
              </li>
            </ul>
          </div>

          <div class="border-t border-border/60 px-6 py-3">
            <div class="flex items-center justify-between gap-3">
              <span class="text-[11px] text-muted-foreground">
                第 {{ safeOrdersPage }} / {{ totalPages }} 頁 · 每頁 {{ ORDERS_PER_PAGE }} 筆
              </span>
              <Pagination
                :current-page="safeOrdersPage"
                :total-pages="totalPages"
                @page-change="ordersPage = $event"
              />
            </div>
          </div>
        </template>
      </div>
    </div>

    <!-- Signed-out view -->
    <div v-else class="flex flex-1 flex-col overflow-hidden sm:flex-row min-h-0">
      <!-- Left image (desktop) -->
      <div class="relative hidden sm:block sm:w-2/5 shrink-0">
        <div class="relative h-full overflow-hidden bg-gradient-to-br from-stone-800 via-stone-900 to-black">
          <div class="absolute inset-0" style="background: linear-gradient(160deg, rgba(28,25,23,0.55) 0%, rgba(28,25,23,0.85) 100%)" />
          <div class="absolute inset-0 flex flex-col justify-end p-7 text-background">
            <div class="mb-4 inline-flex w-fit items-center gap-2 rounded-full bg-background/15 px-3 py-1 text-xs font-medium backdrop-blur">
              <span class="h-1.5 w-1.5 rounded-full bg-cta" />
              質物選物
            </div>
            <h2 class="text-2xl font-semibold leading-tight tracking-tight drop-shadow-sm">
              {{ mode === 'login' ? '歡迎回來' : '加入會員' }}
            </h2>
            <p class="mt-2 max-w-[260px] text-sm leading-relaxed text-background/85">
              {{ mode === 'login' ? '登入後可查詢訂單紀錄。' : '加入會員可查詢訂單紀錄。' }}
            </p>
            <div class="mt-5 flex items-center gap-2 text-[11px] text-background/70">
              <span>宅配到府</span><span class="opacity-50">·</span>
              <span>超商取貨</span>
            </div>
          </div>
        </div>
      </div>

      <!-- Right form -->
      <div class="flex flex-1 flex-col overflow-y-auto min-h-0">
        <div class="border-b border-border/60 px-6 py-5">
          <div class="mb-3 flex items-center justify-between gap-3 sm:hidden">
            <div class="flex items-center gap-3">
              <div class="grid h-10 w-10 shrink-0 place-items-center rounded-full bg-cta text-cta-foreground">
                <span class="text-sm font-semibold">質</span>
              </div>
              <div>
                <p class="text-sm font-semibold">質物選物</p>
                <p class="text-[11px] text-muted-foreground">{{ mode === 'login' ? '歡迎回來' : '建立帳號' }}</p>
              </div>
            </div>
            <button
              type="button"
              class="grid h-8 w-8 place-items-center rounded-full text-muted-foreground transition-colors hover:bg-muted hover:text-foreground"
              aria-label="關閉帳號視窗"
              @click="ui.closeAccount()"
            >
              <X class="h-4 w-4" />
            </button>
          </div>
          <div class="hidden items-center justify-between sm:flex">
            <div>
              <h2 class="text-base font-semibold tracking-tight">
                {{ mode === 'login' ? '會員登入' : '建立帳號' }}
              </h2>
              <p class="mt-1 text-xs text-muted-foreground">
                {{ mode === 'login' ? '登入後可查詢訂單紀錄。' : '加入會員可查詢訂單紀錄。' }}
              </p>
            </div>
            <button
              type="button"
              class="grid h-8 w-8 shrink-0 place-items-center rounded-full text-muted-foreground transition-colors hover:bg-muted hover:text-foreground"
              aria-label="關閉帳號視窗"
              @click="ui.closeAccount()"
            >
              <X class="h-4 w-4" />
            </button>
          </div>
        </div>

        <div v-if="!authEnabled" class="flex-1 space-y-3 px-6 py-10 text-center">
          <div class="mx-auto grid h-12 w-12 place-items-center rounded-full bg-muted">
            <Lock class="h-5 w-5 text-muted-foreground" />
          </div>
          <p class="text-sm font-medium">會員登入尚未開放</p>
          <p class="text-xs text-muted-foreground">此建置未啟用會員登入，仍可使用訪客結帳。</p>
        </div>

        <div v-else-if="checkEmail" class="flex-1 space-y-3 px-6 py-10 text-center" role="status">
          <div class="mx-auto grid h-12 w-12 place-items-center rounded-full bg-muted">
            <Mail class="h-5 w-5 text-muted-foreground" />
          </div>
          <p class="text-sm font-medium">請至信箱收取確認信件</p>
          <p class="text-xs text-muted-foreground">帳號已建立。完成信箱確認後即可登入。</p>
          <Button
            type="button"
            variant="outline"
            class="h-9 rounded-full text-xs"
            @click="mode = 'login'; checkEmail = false; error = null"
          >
            返回登入
          </Button>
        </div>

        <form v-else @submit.prevent="handleSubmit" class="flex-1 space-y-4 px-6 py-5" :aria-busy="submitting">
          <div v-if="mode === 'register'" class="space-y-1.5">
            <label for="account-name" class="text-xs font-medium">姓名</label>
            <div class="relative">
              <UserIcon class="pointer-events-none absolute left-3 top-1/2 h-3.5 w-3.5 -translate-y-1/2 text-muted-foreground" />
              <Input
                id="account-name"
                v-model="name"
                name="name"
                autocomplete="name"
                placeholder="林美玲"
                class="h-9 pl-9"
                :disabled="submitting"
              />
            </div>
          </div>

          <div class="space-y-1.5">
            <label for="account-email" class="text-xs font-medium">電子郵件</label>
            <div class="relative">
              <Mail class="pointer-events-none absolute left-3 top-1/2 h-3.5 w-3.5 -translate-y-1/2 text-muted-foreground" />
              <Input
                id="account-email"
                v-model="email"
                type="email"
                name="email"
                autocomplete="username"
                placeholder="you@example.com"
                class="h-9 pl-9"
                required
                :disabled="submitting"
              />
            </div>
          </div>

          <div class="space-y-1.5">
            <label for="account-password" class="text-xs font-medium">密碼</label>
            <div class="relative">
              <Lock class="pointer-events-none absolute left-3 top-1/2 h-3.5 w-3.5 -translate-y-1/2 text-muted-foreground" />
              <Input
                id="account-password"
                v-model="password"
                type="password"
                name="password"
                :autocomplete="mode === 'login' ? 'current-password' : 'new-password'"
                placeholder="••••••••"
                class="h-9 pl-9"
                required
                :disabled="submitting"
              />
            </div>
          </div>

          <p v-if="error" id="account-auth-error" class="text-xs text-destructive" role="alert">{{ error }}</p>

          <Button
            type="submit"
            class="h-10 w-full rounded-full bg-cta text-cta-foreground hover:brightness-95"
            :disabled="submitting"
          >
            <template v-if="mode === 'login'">
              <LogIn class="mr-1.5 h-4 w-4" /> {{ submitting ? '登入中…' : '登入' }}
            </template>
            <template v-else>
              <UserPlus class="mr-1.5 h-4 w-4" /> {{ submitting ? '建立中…' : '建立帳號' }}
            </template>
          </Button>

          <div v-if="googleOAuthEnabled || lineOAuthEnabled" class="space-y-2 pt-1">
            <p class="text-center text-[11px] text-muted-foreground">或使用社群帳號</p>
            <Button
              v-if="googleOAuthEnabled"
              type="button"
              variant="outline"
              class="h-9 w-full rounded-full text-xs"
              :disabled="submitting"
              @click="handleOAuth('google')"
            >
              使用 Google 登入
            </Button>
            <Button
              v-if="lineOAuthEnabled"
              type="button"
              variant="outline"
              class="h-9 w-full rounded-full text-xs"
              :disabled="submitting"
              @click="handleOAuth('custom:line')"
            >
              使用 LINE 登入
            </Button>
          </div>
        </form>

        <div v-if="authEnabled && !checkEmail" class="border-t border-border/60 px-6 py-4 text-center text-xs text-muted-foreground">
          <template v-if="mode === 'login'">
            還沒有帳號？
            <button type="button" @click="mode = 'register'; error = null; checkEmail = false" class="font-medium text-foreground underline-offset-2 hover:underline">立即註冊</button>
          </template>
          <template v-else>
            已有帳號？
            <button type="button" @click="mode = 'login'; error = null; checkEmail = false" class="font-medium text-foreground underline-offset-2 hover:underline">登入</button>
          </template>
        </div>
      </div>
    </div>
  </Dialog>
</template>
