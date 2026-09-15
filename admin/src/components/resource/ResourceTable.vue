<script setup lang="ts">
import { computed, ref, watch, onMounted, onBeforeUnmount, nextTick } from 'vue'
import { ArrowUp, ArrowDown, ChevronsUpDown, MoreHorizontal } from 'lucide-vue-next'
import { TONE, LABEL } from '@/config/tones'
import { formatNT, evalShowWhen } from '@/lib/utils'
import type { Col, ResourceDef, MenuItem } from '@/lib/types'
import Badge from '@/components/ui/Badge.vue'
import Checkbox from '@/components/ui/Checkbox.vue'
import Button from '@/components/ui/Button.vue'
import DropdownMenu from '@/components/ui/DropdownMenu.vue'
import { useAuthStore } from '@/stores/auth'

const props = defineProps<{
  resource: ResourceDef
  rows: Record<string, any>[]
  selected: Set<number>
  /** Raw value -> display label per column (e.g. category slug -> name). */
  colLabels?: Record<string, Record<string, string>>
}>()

const emit = defineEmits<{
  toggleRow: [i: number]
  toggleAll: []
  rowAction: [i: number, actionKey: string]
}>()

const auth = useAuthStore()

const selectable = computed(() => (props.resource.bulkActions ?? []).filter((a) => auth.can(a.cap)).length > 0)

// ----- Client-side sorting (opt-in per column) ---------------------------
// Sort state never fabricates or reorders server-owned semantics: columns
// without `sortable` are inert and the row order untouched.
const sortKey = ref<string | null>(null)
const sortDir = ref<'asc' | 'desc'>('asc')

function toggleSort(col: Col) {
  if (!col.sortable) return
  if (sortKey.value !== col.k) {
    sortKey.value = col.k
    sortDir.value = 'asc'
  } else {
    sortDir.value = sortDir.value === 'asc' ? 'desc' : 'asc'
  }
}

function ariaSort(col: Col): 'ascending' | 'descending' | 'none' | undefined {
  if (!col.sortable) return undefined
  if (sortKey.value !== col.k) return 'none'
  return sortDir.value === 'asc' ? 'ascending' : 'descending'
}

function sortIcon(col: Col) {
  if (sortKey.value !== col.k) return ChevronsUpDown
  return sortDir.value === 'asc' ? ArrowUp : ArrowDown
}

/** Entries carry the row's index within `props.rows` so selection and row
 *  actions keep their meaning regardless of client-side sort order. */
const displayEntries = computed(() => {
  const entries = props.rows.map((row, i) => ({ row, i }))
  const key = sortKey.value
  if (!key) return entries
  const col = props.resource.cols.find((c) => c.k === key)
  if (!col?.sortable) return entries
  const dir = sortDir.value === 'asc' ? 1 : -1
  return [...entries].sort((a, b) => {
    const va = a.row[key]
    const vb = b.row[key]
    const aEmpty = va === null || va === undefined || va === ''
    const bEmpty = vb === null || vb === undefined || vb === ''
    if (aEmpty !== bEmpty) return aEmpty ? 1 : -1
    if (aEmpty) return 0
    if (typeof va === 'number' && typeof vb === 'number') return (va - vb) * dir
    return String(va).localeCompare(String(vb), 'zh-Hant') * dir
  })
})

function toneFor(v: unknown) {
  return TONE[String(v)] ?? 'neutral'
}

function labelFor(col: Col, v: unknown) {
  const key = String(v)
  const mapped = props.colLabels?.[col.k]?.[key]
  if (mapped !== undefined) return mapped
  return LABEL[key] !== undefined ? LABEL[key] : key
}

/** Renders null/undefined as an em dash. Preserves numeric 0 as "0".
 *  Does not conflate absent (null/undefined) with present-but-zero. */
function absentDash(v: unknown): string {
  return v === null || v === undefined ? '—' : String(v)
}

function cellContent(col: Col, row: Record<string, any>) {
  const v = row[col.k]
  if (col.r === 'mono') return { type: 'mono' as const, value: absentDash(v) }
  if (col.r === 'badge') return { type: 'badge' as const, value: v, tone: toneFor(v), label: labelFor(col, v) }
  if (col.r === 'number') {
    if (v === null || v === undefined) return { type: 'number' as const, value: '—' }
    const isMoney = col.k === 'price' || col.k === 'total' || col.k === 'total_spent'
    return { type: 'number' as const, value: isMoney ? formatNT(v) : String(v) }
  }
  if (col.r === 'datetime') return { type: 'datetime' as const, value: v }
  return { type: 'text' as const, value: absentDash(v) }
}

function visibleActions(row: Record<string, any>) {
  return (props.resource.rowActions ?? []).filter((a) => evalShowWhen(a.showWhen, row))
}

/** Rows with multiple actions collapse the overflow into a dropdown menu;
 *  the first action stays inline for one-click access. */
const INLINE_ACTIONS = 1

function inlineActions(row: Record<string, any>) {
  const acts = visibleActions(row)
  return acts.length <= 2 ? acts : acts.slice(0, INLINE_ACTIONS)
}

function menuActions(row: Record<string, any>) {
  const acts = visibleActions(row)
  return acts.length <= 2 ? [] : acts.slice(INLINE_ACTIONS)
}

function menuItemsFor(row: Record<string, any>): MenuItem[] {
  return menuActions(row).map((a) => ({
    key: a.k,
    label: a.l,
    danger: a.variant === 'danger',
    disabled: !actionAllowed(a),
    hint: actionAllowed(a) ? undefined : `需要 ${missingCapsLabel(a)}`,
  }))
}

function onMenuSelect(rowIndex: number, actionKey: string) {
  emit('rowAction', rowIndex, actionKey)
}

// ----- Horizontal scroll + pinned columns --------------------------------
// The table is width:max-content inside an overflow-x:auto wrap, so wide
// column sets scroll instead of squeezing. Pinned cells are position:sticky;
// their left/right offsets are measured from the rendered header widths so
// several pins on one side stack outward correctly.

const wrapEl = ref<HTMLElement | null>(null)
const pinLeftPx = ref<number[]>([])
const pinRightPx = ref<number[]>([])

const anyPinLeft = computed(() => props.resource.cols.some((c) => c.pin === 'left'))
const pinActions = computed(() => !!props.resource.pinActions)
// The select-all checkbox column must pin too when a left-pinned column
// exists — otherwise horizontal scroll slides the pinned cell over it.
const pinCheckbox = computed(() => selectable.value && anyPinLeft.value)

function measurePins() {
  const headRow = wrapEl.value?.querySelector('thead tr')
  if (!headRow) return
  const ths = [...headRow.children] as HTMLElement[]
  let i = 0
  let chkW = 0
  if (selectable.value) chkW = ths[i++]?.offsetWidth ?? 0
  const widths = props.resource.cols.map(() => ths[i++]?.offsetWidth ?? 0)
  const actW = ths[i]?.offsetWidth ?? 0

  const left = new Array<number>(props.resource.cols.length).fill(0)
  const right = new Array<number>(props.resource.cols.length).fill(0)
  let acc = pinCheckbox.value ? chkW : 0
  props.resource.cols.forEach((c, idx) => {
    if (c.pin === 'left') {
      left[idx] = acc
      acc += widths[idx] ?? 0
    }
  })
  acc = pinActions.value ? actW : 0
  for (let idx = props.resource.cols.length - 1; idx >= 0; idx--) {
    const c = props.resource.cols[idx]
    if (c.pin === 'right') {
      right[idx] = acc
      acc += widths[idx] ?? 0
    }
  }
  // Assign only on real change so the style binding doesn't retrigger render.
  if (JSON.stringify(left) !== JSON.stringify(pinLeftPx.value)) pinLeftPx.value = left
  if (JSON.stringify(right) !== JSON.stringify(pinRightPx.value)) pinRightPx.value = right
}

let pinObserver: ResizeObserver | null = null

onMounted(() => {
  void nextTick(measurePins)
  if (typeof ResizeObserver !== 'undefined' && wrapEl.value) {
    pinObserver = new ResizeObserver(measurePins)
    pinObserver.observe(wrapEl.value)
  }
})

watch(
  () => [props.rows, props.resource, selectable.value, displayEntries.value.length],
  () => void nextTick(measurePins),
)

onBeforeUnmount(() => {
  pinObserver?.disconnect()
  pinObserver = null
})

/** Inline sticky offset for a pinned data column (undefined = not pinned). */
function pinStyle(col: Col, i: number): Record<string, string> | undefined {
  if (col.pin === 'left') return { left: `${pinLeftPx.value[i] ?? 0}px` }
  if (col.pin === 'right') return { right: `${pinRightPx.value[i] ?? 0}px` }
  return undefined
}

/** Returns the merged all-of required capability list for an action,
 *  combining cap (single) and allCaps (list) with de-duplication.
 *  When both are set, BOTH gates apply — allCaps never shadows cap.
 *  Returns a stable deduped array (preserves first-seen order). */
function requiredCaps(a: { cap?: string; allCaps?: string[] }): string[] {
  const seen = new Set<string>()
  const out: string[] = []
  if (a.cap) {
    if (!seen.has(a.cap)) { seen.add(a.cap); out.push(a.cap) }
  }
  if (a.allCaps) {
    for (const c of a.allCaps) {
      if (!seen.has(c)) { seen.add(c); out.push(c) }
    }
  }
  return out
}

/** Returns true if the current principal holds every capability required
 *  by the action. cap and allCaps are merged into one all-of list. */
function actionAllowed(a: { cap?: string; allCaps?: string[] }): boolean {
  const required = requiredCaps(a)
  if (required.length === 0) return true
  return required.every((c) => auth.can(c))
}

/** Human-readable list of missing capabilities for the disabled tooltip.
 *  Lists only the actually-missing items from the merged required list. */
function missingCapsLabel(a: { cap?: string; allCaps?: string[] }): string {
  const required = requiredCaps(a)
  const missing = required.filter((c) => !auth.can(c))
  return missing.join('、')
}
</script>

<template>
  <div ref="wrapEl" class="rtable-wrap">
  <table>
    <thead>
      <tr>
        <th
          v-if="selectable"
          style="width:34px"
          :class="{ 'pin-l': pinCheckbox }"
          :style="pinCheckbox ? { left: '0px' } : undefined"
        >
          <Checkbox
            :checked="selected.size === rows.length && rows.length > 0"
            @click="$emit('toggleAll')"
          />
        </th>
        <th
          v-for="(col, ci) in resource.cols"
          :key="col.k"
          :class="[
            col.r === 'number' ? 'num' : '',
            { 'pin-l': col.pin === 'left', 'pin-r': col.pin === 'right' },
          ]"
          :style="pinStyle(col, ci)"
          :aria-sort="ariaSort(col)"
        >
          <button
            v-if="col.sortable"
            type="button"
            class="th-sort"
            @click="toggleSort(col)"
          >
            {{ col.l }}
            <component :is="sortIcon(col)" class="th-sort-ic" aria-hidden="true" />
          </button>
          <template v-else>{{ col.l }}</template>
        </th>
        <th
          class="num"
          :class="{ 'pin-r': pinActions }"
          :style="pinActions ? { right: '0px' } : undefined"
        >動作</th>
      </tr>
    </thead>
    <tbody>
      <tr
        v-for="entry in displayEntries"
        :key="entry.i"
      >
        <td
          v-if="selectable"
          :class="{ 'pin-l': pinCheckbox }"
          :style="pinCheckbox ? { left: '0px' } : undefined"
        >
          <Checkbox
            :checked="selected.has(entry.i)"
            @click="$emit('toggleRow', entry.i)"
          />
        </td>
        <td
          v-for="(col, ci) in resource.cols"
          :key="col.k"
          :class="[
            col.r === 'number' ? 'num' : '',
            { 'pin-l': col.pin === 'left', 'pin-r': col.pin === 'right' },
          ]"
          :style="pinStyle(col, ci)"
        >
          <template v-if="cellContent(col, entry.row).type === 'mono'">
            <span class="mono">{{ cellContent(col, entry.row).value }}</span>
          </template>
          <template v-else-if="cellContent(col, entry.row).type === 'badge'">
            <Badge
              v-if="cellContent(col, entry.row).value !== '' && cellContent(col, entry.row).value != null"
              :tone="cellContent(col, entry.row).tone"
              :label="cellContent(col, entry.row).label"
            />
            <span v-else class="muted">—</span>
          </template>
          <template v-else-if="cellContent(col, entry.row).type === 'number'">
            {{ cellContent(col, entry.row).value }}
          </template>
          <template v-else-if="cellContent(col, entry.row).type === 'datetime'">
            <span v-if="cellContent(col, entry.row).value" class="muted">{{ cellContent(col, entry.row).value }}</span>
            <span v-else class="muted">無限</span>
          </template>
          <template v-else>
            {{ cellContent(col, entry.row).value }}
          </template>
        </td>
        <td
          class="num"
          :class="{ 'pin-r': pinActions }"
          :style="pinActions ? { right: '0px' } : undefined"
        >
          <div style="display:flex;flex-wrap:wrap;justify-content:flex-end;gap:5px;align-items:center">
            <template v-for="a in inlineActions(entry.row)" :key="a.k">
              <Button
                v-if="!actionAllowed(a)"
                size="sm"
                disabled
                :title="`需要 ${missingCapsLabel(a)}`"
              >{{ a.l }}</Button>
              <Button
                v-else
                size="sm"
                :variant="a.variant ?? 'default'"
                @click="$emit('rowAction', entry.i, a.k)"
              >{{ a.l }}</Button>
            </template>
            <DropdownMenu
              v-if="menuActions(entry.row).length > 0"
              :items="menuItemsFor(entry.row)"
              align="end"
              trigger-class="rowactions-trigger"
              aria-label="更多動作"
              @select="onMenuSelect(entry.i, $event)"
            >
              <template #trigger>
                <MoreHorizontal style="width:15px;height:15px" aria-hidden="true" />
                更多
              </template>
            </DropdownMenu>
          </div>
        </td>
      </tr>
      <tr v-if="rows.length === 0">
        <td :colspan="resource.cols.length + (selectable ? 2 : 1)">
          <div class="emptybox">
            <div class="ic">
              <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.7" stroke-linecap="round" stroke-linejoin="round">
                <path d="M4 7h16v13H4z" /><path d="M4 7l4-4h8l4 4" />
              </svg>
            </div>
            <b>還沒有{{ resource.label }}</b>
            <p>建立第一筆之後就會出現在這裡。</p>
          </div>
        </td>
      </tr>
    </tbody>
  </table>
  </div>
</template>
