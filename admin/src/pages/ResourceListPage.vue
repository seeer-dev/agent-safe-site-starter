<script setup lang="ts">
import { ref, watch, computed, reactive, onMounted, nextTick } from 'vue'
import { useRoute } from 'vue-router'
import ResourceTable from '@/components/resource/ResourceTable.vue'
import Button from '@/components/ui/Button.vue'
import Input from '@/components/ui/Input.vue'
import Select from '@/components/ui/Select.vue'
import Textarea from '@/components/ui/Textarea.vue'
import Modal from '@/components/ui/Modal.vue'
import MediaUploader from '@/components/MediaUploader.vue'
import ConfirmDialog, { type ConfirmBody, type ConfirmMeta } from '@/components/ui/ConfirmDialog.vue'
import { RES } from '@/config/resources'
import { MACHINES } from '@/config/machines'
import { LABEL } from '@/config/tones'
import { useAuthStore } from '@/stores/auth'
import { api, ApiError } from '@/lib/api-client'
import type { RowAction, BulkAction, FieldDef } from '@/lib/types'
import { Plus } from 'lucide-vue-next'

const route = useRoute()
const auth = useAuthStore()

const resourceKey = computed(() => route.path.replace('/res/', ''))
const resource = computed(() => RES[resourceKey.value])

// ----- Data fetching -------------------------------------------------------
const rows = ref<Record<string, any>[]>([])
const loading = ref(false)
const error = ref<string | null>(null)
const mutating = ref(false)

async function loadRows() {
  const r = resource.value
  if (!r.api) {
    // No fixture fallback — require a real API endpoint.
    rows.value = []
    error.value = '此資源未設定 API 端點，無法載入資料。'
    return
  }
  loading.value = true
  error.value = null
  try {
    const data = await api.get<Record<string, any[]>>(r.api.list)
    // API returns { products: [...] } / { orders: [...] } / { items: [...] } etc.
    const arr = Object.values(data).find((v) => Array.isArray(v)) as
      | Record<string, any>[]
      | undefined
    const raw = arr ?? []
    rows.value = r.rowMap ? raw.map((row) => r.rowMap!(row)) : raw
  } catch (e: any) {
    error.value = e.message ?? String(e)
    rows.value = []
  } finally {
    loading.value = false
  }
}

onMounted(loadRows)
watch(resourceKey, () => {
  selected.value = new Set()
  loadRows()
})

// Selection state
const selected = ref<Set<number>>(new Set())

function toggleRow(i: number) {
  const s = new Set(selected.value)
  if (s.has(i)) s.delete(i)
  else s.add(i)
  selected.value = s
}

function toggleAll() {
  if (selected.value.size === rows.value.length) {
    selected.value = new Set()
  } else {
    selected.value = new Set(rows.value.map((_, i) => i))
  }
}

// Form modal
const formOpen = ref(false)
const formRowIndex = ref<number | null>(null)
const formIsNew = computed(() => formRowIndex.value === null)
const formData = reactive<Record<string, any>>({})
const formError = ref<string | null>(null)
const formInitial = ref('{}')
const formDirty = computed(() => JSON.stringify(formData) !== formInitial.value)
const formTriggerEl = ref<HTMLElement | null>(null)

function unixToDatetimeLocal(value: unknown): string {
  const seconds = Number(value)
  if (!Number.isFinite(seconds) || seconds <= 0) return ''
  const date = new Date(seconds * 1000)
  const local = new Date(date.getTime() - date.getTimezoneOffset() * 60_000)
  return local.toISOString().slice(0, 16)
}

function seedFieldValue(fd: FieldDef, value: unknown): unknown {
  if (fd.k === 'images') return Array.isArray(value) ? value : []
  if (fd.w === 'media-uploader') {
    // Normalize admin response entries (object_key) to form model (key)
    if (!Array.isArray(value)) return []
    return (value as Record<string, string>[]).map((e) => ({
      key: e.key || e.object_key || '',
      alt_text: e.alt_text || '',
    }))
  }
  if (fd.w === 'switch') return value === true || value === 1 || value === 'true' ? 'true' : 'false'
  if (fd.w === 'datetime') return unixToDatetimeLocal(value)
  if (fd.w === 'number') return value == null ? '' : String(value)
  return value ?? ''
}

function buildFormPayload(): Record<string, unknown> {
  const payload: Record<string, unknown> = {}
  for (const section of resource.value.form.sections) {
    for (const fd of section.fields) {
      const value = formData[fd.k]
      if (fd.req && (value == null || value === '' || (Array.isArray(value) && value.length === 0))) {
        throw new Error(`${fd.l} 為必填欄位`)
      }
      if (fd.w === 'switch') {
        payload[fd.k] = value === true || value === 'true'
      } else if (fd.w === 'number') {
        if ((value === '' || value == null) && fd.nullable) {
          payload[fd.k] = null
        } else {
          const number = value === '' || value == null ? 0 : Number(value)
          if (!Number.isFinite(number)) throw new Error(`${fd.l} 必須是有效數字`)
          payload[fd.k] = number
        }
      } else if (fd.w === 'datetime') {
        if (value === '' || value == null) {
          payload[fd.k] = 0
        } else {
          const milliseconds = Date.parse(String(value))
          if (!Number.isFinite(milliseconds)) throw new Error(`${fd.l} 必須是有效日期時間`)
          payload[fd.k] = Math.floor(milliseconds / 1000)
        }
      } else if (fd.w === 'media-uploader') {
        // Convert form model entries to API input shape:
        // {key, alt_text} — existing entries from admin response use
        // object_key, new entries from upload use key.
        const entries = Array.isArray(value) ? value : []
        payload[fd.k] = entries.map((e: Record<string, string>) => ({
          key: e.key || e.object_key || '',
          alt_text: e.alt_text || '',
        }))
      } else {
        payload[fd.k] = value
      }
    }
  }
  return payload
}

function openForm(i: number | null) {
  formTriggerEl.value = (document.activeElement as HTMLElement) || null
  formRowIndex.value = i
  formError.value = null
  // Seed form data from the row (edit) or empty (create)
  const seed = i !== null ? rows.value[i] ?? {} : {}
  for (const k of Object.keys(formData)) delete formData[k]
  for (const sec of resource.value.form.sections) {
    for (const fd of sec.fields) {
      formData[fd.k] = seedFieldValue(fd, seed[fd.k])
    }
  }
  formInitial.value = JSON.stringify(formData)
  formOpen.value = true
}

/** Focus the first editable field inside the form modal, or the modal
 *  container as fallback. Called via watch after formOpen becomes true. */
function focusFormModal() {
  nextTick(() => {
    const modal = document.querySelector('.modal[role="dialog"]')
    if (!modal) return
    // Find the first editable input/textarea/select that is not disabled
    // or read-only. Skip inputs inside read-only fields (p.inp).
    const editable = modal.querySelector<HTMLInputElement | HTMLTextAreaElement | HTMLSelectElement>(
      '.fgrid input:not([disabled]), .fgrid textarea:not([disabled]), .fgrid select:not([disabled])',
    )
    if (editable) {
      editable.focus()
    } else {
      // Fallback: focus the modal container itself (tabindex=-1)
      ;(modal as HTMLElement).focus()
    }
  })
}

watch(formOpen, (open) => {
  if (open) focusFormModal()
})

function closeForm() {
  formOpen.value = false
  formRowIndex.value = null
  formError.value = null
  // Restore focus to the trigger element after the modal closes
  nextTick(() => {
    formTriggerEl.value?.focus()
    formTriggerEl.value = null
  })
}

function isFieldReadOnly(fd: FieldDef): boolean {
  return !!(fd.ro || formIsReadOnly.value || (fd.roOnEdit && !formIsNew.value))
}

function requestCloseForm() {
  if (!formIsReadOnly.value && formDirty.value && !window.confirm('尚有未儲存的變更，確定要離開？')) return
  closeForm()
}

async function saveForm() {
  const r = resource.value
  if (!r.api) {
    formError.value = '此資源未設定 API 端點，無法儲存。'
    return
  }
  formError.value = null
  mutating.value = true
  try {
    const payload = buildFormPayload()
    if (formIsNew.value && r.api.create) {
      await api.post(r.api.create, payload)
    } else if (!formIsNew.value && r.api.update) {
      const row = rows.value[formRowIndex.value!]
      const id = row?.id ?? row?.[r.cols[0].k]
      // Inject expected_draft_version from the row for optimistic
      // concurrency. This is NOT a form field — the user cannot edit it.
      // It is read from the row's draft_version at form-open time.
      if (row && row.draft_version != null) {
        payload.expected_draft_version = Number(row.draft_version)
      }
      if (r.expectedVersionField && row && row[r.expectedVersionField] != null) {
        payload.expected_version = Number(row[r.expectedVersionField])
      }
      await api.put(r.api.update.replace('{id}', String(id)), payload)
    }
    closeForm()
    await loadRows()
  } catch (e: any) {
    if (e instanceof ApiError && e.status === 409) {
      closeForm()
      await loadRows()
      error.value = '資料已被其他人更新，列表已重新載入；請重新開啟資料後再編輯。'
    } else {
      formError.value = e.message ?? String(e)
    }
  } finally {
    mutating.value = false
  }
}

// Confirm dialog
const confirmOpen = ref(false)
const confirmAction = ref<RowAction | BulkAction | null>(null)
const confirmRowIndex = ref<number | null>(null)
const confirmTitle = ref('')
const confirmBody = ref<ConfirmBody | null>(null)
const confirmMeta = ref<ConfirmMeta | null>(null)
const confirmRequireReason = ref(false)
const confirmVariant = ref<'pri' | 'danger'>('pri')
const confirmIsBulk = ref(false)
const confirmReason = ref('')
const confirmExpiry = ref('')

function askRowAction(i: number, actionKey: string) {
  const r = resource.value
  const a = r.rowActions.find((x) => x.k === actionKey)
  if (!a) return
  if (a.form) {
    openForm(i)
    return
  }
  // Restock action opens a dedicated per-item modal (not the confirm dialog)
  // because the operator must declare per-item returned/restocked quantities.
  if (a.restockItems) {
    openRestockModal(i, a)
    return
  }
  const row = rows.value[i]
  confirmAction.value = a
  confirmRowIndex.value = i
  confirmIsBulk.value = false
  confirmTitle.value = a.confirm ?? `確認${a.l}`
  confirmBody.value = {
    emphasis: String(row[r.cols[0].k] ?? ''),
    text: `— ${a.l}`,
  }
  confirmMeta.value = {
    expectValue: a.expect ? String(row[a.expect] ?? '') : undefined,
    payload: a.payload ?? undefined,
    op: a.op ?? undefined,
    expiryInput: a.expiryInput ?? false,
  }
  confirmRequireReason.value = !!a.reason
  confirmVariant.value = a.variant === 'danger' ? 'danger' : 'pri'
  confirmReason.value = ''
  confirmExpiry.value = ''
  confirmOpen.value = true
}

function askBulkAction(actionKey: string) {
  const r = resource.value
  const a = (r.bulkActions ?? []).find((x) => x.k === actionKey)
  if (!a) return
  confirmAction.value = a
  confirmRowIndex.value = null
  confirmIsBulk.value = true
  confirmTitle.value = a.confirm ?? `確認${a.l}`
  confirmBody.value = {
    emphasis: String(selected.value.size),
    text: `筆 — ${a.l}`,
  }
  confirmMeta.value = {
    note: '批次動作逐筆套用，完成後會顯示每一筆的結果；失敗的不會被當成成功。',
    op: a.op ?? undefined,
  }
  confirmRequireReason.value = !!a.reason
  confirmVariant.value = a.variant === 'danger' ? 'danger' : 'pri'
  confirmReason.value = ''
  confirmOpen.value = true
}

// Receipt for bulk actions
const bulkReceipt = ref<{ actionLabel: string; items: { id: string; ok: boolean; msg?: string }[] } | null>(null)

// ----- Restock modal (per-item returned/restocked quantities) -----
// The restock modal fetches the AUTHORITATIVE order detail (GET order by
// ID) on open — the list row's items come from items_json and do NOT
// include the order_items ledger columns (returned_quantity,
// restocked_quantity). The modal must display the existing cumulative
// ledger values so the operator knows what has already been
// received/restocked, and the input fields are THIS-ACTION deltas
// (defaulting to 0), not cumulative totals.
//
// Dynamic max constraints per item:
//   maxReturned  = quantity - existingReturned
//   maxRestocked = (existingReturned + thisReturned) - existingRestocked
// The restocked max is reactive on the returned input.
//
// The idempotency key is generated ONCE when the modal opens. On the
// FIRST submit, the payload fingerprint is computed and stored. On
// subsequent submits, if the payload fingerprint changed (operator
// edited reason/quantities), a NEW key is generated to avoid same-key
// different-intent. If the fingerprint is unchanged (retry), the same
// key is reused so the server can detect replay.
interface RestockItemRow {
  sku: string
  name: string
  quantity: number
  existingReturned: number
  existingRestocked: number
  returned: number  // this-action delta
  restocked: number // this-action delta
}
const restockOpen = ref(false)
const restockLoading = ref(false)
const restockTitle = ref('')
const restockAction = ref<RowAction | null>(null)
const restockRowIndex = ref<number | null>(null)
const restockOrderId = ref('')
const restockExpectedVersion = ref(0)
const restockListVersion = ref(0) // version from the stale list row, for warning
const restockIdempotencyKey = ref('')
const restockReason = ref('')
const restockItems = ref<RestockItemRow[]>([])
const restockError = ref<string | null>(null)
const restockSubmittedFingerprint = ref<string | null>(null)
const restockModalRef = ref<HTMLElement | null>(null)

/** Compute a stable fingerprint of the current restock payload (without
 *  the idempotency key, which is derived from the fingerprint). This is
 *  used to decide whether to rotate the key on a subsequent submit. */
function computeRestockPayloadFingerprint(): string {
  const payload = {
    order_id: restockOrderId.value,
    expected_version: restockExpectedVersion.value,
    reason: restockReason.value.trim(),
    items: restockItems.value
      .filter((it) => it.returned > 0 || it.restocked > 0)
      .map((it) => ({ sku: it.sku, returned_quantity: it.returned, restocked_quantity: it.restocked })),
  }
  return JSON.stringify(payload)
}

async function openRestockModal(i: number, a: RowAction) {
  const r = resource.value
  const row = rows.value[i]
  if (!row || !r.api?.restock || !r.api?.get) return
  restockAction.value = a
  restockRowIndex.value = i
  restockOrderId.value = String(row.id ?? '')
  restockListVersion.value = a.expect ? Number(row[a.expect] ?? 0) : 0
  restockTitle.value = a.confirm ?? `確認${a.l}`
  restockIdempotencyKey.value = crypto.randomUUID()
  restockSubmittedFingerprint.value = null
  restockReason.value = ''
  restockError.value = null
  restockItems.value = []
  restockOpen.value = true
  restockLoading.value = true
  // Fetch the AUTHORITATIVE order detail — the list row's items come
  // from items_json and do NOT include the order_items ledger columns.
  // The GET endpoint returns items with returned_quantity/restocked_quantity
  // merged from the order_items ledger by the service layer.
  try {
    const detailPath = r.api.get.replace('{id}', restockOrderId.value)
    const detail = await api.get<Record<string, any>>(detailPath)
    const rawItems = Array.isArray(detail.items) ? detail.items : []
    restockExpectedVersion.value = Number(detail.version ?? restockListVersion.value)
    restockItems.value = rawItems.map((it: any) => ({
      sku: String(it.sku ?? ''),
      name: String(it.name ?? ''),
      quantity: Number(it.quantity ?? 0),
      existingReturned: Number(it.returned_quantity ?? 0),
      existingRestocked: Number(it.restocked_quantity ?? 0),
      returned: 0,
      restocked: 0,
    }))
    // Warn if the list row version is stale relative to the authoritative detail.
    if (restockListVersion.value > 0 && restockListVersion.value !== restockExpectedVersion.value) {
      restockError.value = `注意：列表中的版本為 ${restockListVersion.value}，但訂單詳情版本為 ${restockExpectedVersion.value}。資料可能已被其他人更新，請確認後再送出。`
    }
  } catch (e: any) {
    restockError.value = `無法載入訂單詳情：${e.message ?? String(e)}`
  } finally {
    restockLoading.value = false
  }
}

const restockCanSubmit = computed(() => {
  if (restockLoading.value) return false
  if (restockReason.value.trim().length === 0) return false
  if (restockItems.value.length === 0) return false
  // At least one item must have a positive delta (returned OR restocked).
  // This allows "previously received 2, now restock remaining 1" where
  // this-action returned=0, restocked=1.
  if (!restockItems.value.some((it) => it.returned > 0 || it.restocked > 0)) return false
  // Per-item validation using CUMULATIVE constraints only.
  // There is NO per-action restocked <= returned constraint — a restock-only
  // follow-up (returned=0, restocked=1) is legal when existingReturned >
  // existingRestocked. The cumulative constraint is the sole authority.
  for (const it of restockItems.value) {
    if (it.returned < 0 || it.restocked < 0) return false
    // Cumulative returned must not exceed quantity.
    if (it.existingReturned + it.returned > it.quantity) return false
    // Cumulative restocked must not exceed cumulative returned.
    if (it.existingRestocked + it.restocked > it.existingReturned + it.returned) return false
  }
  return true
})

/** Max returned delta for an item: quantity - existingReturned. */
function maxReturned(it: RestockItemRow): number {
  return Math.max(0, it.quantity - it.existingReturned)
}

/** Max restocked delta for an item: (existingReturned + thisReturned) - existingRestocked.
 *  Reactive on the returned input. */
function maxRestocked(it: RestockItemRow): number {
  return Math.max(0, it.existingReturned + it.returned - it.existingRestocked)
}

function buildRestockBody(): Record<string, any> {
  return {
    expected_version: restockExpectedVersion.value,
    idempotency_key: restockIdempotencyKey.value,
    reason: restockReason.value.trim(),
    items: restockItems.value
      .filter((it) => it.returned > 0 || it.restocked > 0)
      .map((it) => ({
        sku: it.sku,
        returned_quantity: it.returned,
        restocked_quantity: it.restocked,
      })),
  }
}

async function submitRestock() {
  const r = resource.value
  if (!r.api?.restock || !restockAction.value) return
  if (!restockCanSubmit.value) return
  // Stable key binding: if this is NOT the first submit, and the payload
  // fingerprint changed since the last submit, rotate the key to avoid
  // same-key different-intent. If the fingerprint is unchanged (retry),
  // reuse the same key so the server can detect replay.
  const currentFingerprint = computeRestockPayloadFingerprint()
  if (restockSubmittedFingerprint.value !== null && restockSubmittedFingerprint.value !== currentFingerprint) {
    restockIdempotencyKey.value = crypto.randomUUID()
  }
  restockSubmittedFingerprint.value = currentFingerprint
  mutating.value = true
  restockError.value = null
  try {
    const path = r.api.restock.replace('{id}', restockOrderId.value)
    await api.post(path, buildRestockBody())
    restockOpen.value = false
    restockAction.value = null
    await loadRows()
  } catch (e: any) {
    // Keep the modal open on error — the idempotency key is preserved
    // so the operator can retry the SAME intent without generating a
    // new key. On 409 conflict, the operator reloads and reopens.
    const isConflict = e?.status === 409
    restockError.value = isConflict
      ? `衝突：這筆訂單已被別人改過（版本不符或退貨狀態變更）。請重新載入列表後再試一次。原始錯誤：${e.message ?? String(e)}`
      : e.message ?? String(e)
  } finally {
    mutating.value = false
  }
}

/** Resolve the API path + HTTP method for a row action based on its operationId. */
function resolveActionEndpoint(a: RowAction | BulkAction): { method: 'PATCH' | 'DELETE' | 'POST' | 'PUT'; path: string } | null {
  const r = resource.value
  if (!r.api) return null
  if (a.op === r.ops.status && r.api.status) return { method: 'PATCH', path: r.api.status }
  if (a.op === r.ops.returnStatus && r.api.returnStatus) return { method: 'PATCH', path: r.api.returnStatus }
  if (a.op === r.ops.publish && r.api.publish) return { method: 'POST', path: r.api.publish }
  if (a.op === r.ops.approve && r.api.approve) return { method: 'POST', path: r.api.approve }
  if (a.op === r.ops.restock && r.api.restock) return { method: 'POST', path: r.api.restock }
  if (a.op === r.ops.del && r.api.delete) return { method: 'DELETE', path: r.api.delete }
  if (a.op === r.ops.update && r.api.update) return { method: 'PUT', path: r.api.update }
  if (a.op === r.ops.create && r.api.create) return { method: 'POST', path: r.api.create }
  return null
}

function rowIdOf(row: Record<string, any>): string {
  return String(row.id ?? row[resource.value.cols[0].k] ?? '')
}

/** Build the request body for a status/return-status PATCH. */
function buildStatusBody(a: RowAction | BulkAction, row: Record<string, any>): Record<string, any> {
  const r = resource.value
  // Order status & return-status endpoints expect { expected_version, new_status }
  if ((a.op === r.ops.status || a.op === r.ops.returnStatus) && 'expect' in a && a.expect) {
    return {
      expected_version: row[a.expect],
      new_status: a.payload?.status,
      note: confirmReason.value.trim(),
    }
  }
  // Product/member status endpoints expect { status }
  return { ...(a.payload ?? {}) }
}

/** Build the request body for a POST action (approve/publish). */
function buildPostBody(a: RowAction, row: Record<string, any>): Record<string, any> {
  const r = resource.value
  // Approve: send expected_draft_version + expiry_unix from operator input.
  if (a.op === r.ops.approve && 'expect' in a && a.expect) {
    const expiryUnix = datetimeLocalToUnix(confirmExpiry.value)
    return {
      expected_draft_version: row[a.expect],
      expiry_unix: expiryUnix,
    }
  }
  // Publish: send expected_draft_version only.
  if (a.op === r.ops.publish && 'expect' in a && a.expect) {
    return {
      expected_draft_version: row[a.expect],
    }
  }
  return { ...(a.payload ?? {}) }
}

/** Convert a datetime-local string (e.g. "2024-01-01T12:00") to a Unix
 *  timestamp in seconds. Returns 0 if the input is empty/invalid. */
function datetimeLocalToUnix(dt: string): number {
  if (!dt) return 0
  const ms = Date.parse(dt)
  if (isNaN(ms)) return 0
  return Math.floor(ms / 1000)
}

async function runConfirm() {
  const a = confirmAction.value
  if (!a) return
  const r = resource.value

  if (confirmIsBulk.value) {
    const indices = [...selected.value]
    const items: { id: string; ok: boolean; msg?: string }[] = []
    const endpoint = resolveActionEndpoint(a)
    if (!endpoint || !r.api) {
      // No fixture fallback — surface truthful error.
      bulkReceipt.value = {
        actionLabel: a.l,
        items: indices.map((i) => ({
          id: String(rows.value[i]?.[r.cols[0].k] ?? i),
          ok: false,
          msg: '此資源未設定 API 端點，無法執行動作。',
        })),
      }
      selected.value = new Set()
      confirmOpen.value = false
      confirmAction.value = null
      return
    }
    mutating.value = true
    for (const i of indices) {
      const row = rows.value[i]
      const id = rowIdOf(row)
      try {
        const path = endpoint.path.replace('{id}', id)
        if (endpoint.method === 'PATCH') {
          await api.patch(path, buildStatusBody(a, row))
        } else if (endpoint.method === 'DELETE') {
          await api.del(path)
        } else if (endpoint.method === 'PUT') {
          await api.put(path, a.payload ?? {})
        }
        items.push({ id, ok: true })
      } catch (e: any) {
        items.push({ id, ok: false, msg: e.message ?? String(e) })
      }
    }
    bulkReceipt.value = { actionLabel: a.l, items }
    selected.value = new Set()
    confirmOpen.value = false
    confirmAction.value = null
    mutating.value = false
    await loadRows()
    return
  }

  // Single row action
  const i = confirmRowIndex.value
  const row = i !== null ? rows.value[i] : null
  const endpoint = resolveActionEndpoint(a)

  if (!endpoint || !r.api || !row) {
    // No fixture fallback — surface truthful error.
    confirmMeta.value = {
      ...confirmMeta.value,
      error: '此資源未設定 API 端點，無法執行動作。',
    }
    return
  }

  mutating.value = true
  try {
    const path = endpoint.path.replace('{id}', rowIdOf(row))
    if (endpoint.method === 'PATCH') {
      await api.patch(path, buildStatusBody(a, row))
    } else if (endpoint.method === 'DELETE') {
      await api.del(path)
    } else if (endpoint.method === 'PUT') {
      await api.put(path, a.payload ?? {})
    } else if (endpoint.method === 'POST') {
      await api.post(path, buildPostBody(a, row))
    }
    confirmOpen.value = false
    confirmAction.value = null
    await loadRows()
  } catch (e: any) {
    // Surface the error inline on the confirm dialog (structured, no v-html).
    // 409 Conflict: the row was modified by someone else — keep the dialog
    // open so the operator can reload and retry. Do NOT fake success.
    const isConflict = e?.status === 409
    const msg = isConflict
      ? `衝突：這筆資料已被別人改過（版本不符或核可已過期/不存在）。請重新載入列表後再試一次。原始錯誤：${e.message ?? String(e)}`
      : e.message ?? String(e)
    confirmMeta.value = {
      ...confirmMeta.value,
      error: msg,
    }
  } finally {
    mutating.value = false
  }
}

// Bulk actions available to current role
const availableBulkActions = computed(() =>
  (resource.value.bulkActions ?? []).filter((a) => auth.can(a.cap)),
)

const canCreate = computed(() =>
  resource.value.createCap && auth.can(resource.value.createCap) && resource.value.ops.create,
)

const formIsReadOnly = computed(() => resource.value.form.readOnly ?? false)
</script>

<template>
  <!-- Page header -->
  <div class="pagehd">
    <div>
      <h1>{{ resource.label }}</h1>
      <div class="sub">{{ resource.desc }}</div>
    </div>
    <div style="display:flex;gap:8px">
      <Button v-if="canCreate" variant="pri" @click="openForm(null)">
        <Plus />
        新增{{ resource.label }}
      </Button>
      <Button v-else-if="resource.ops.create" disabled>新增{{ resource.label }}</Button>
      <Button size="sm" variant="ghost" :disabled="loading" @click="loadRows">重新載入</Button>
    </div>
  </div>

  <!-- State machines (orders only) -->
  <template v-if="MACHINES[resourceKey]">
    <section
      v-for="(m, mi) in MACHINES[resourceKey]"
      :key="mi"
      class="panel"
      style="margin-bottom:12px"
    >
      <div class="machine">
        <b style="margin-right:10px;font-size:14.5px">{{ m.t }}</b>
        <template v-for="(s, si) in m.flow" :key="s">
          <span v-if="si" class="ar">→</span>
          <span :class="['n', { cur: si === 0 }]">{{ LABEL[s] ?? s }}</span>
        </template>
        <span class="alt">{{ m.alt }}</span>
      </div>
    </section>
    <div class="note" style="margin-bottom:12px">
      <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.7" stroke-linecap="round" stroke-linejoin="round">
        <circle cx="12" cy="12" r="9" /><path d="M12 16v-4M12 8h.01" />
      </svg>
      <div>
        <b>兩條狀態機互不干涉。</b>履約走一條、退貨走另一條，各自只在對應狀態出現按鈕。每個轉移都會帶 <code>expected_version</code>——如果列表資料過期（別人先改過），伺服器會擋下來而不是覆寫。
      </div>
    </div>
  </template>

  <!-- Main panel: filters + bulk + table + footer -->
  <section class="panel">
    <!-- Toolbar / filters -->
    <div class="toolbar">
      <template v-if="resource.filters.length">
        <template v-for="f in resource.filters" :key="f.k">
          <Select
            v-if="f.w === 'select'"
            :options="f.opts ?? []"
            modelValue=""
            style="min-width:120px"
          />
          <Input
            v-else
            :placeholder="f.l"
            style="min-width:120px"
          />
        </template>
      </template>
      <span v-else class="muted">這個資源沒有定義篩選器</span>
      <div style="flex:1" />
      <span class="muted">每頁 {{ resource.pageSize }} 筆</span>
    </div>

    <!-- Bulk action bar -->
    <div
      v-if="selected.size > 0 && availableBulkActions.length > 0"
      class="bulkbar"
    >
      <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.7" stroke-linecap="round" stroke-linejoin="round">
        <path d="M20 6 9 17l-5-5" />
      </svg>
      已選 <b>{{ selected.size }}</b> 筆
      <div style="flex:1" />
      <Button
        v-for="a in availableBulkActions"
        :key="a.k"
        size="sm"
        :variant="a.variant === 'danger' ? 'danger' : 'default'"
        @click="askBulkAction(a.k)"
      >{{ a.l }}</Button>
      <Button size="sm" variant="ghost" @click="selected = new Set()">取消</Button>
    </div>

    <!-- Loading / error states -->
    <div v-if="loading" class="emptybox">
      <div class="ic">
        <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.7" stroke-linecap="round" stroke-linejoin="round">
          <path d="M12 6v6l4 2" /><circle cx="12" cy="12" r="9" />
        </svg>
      </div>
      <b>載入中…</b>
      <p>正在從 API 取得{{ resource.label }}資料。</p>
    </div>
    <div v-else-if="error" class="emptybox">
      <div class="ic" style="color:var(--danger)">
        <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.7" stroke-linecap="round" stroke-linejoin="round">
          <circle cx="12" cy="12" r="9" /><path d="M12 8v4M12 16h.01" />
        </svg>
      </div>
      <b>載入失敗</b>
      <p class="mono" style="word-break:break-all">{{ error }}</p>
      <Button size="sm" variant="ghost" @click="loadRows">重試</Button>
    </div>

    <!-- Table -->
    <ResourceTable
      v-else
      :resource="resource"
      :rows="rows"
      :selected="selected"
      @toggle-row="toggleRow"
      @toggle-all="toggleAll"
      @row-action="askRowAction"
    />

    <!-- Footer -->
    <div class="tfoot">
      <span>顯示 {{ rows.length }} 筆</span>
      <span>
        operationId：
        <span
          v-for="(opVal, i) in Object.values(resource.ops).filter(Boolean)"
          :key="i"
          class="mono"
        >{{ i > 0 ? '　' : '' }}{{ opVal }}</span>
      </span>
    </div>
  </section>

  <!-- Bulk receipt -->
  <section v-if="bulkReceipt" class="panel" style="margin-top:12px">
    <div class="phd">
      <h3>{{ bulkReceipt.actionLabel }} 結果</h3>
      <span class="more muted">逐筆回報</span>
    </div>
    <div
      v-for="(item, i) in bulkReceipt.items"
      :key="i"
      class="rowi"
    >
      <span class="mono">{{ item.id }}</span>
      <span v-if="item.ok" class="st success" style="margin-left:auto">已套用</span>
      <template v-else>
        <span class="st danger" style="margin-left:auto">失敗</span>
        <span class="muted mono" style="margin-left:8px;word-break:break-all">{{ item.msg }}</span>
      </template>
    </div>
  </section>

  <!-- Form / Detail modal -->
  <Modal
    :open="formOpen"
    :title="(formIsNew ? '新增' : (formIsReadOnly ? '' : '編輯')) + resource.form.title"
    @close="requestCloseForm"
  >
    <!-- Read-only note -->
    <div v-if="formIsReadOnly" class="note">
      <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.7" stroke-linecap="round" stroke-linejoin="round">
        <circle cx="12" cy="12" r="9" /><path d="M12 16v-4M12 8h.01" />
      </svg>
      <div>這個資源的表單是<b>唯讀明細</b>（<code>hideSubmit</code>）。訂單不靠表單改狀態，狀態一律走清單上的動作按鈕，才能帶 <code>expected_version</code>。</div>
    </div>

    <!-- Form error -->
    <div v-if="formError" class="note" style="color:var(--danger)">
      <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.7" stroke-linecap="round" stroke-linejoin="round">
        <circle cx="12" cy="12" r="9" /><path d="M12 16v-4M12 8h.01" />
      </svg>
      <div><b>儲存失敗：</b><span class="mono" style="word-break:break-all">{{ formError }}</span></div>
    </div>
    <div v-else-if="formDirty && !formIsReadOnly" class="note">尚有未儲存的變更</div>

    <!-- Form sections -->
    <div class="fgrid">
      <template v-for="sec in resource.form.sections" :key="sec.t">
        <div class="span2" style="font-size:12px;font-weight:700;color:var(--text-3);margin:4px 0 8px">{{ sec.t }}</div>
        <template v-for="fd in sec.fields" :key="fd.k">
          <div :class="['field', fd.span === 2 ? 'span2' : '']">
            <label
              :for="isFieldReadOnly(fd) ? undefined : (fd.w === 'media-uploader' ? undefined : 'field-' + fd.k)"
              :id="isFieldReadOnly(fd) ? undefined : (fd.w === 'media-uploader' ? 'label-field-' + fd.k : undefined)"
            >
              {{ fd.l }}
              <span v-if="fd.req" class="req">*</span>
            </label>
            <!-- Read-only fields -->
            <p v-if="isFieldReadOnly(fd)" class="inp" style="background:var(--surface-2);color:var(--text-2);cursor:default">
              {{ formData[fd.k] ?? '—' }}
            </p>
            <!-- Textarea -->
            <Textarea
              v-else-if="fd.w === 'textarea'"
              :id="'field-' + fd.k"
              :modelValue="formData[fd.k] ?? ''"
              :placeholder="fd.l"
              @update:modelValue="formData[fd.k] = $event"
            />
            <!-- Select -->
            <Select
              v-else-if="fd.w === 'select'"
              :id="'field-' + fd.k"
              :options="fd.opts ?? []"
              :modelValue="formData[fd.k] ?? ''"
              @update:modelValue="formData[fd.k] = $event"
            />
            <!-- Switch (rendered as select) -->
            <Select
              v-else-if="fd.w === 'switch'"
              :id="'field-' + fd.k"
              :options="['true', 'false']"
              :modelValue="formData[fd.k] ?? 'false'"
              @update:modelValue="formData[fd.k] = $event"
            />
            <!-- Tags input -->
            <Input
              v-else-if="fd.w === 'tags'"
              :id="'field-' + fd.k"
              :modelValue="formData[fd.k] ?? ''"
              placeholder="以逗號分隔"
              @update:modelValue="formData[fd.k] = $event"
            />
            <!-- Media uploader (product_images) -->
            <MediaUploader
              v-else-if="fd.w === 'media-uploader'"
              :modelValue="formData[fd.k] ?? []"
              :readOnly="formIsReadOnly"
              :help="fd.help"
              :labelledby="'label-field-' + fd.k"
              @update:modelValue="formData[fd.k] = $event"
            />
            <!-- Number / Text -->
            <Input
              v-else
              :id="'field-' + fd.k"
              :type="fd.w === 'number' ? 'number' : fd.w === 'datetime' ? 'datetime-local' : 'text'"
              :modelValue="formData[fd.k] ?? ''"
              :placeholder="fd.l"
              @update:modelValue="formData[fd.k] = $event"
            />
          </div>
        </template>
      </template>
    </div>

    <!-- Form footer -->
    <template #footer>
      <Button v-if="formIsReadOnly" @click="closeForm">關閉</Button>
      <template v-else>
        <Button variant="ghost" @click="requestCloseForm">取消</Button>
        <Button variant="pri" :disabled="mutating || (!formIsNew && !formDirty)" @click="saveForm">儲存</Button>
      </template>
    </template>
  </Modal>

  <!-- Confirm dialog -->
  <ConfirmDialog
    :open="confirmOpen"
    :title="confirmTitle"
    :variant="confirmVariant"
    :body="confirmBody"
    :meta="confirmMeta"
    :requireReason="confirmRequireReason"
    :reason="confirmReason"
    :expiry="confirmExpiry"
    confirmLabel="確認"
    @confirm="runConfirm"
    @cancel="confirmOpen = false"
    @update:reason="confirmReason = $event"
    @update:expiry="confirmExpiry = $event"
  />

  <!-- Restock modal: per-item editable returned/restocked quantities -->
  <Modal
    :open="restockOpen"
    :title="restockTitle"
    max-width="min(640px, 94vw)"
    @close="restockOpen = false"
  >
    <div v-if="restockLoading" class="note" role="status" aria-live="polite">
      載入訂單詳情中...
    </div>
    <template v-else>
    <div class="note">
      訂單 <span class="mono">{{ restockOrderId }}</span> — 逐品項驗收回補。
      已收件數量為本次實際收到退貨的單位數；回補可售數量為檢驗後可重新銷售的單位數（瑕疵品不回補）。
      預設為 0，請依實際驗收結果填入。下方顯示既有累積量供參考。
    </div>
    <table v-if="restockItems.length > 0" class="restock-table">
      <thead>
        <tr>
          <th>品項</th>
          <th class="num">訂購</th>
          <th class="num">既有已收</th>
          <th class="num">本次收件</th>
          <th class="num">既有已補</th>
          <th class="num">本次回補</th>
        </tr>
      </thead>
      <tbody>
        <tr v-for="(it, idx) in restockItems" :key="it.sku">
          <td>
            <div class="mono">{{ it.sku }}</div>
            <div class="muted">{{ it.name }}</div>
          </td>
          <td class="num">{{ it.quantity }}</td>
          <td class="num muted">{{ it.existingReturned }}</td>
          <td class="num">
            <input
              type="number"
              class="restock-input"
              :min="0"
              :max="maxReturned(it)"
              :value="it.returned"
              :aria-label="`本次收件數量 for ${it.sku}`"
              @input="restockItems[idx].returned = Math.max(0, Number(($event.target as HTMLInputElement).value) || 0)"
            />
          </td>
          <td class="num muted">{{ it.existingRestocked }}</td>
          <td class="num">
            <input
              type="number"
              class="restock-input"
              :min="0"
              :max="maxRestocked(it)"
              :value="it.restocked"
              :aria-label="`本次回補可售數量 for ${it.sku}`"
              @input="restockItems[idx].restocked = Math.max(0, Math.min(maxRestocked(it), Number(($event.target as HTMLInputElement).value) || 0))"
            />
          </td>
        </tr>
      </tbody>
    </table>
    <div v-else class="note danger">此訂單沒有品項資料，無法驗收回補。</div>
    <div class="field" style="margin-top:12px">
      <label for="restock-reason-input">原因 <span style="color:var(--danger)">*</span></label>
      <Textarea
        id="restock-reason-input"
        :model-value="restockReason"
        placeholder="請輸入驗收原因..."
        @update:model-value="restockReason = $event"
      />
    </div>
    <div v-if="restockError" class="st danger" style="margin-top:8px" role="alert" aria-live="polite">
      {{ restockError }}
    </div>
    </template>
    <template #footer>
      <Button variant="ghost" @click="restockOpen = false" :disabled="mutating">取消</Button>
      <Button
        variant="pri"
        :disabled="!restockCanSubmit || mutating || restockLoading"
        @click="submitRestock"
      >
        確認回補
      </Button>
    </template>
  </Modal>
</template>

<style scoped>
.restock-table {
  width: 100%;
  border-collapse: collapse;
  margin-top: 12px;
  font-size: 0.9rem;
}
.restock-table th,
.restock-table td {
  padding: 8px 10px;
  border-bottom: 1px solid var(--border, #e5e7eb);
  text-align: left;
  vertical-align: top;
}
.restock-table th {
  font-weight: 600;
  color: var(--text-2, #666);
  font-size: 0.85rem;
}
.restock-table .num {
  text-align: center;
  white-space: nowrap;
}
.restock-input {
  width: 64px;
  padding: 4px 6px;
  border: 1px solid var(--border, #ccc);
  border-radius: 4px;
  font-size: 0.9rem;
  text-align: center;
  background: var(--surface, #fff);
  color: var(--text, #222);
}
.restock-input:focus {
  outline: none;
  border-color: var(--primary, #2563eb);
  box-shadow: 0 0 0 2px rgba(37, 99, 235, 0.15);
}
</style>
