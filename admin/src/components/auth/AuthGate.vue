<script setup lang="ts">
import { computed, ref, onMounted } from 'vue'
import { KeyRound, Loader2, Lock, LogOut, Mail, Moon, Sun } from 'lucide-vue-next'
import { useAuthStore } from '@/stores/auth'
import { useThemeStore } from '@/stores/theme'
import Button from '@/components/ui/Button.vue'
import Input from '@/components/ui/Input.vue'
import Checkbox from '@/components/ui/Checkbox.vue'
import type { OAuthProvider } from '@/lib/auth/config'

const REMEMBER_EMAIL_KEY = 'admin.rememberEmail'
const LAST_OAUTH_KEY = 'admin.lastOAuth'

const auth = useAuthStore()
const theme = useThemeStore()

const email = ref('')
const password = ref('')
const devToken = ref('')
const rememberMe = ref(false)
const lastOAuth = ref<OAuthProvider | null>(null)

const submitting = computed(() => auth.status === 'connecting')
const showDevForm = computed(() => auth.mode === 'dev' && auth.status === 'unverified')
const showPasswordForm = computed(() => auth.mode === 'supabase' && auth.status === 'unverified')

// OAuth providers sorted so the last-used one comes first.
const oauthProviders = computed(() => {
  const providers: OAuthProvider[] = []
  if (auth.googleOAuthEnabled) providers.push('google')
  if (auth.lineOAuthEnabled) providers.push('custom:line')
  if (lastOAuth.value) {
    const idx = providers.indexOf(lastOAuth.value)
    if (idx > 0) {
      providers.splice(idx, 1)
      providers.unshift(lastOAuth.value)
    }
  }
  return providers
})

const oauthLabel: Record<OAuthProvider, string> = {
  google: '使用 Google 登入',
  'custom:line': '使用 LINE 登入',
}

onMounted(() => {
  try {
    const saved = localStorage.getItem(REMEMBER_EMAIL_KEY)
    if (saved) {
      email.value = saved
      rememberMe.value = true
    }
  } catch (_) { /* noop */ }
  try {
    const last = localStorage.getItem(LAST_OAUTH_KEY)
    if (last === 'google' || last === 'custom:line') {
      lastOAuth.value = last
    }
  } catch (_) { /* noop */ }
})

async function submitPassword() {
  if (submitting.value) return
  try {
    if (rememberMe.value) {
      localStorage.setItem(REMEMBER_EMAIL_KEY, email.value.trim())
    } else {
      localStorage.removeItem(REMEMBER_EMAIL_KEY)
    }
  } catch (_) { /* noop */ }
  await auth.signIn(email.value.trim(), password.value)
}

async function submitDevToken() {
  if (submitting.value) return
  await auth.signInWithDevToken(devToken.value)
  if (auth.status === 'verified' || auth.status === 'forbidden') {
    devToken.value = ''
  }
}

async function submitOAuth(provider: OAuthProvider) {
  if (submitting.value) return
  try {
    localStorage.setItem(LAST_OAUTH_KEY, provider)
  } catch (_) { /* noop */ }
  lastOAuth.value = provider
  await auth.signInWithOAuth(provider)
}
</script>

<template>
  <div class="authgate">
    <!-- Background decoration -->
    <div class="auth-bg" aria-hidden="true">
      <div class="auth-bg-circle auth-bg-circle-1" />
      <div class="auth-bg-circle auth-bg-circle-2" />
    </div>

    <!-- Theme toggle -->
    <button
      class="tbtn auth-theme-toggle"
      type="button"
      :aria-label="theme.isDark ? '切換亮色模式' : '切換暗色模式'"
      @click="theme.toggle()"
    >
      <Moon v-if="!theme.isDark" />
      <Sun v-else />
    </button>

    <!-- Login card -->
    <section class="authcard" aria-live="polite">
      <!-- Left brand panel (hidden on mobile) -->
      <div class="auth-brand-panel">
        <div class="auth-brand-inner">
          <div class="auth-brand-top">
            <div class="auth-brand-logo">
              <svg viewBox="0 0 24 24" width="20" height="20" fill="none" stroke="currentColor" stroke-width="2.2"><path d="M12 22s8-4 8-10V5l-8-3-8 3v7c0 6 8 10 8 10"/></svg>
            </div>
            <span class="auth-brand-name">質物選物</span>
          </div>
          <div class="auth-brand-mid">
            <h2>商家後台管理系統</h2>
            <p>從訂單、商品到會員權限，一站管理你的電商營運。</p>
            <div class="auth-brand-tags">
              <span>訂單管理</span>
              <span>商品上架</span>
              <span>權限控制</span>
            </div>
          </div>
          <div class="auth-brand-bottom">
            <span class="auth-brand-dot" />
            <span>© {{ new Date().getFullYear() }} 質物選物</span>
          </div>
        </div>
      </div>

      <!-- Right form panel -->
      <div class="auth-form-panel">
        <!-- Connecting -->
        <div v-if="auth.status === 'connecting'" class="auth-state">
          <Loader2 class="spin" aria-hidden="true" />
          <b>驗證中…</b>
          <p>正在向伺服器確認身分與權限。</p>
        </div>

        <!-- Unavailable -->
        <div v-else-if="auth.status === 'unavailable'" class="auth-state">
          <div class="auth-state-icon">
            <Lock />
          </div>
          <b>登入功能尚未設定</b>
          <p>此建置未提供可用的身分設定，因此無法登入後台。</p>
        </div>

        <!-- Forbidden -->
        <div v-else-if="auth.status === 'forbidden'" class="auth-state">
          <div class="auth-state-icon auth-state-icon-warn">
            <Lock />
          </div>
          <b>{{ auth.verifyError || '此帳號無管理員權限' }}</b>
          <p>已確認登入身分，但伺服器未授予後台權限。受保護資料不會顯示。</p>
          <Button variant="sec" type="button" @click="auth.logout()">
            <LogOut />
            登出
          </Button>
        </div>

        <!-- Failed -->
        <div v-else-if="auth.status === 'failed'" class="auth-state">
          <div class="auth-state-icon auth-state-icon-danger">
            <Lock />
          </div>
          <b>無法驗證身分</b>
          <p>{{ auth.verifyError || '目前無法驗證身分，請稍後再試。' }}</p>
          <div class="auth-actions">
            <Button variant="pri" type="button" @click="auth.verify()">重試</Button>
            <Button variant="sec" type="button" @click="auth.logout()">
              <LogOut />
              登出
            </Button>
          </div>
        </div>

        <!-- Password form -->
        <form
          v-else-if="showPasswordForm"
          class="authform"
          @submit.prevent="submitPassword"
        >
          <div class="auth-form-header">
            <h1>登入</h1>
            <p>請輸入您的帳號密碼以繼續</p>
            <span class="auth-site-tag">
              <span class="auth-site-dot" />
              tw-minimal-cart
            </span>
          </div>

          <div v-if="auth.formAlert" class="auth-error" role="alert">
            <svg viewBox="0 0 24 24" width="16" height="16" fill="none" stroke="currentColor" stroke-width="2"><circle cx="12" cy="12" r="10"/><path d="M12 8v4M12 16h.01"/></svg>
            <span>{{ auth.formAlert }}</span>
          </div>

          <div class="auth-field">
            <label for="admin-email">電子郵件</label>
            <div class="input-wrap">
              <Mail class="lead-icon" aria-hidden="true" />
              <Input
                id="admin-email"
                v-model="email"
                type="email"
                name="email"
                autocomplete="username"
                placeholder="you@example.com"
                :disabled="submitting"
                required
              />
            </div>
          </div>
          <div class="auth-field">
            <label for="admin-password">密碼</label>
            <div class="input-wrap">
              <Lock class="lead-icon" aria-hidden="true" />
              <Input
                id="admin-password"
                v-model="password"
                type="password"
                name="password"
                autocomplete="current-password"
                placeholder="••••••••"
                :disabled="submitting"
                required
                toggle
              />
            </div>
          </div>
          <div class="auth-remember">
            <Checkbox
              :checked="rememberMe"
              :disabled="submitting"
              @update:checked="rememberMe = $event"
            />
            <label class="auth-remember-label">記住我的帳號</label>
          </div>
          <Button variant="pri" type="submit" :disabled="submitting" class="auth-submit">
            {{ submitting ? '登入中…' : '登入' }}
          </Button>
          <div v-if="oauthProviders.length" class="oauth">
            <p class="oauth-divider">或使用社群帳號</p>
            <Button
              v-for="provider in oauthProviders"
              :key="provider"
              variant="sec"
              type="button"
              :disabled="submitting"
              @click="submitOAuth(provider)"
            >
              {{ oauthLabel[provider] }}
              <span v-if="lastOAuth === provider" class="oauth-last-tag">上次使用</span>
            </Button>
          </div>
        </form>

        <!-- Dev token form -->
        <form
          v-else-if="showDevForm"
          class="authform"
          @submit.prevent="submitDevToken"
        >
          <div class="auth-form-header">
            <h1>開發模式</h1>
            <p>輸入開發用權杖以連線</p>
            <span class="auth-site-tag">
              <span class="auth-site-dot" />
              tw-minimal-cart
            </span>
          </div>

          <p class="auth-note">本機開發模式。權杖只留在記憶體，不會寫入瀏覽器儲存。</p>

          <div v-if="auth.formAlert" class="auth-error" role="alert">
            <svg viewBox="0 0 24 24" width="16" height="16" fill="none" stroke="currentColor" stroke-width="2"><circle cx="12" cy="12" r="10"/><path d="M12 8v4M12 16h.01"/></svg>
            <span>{{ auth.formAlert }}</span>
          </div>

          <div class="auth-field">
            <label for="admin-dev-token">開發用權杖</label>
            <div class="input-wrap">
              <KeyRound class="lead-icon" aria-hidden="true" />
              <Input
                id="admin-dev-token"
                v-model="devToken"
                type="password"
                name="dev-token"
                autocomplete="off"
                placeholder="輸入開發權杖"
                :disabled="submitting"
                required
                toggle
              />
            </div>
          </div>
          <Button variant="pri" type="submit" :disabled="submitting" class="auth-submit">
            {{ submitting ? '連線中…' : '連線' }}
          </Button>
        </form>
      </div>
    </section>
  </div>
</template>

<style scoped>
.authgate {
  min-height: 100vh;
  display: flex;
  align-items: center;
  justify-content: center;
  padding: 24px;
  position: relative;
  overflow: hidden;
  background: var(--surface-2);
}

/* ---- Background decoration ---- */
.auth-bg {
  position: absolute;
  inset: 0;
  pointer-events: none;
}
.auth-bg-circle {
  position: absolute;
  border-radius: 50%;
  filter: blur(80px);
}
.auth-bg-circle-1 {
  top: -80px;
  left: -60px;
  width: 480px;
  height: 480px;
  background: var(--brand-100);
  opacity: 0.35;
}
.auth-bg-circle-2 {
  bottom: -120px;
  right: -80px;
  width: 440px;
  height: 440px;
  background: var(--brand-50);
  opacity: 0.4;
}

/* ---- Theme toggle ---- */
.auth-theme-toggle {
  position: absolute;
  top: 16px;
  right: 16px;
  z-index: 10;
}

/* ---- Login card (split panel) ---- */
.authcard {
  position: relative;
  display: flex;
  width: min(880px, 100%);
  min-height: 500px;
  border-radius: 14px;
  overflow: hidden;
  border: 1px solid var(--border);
  box-shadow: var(--shadow-lg);
}

/* ---- Left brand panel ---- */
.auth-brand-panel {
  display: none;
  width: 48%;
  position: relative;
  overflow: hidden;
  background: linear-gradient(135deg, var(--brand-600), var(--brand), var(--brand-600));
}
.auth-brand-inner {
  position: relative;
  z-index: 1;
  display: flex;
  flex-direction: column;
  justify-content: space-between;
  height: 100%;
  padding: 28px 32px;
  color: #fff;
}
.auth-brand-inner::before {
  content: "";
  position: absolute;
  inset: 0;
  background: linear-gradient(to top, rgba(0,0,0,0.25), transparent);
  pointer-events: none;
}
.auth-brand-top {
  display: flex;
  align-items: center;
  gap: 10px;
}
.auth-brand-logo {
  display: flex;
  align-items: center;
  justify-content: center;
  width: 36px;
  height: 36px;
  border-radius: 8px;
  border: 1px solid rgba(255,255,255,0.3);
  background: rgba(255,255,255,0.15);
  backdrop-filter: blur(4px);
}
.auth-brand-name {
  font-size: 14px;
  font-weight: 600;
  letter-spacing: 0.02em;
}
.auth-brand-mid h2 {
  margin: 0;
  font-size: 22px;
  font-weight: 700;
  line-height: 1.2;
}
.auth-brand-mid p {
  margin: 10px 0 0;
  font-size: 12.5px;
  line-height: 1.6;
  color: rgba(255,255,255,0.8);
}
.auth-brand-tags {
  display: flex;
  flex-wrap: wrap;
  gap: 5px;
  margin-top: 20px;
}
.auth-brand-tags span {
  padding: 4px 10px;
  border-radius: 20px;
  border: 1px solid rgba(255,255,255,0.3);
  background: rgba(255,255,255,0.15);
  font-size: 11px;
  font-weight: 500;
  backdrop-filter: blur(4px);
}
.auth-brand-bottom {
  display: flex;
  align-items: center;
  gap: 8px;
  font-size: 11px;
  color: rgba(255,255,255,0.6);
}
.auth-brand-dot {
  width: 4px;
  height: 4px;
  border-radius: 50%;
  background: rgba(255,255,255,0.6);
}

/* ---- Right form panel ---- */
.auth-form-panel {
  flex: 1;
  display: flex;
  flex-direction: column;
  justify-content: center;
  background: var(--surface);
  padding: 28px 32px;
}

/* ---- Form header ---- */
.auth-form-header {
  margin-bottom: 20px;
}
.auth-form-header h1 {
  margin: 0;
  font-size: 20px;
  font-weight: 700;
  color: var(--text);
}
.auth-form-header p {
  margin: 6px 0 0;
  font-size: 13px;
  color: var(--text-3);
}
.auth-site-tag {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  margin-top: 12px;
  padding: 4px 10px;
  border-radius: 20px;
  background: var(--surface-2);
  border: 1px solid var(--border);
  font-size: 11px;
  font-family: ui-monospace, Consolas, monospace;
  color: var(--text-2);
}
.auth-site-dot {
  width: 6px;
  height: 6px;
  border-radius: 50%;
  background: var(--brand);
}

/* ---- Error alert ---- */
.auth-error {
  display: flex;
  align-items: flex-start;
  gap: 10px;
  margin-bottom: 20px;
  padding: 12px;
  border-radius: 8px;
  background: var(--red-bg);
  border: 1px solid var(--red);
  color: var(--red);
  font-size: 13px;
}
.auth-error svg {
  flex-shrink: 0;
  margin-top: 1px;
}

/* ---- Dev note ---- */
.auth-note {
  margin: 0 0 16px;
  padding: 10px 12px;
  border: 1px solid var(--brand-100);
  background: var(--brand-50);
  border-radius: 8px;
  color: var(--brand-600);
  font-size: 12.5px;
  line-height: 1.6;
}

/* ---- Form fields ---- */
.authform {
  display: flex;
  flex-direction: column;
  gap: 14px;
}
.auth-field label {
  display: block;
  font-size: 13px;
  font-weight: 500;
  margin-bottom: 6px;
  color: var(--text-2);
}
.input-wrap {
  position: relative;
}
.input-wrap :deep(.inp) {
  padding-left: 34px;
  padding-right: 10px;
  width: 100%;
  box-sizing: border-box;
}
/* When toggle is active, the eye button needs space too — the
   .input-toggle-wrap .inp rule in globals.css adds padding-right:34px,
   and here we just ensure the lead icon doesn't overlap. */
.input-wrap :deep(.input-toggle-wrap .inp) {
  padding-left: 34px;
  padding-right: 34px;
}
.lead-icon {
  position: absolute;
  left: 10px;
  top: 50%;
  width: 15px;
  height: 15px;
  transform: translateY(-50%);
  color: var(--text-3);
  pointer-events: none;
  z-index: 1;
}

/* ---- Submit button ---- */
.auth-submit {
  width: 100%;
  justify-content: center;
  padding: 9px 16px;
  font-size: 13.5px;
}

/* ---- Remember me ---- */
.auth-remember {
  display: flex;
  align-items: center;
  gap: 8px;
  margin-top: -4px;
}
.auth-remember-label {
  font-size: 12.5px;
  color: var(--text-2);
  cursor: pointer;
  user-select: none;
}

/* ---- OAuth section ---- */
.oauth {
  display: flex;
  flex-direction: column;
  gap: 8px;
  margin-top: 8px;
}
.oauth-divider {
  text-align: center;
  font-size: 11px;
  font-weight: 600;
  color: var(--text-3);
  text-transform: uppercase;
  letter-spacing: 0.05em;
  margin: 0 0 4px;
}
.oauth-last-tag {
  margin-left: auto;
  font-size: 10px;
  font-weight: 600;
  color: var(--brand-600);
  background: var(--brand-50);
  padding: 2px 7px;
  border-radius: 20px;
}

/* ---- State views (connecting, unavailable, forbidden, failed) ---- */
.auth-state {
  display: flex;
  flex-direction: column;
  align-items: center;
  text-align: center;
  gap: 10px;
  padding: 20px 0;
}
.auth-state b {
  font-size: 16px;
  color: var(--text);
}
.auth-state p {
  margin: 0;
  font-size: 13px;
  color: var(--text-3);
  max-width: 300px;
  line-height: 1.6;
}
.auth-state-icon {
  width: 42px;
  height: 42px;
  border-radius: 11px;
  background: var(--surface-2);
  color: var(--text-3);
  display: grid;
  place-items: center;
}
.auth-state-icon svg {
  width: 21px;
  height: 21px;
}
.auth-state-icon-warn {
  background: var(--amber-bg);
  color: var(--amber);
}
.auth-state-icon-danger {
  background: var(--red-bg);
  color: var(--red);
}
.auth-actions {
  display: flex;
  gap: 8px;
  flex-wrap: wrap;
  justify-content: center;
  margin-top: 4px;
}

/* ---- Spinner ---- */
.spin {
  width: 22px;
  height: 22px;
  animation: spin 1s linear infinite;
}
@keyframes spin {
  to { transform: rotate(360deg); }
}

/* ---- Responsive: show brand panel on ≥640px ---- */
@media (min-width: 640px) {
  .auth-brand-panel {
    display: block;
  }
}
@media (max-width: 639px) {
  .authcard {
    min-height: auto;
  }
  .auth-form-panel {
    padding: 24px 20px;
  }
}
</style>
