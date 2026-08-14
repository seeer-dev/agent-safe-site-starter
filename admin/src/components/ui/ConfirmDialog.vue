<script setup lang="ts">
import { computed, ref, watch, nextTick } from 'vue'
import Modal from './Modal.vue'
import Button from './Button.vue'
import Textarea from './Textarea.vue'

export interface ConfirmBody {
  /** Bold emphasis value (e.g. row identifier or selected count). */
  emphasis?: string
  /** Text following the emphasis (e.g. "— 刪除"). */
  text: string
}

export interface ConfirmMeta {
  /** expected_version value for optimistic-concurrency actions. */
  expectValue?: string
  /** Static payload object to display as JSON. */
  payload?: Record<string, any>
  /** operationId string. */
  op?: string
  /** Error message from a failed action. */
  error?: string
  /** Generic note text (e.g. bulk action description). */
  note?: string
  /** If true, show a required datetime-local input for expiry. */
  expiryInput?: boolean
}

const props = withDefaults(defineProps<{
  open: boolean
  title: string
  variant?: 'pri' | 'danger'
  body?: ConfirmBody | null
  meta?: ConfirmMeta | null
  requireReason?: boolean
  confirmLabel?: string
  reason?: string
  expiry?: string
}>(), {
  variant: 'pri',
  body: null,
  meta: null,
  requireReason: false,
  confirmLabel: '確認',
  reason: '',
  expiry: '',
})

const emit = defineEmits<{
  (e: 'confirm'): void
  (e: 'cancel'): void
  (e: 'update:reason', value: string): void
  (e: 'update:expiry', value: string): void
}>()

// ----- Expiry validation -----
// The expiry must be a valid datetime-local value AND in the future.
// We compute this reactively so canConfirm gates the submit button and
// an inline message guides the operator. The server still does the
// final check (defense in depth), but we prevent obvious mistakes here.

const expiryInputRef = ref<HTMLInputElement | null>(null)

const expiryValidation = computed<{ valid: boolean; msg: string }>(() => {
  if (!props.meta?.expiryInput) return { valid: true, msg: '' }
  const val = props.expiry
  if (!val) return { valid: false, msg: '請選擇核可到期時間。' }
  const ms = Date.parse(val)
  if (isNaN(ms)) return { valid: false, msg: '日期格式無效，請重新選擇。' }
  const now = Date.now()
  if (ms <= now) return { valid: false, msg: '到期時間必須在現在之後。' }
  return { valid: true, msg: '' }
})

const canConfirm = computed(() => {
  if (props.requireReason && props.reason.trim().length === 0) return false
  if (!expiryValidation.value.valid) return false
  return true
})

const payloadJson = computed(() => props.meta?.payload ? JSON.stringify(props.meta.payload) : '')

// Local timezone hint for the datetime-local input.
const tzHint = computed(() => {
  try {
    const tz = Intl.DateTimeFormat().resolvedOptions().timeZone
    return tz || '本地時區'
  } catch {
    return '本地時區'
  }
})

function onConfirm() {
  if (!canConfirm.value) return
  emit('confirm')
}

function onCancel() {
  emit('cancel')
}

function onReasonUpdate(value: string) {
  emit('update:reason', value)
}

function onExpiryUpdate(value: string) {
  // Preserve the user's value — do not clear on invalid input.
  emit('update:expiry', value)
}

// When the dialog opens with an expiry input, focus it so the operator
// can immediately type/select. When a validation error is shown, keep
// focus on the input so the operator can correct it without losing
// their place.
watch(() => props.open, async (isOpen) => {
  if (isOpen && props.meta?.expiryInput) {
    await nextTick()
    expiryInputRef.value?.focus()
  }
})
</script>

<template>
  <Modal
    :open="open"
    :title="title"
    max-width="min(480px, 94vw)"
    @close="onCancel"
  >
    <!-- Body: structured, no v-html. All values are text-interpolated. -->
    <div v-if="body" class="note">
      <b v-if="body.emphasis">{{ body.emphasis }}</b> {{ body.text }}
    </div>
    <!-- Meta: structured, no v-html. All values are text-interpolated. -->
    <div v-if="meta" class="note">
      <template v-if="meta.note">{{ meta.note }}</template>
      <template v-if="meta.expectValue">
        送出時會一併帶 expected_version = <span class="mono">{{ meta.expectValue }}</span>；若這筆資料已被別人改過，伺服器會回衝突而不是覆寫。
      </template>
      <div v-if="payloadJson" style="margin-top:8px" class="muted">
        payload：<span class="mono">{{ payloadJson }}</span>
      </div>
      <div v-if="meta.op" style="margin-top:6px">
        operationId：<span class="mono">{{ meta.op }}</span>
      </div>
      <div v-if="meta.error" style="margin-top:8px" class="st danger">
        失敗：{{ meta.error }}
      </div>
    </div>
    <div v-if="requireReason" class="field">
      <label>原因</label>
      <Textarea
        :model-value="reason"
        placeholder="請輸入原因..."
        @update:model-value="onReasonUpdate"
      />
    </div>
    <div v-if="meta?.expiryInput" class="field">
      <label for="confirm-expiry-input">
        核可到期時間 <span style="color:var(--danger)">*</span>
      </label>
      <input
        id="confirm-expiry-input"
        ref="expiryInputRef"
        type="datetime-local"
        class="expiry-input"
        :class="{ 'expiry-input-invalid': !expiryValidation.valid && expiry }"
        :value="expiry"
        @input="onExpiryUpdate(($event.target as HTMLInputElement).value)"
      />
      <div class="muted" style="margin-top:4px;font-size:0.85em">
        時區：{{ tzHint }}。操作者自行決定核可有效期，過期後需重新核可才能發布。
      </div>
      <div
        v-if="!expiryValidation.valid && expiry"
        class="st danger"
        style="margin-top:4px;font-size:0.85em"
        role="alert"
        aria-live="polite"
      >
        {{ expiryValidation.msg }}
      </div>
    </div>
    <template #footer>
      <Button variant="ghost" @click="onCancel">取消</Button>
      <Button
        :variant="variant === 'danger' ? 'danger' : 'pri'"
        :disabled="!canConfirm"
        @click="onConfirm"
      >
        {{ confirmLabel }}
      </Button>
    </template>
  </Modal>
</template>

<style scoped>
.expiry-input {
  width: 100%;
  padding: 8px 10px;
  border: 1px solid var(--border, #ccc);
  border-radius: 6px;
  font-size: 0.9rem;
  background: var(--surface, #fff);
  color: var(--text, #222);
}
.expiry-input:focus {
  outline: none;
  border-color: var(--primary, #2563eb);
  box-shadow: 0 0 0 2px rgba(37, 99, 235, 0.15);
}
.expiry-input-invalid {
  border-color: var(--danger, #dc2626);
}
.expiry-input-invalid:focus {
  border-color: var(--danger, #dc2626);
  box-shadow: 0 0 0 2px rgba(220, 38, 38, 0.15);
}
</style>
