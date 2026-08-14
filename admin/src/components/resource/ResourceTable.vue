<script setup lang="ts">
import { computed } from 'vue'
import { TONE, LABEL } from '@/config/tones'
import { formatNT, evalShowWhen } from '@/lib/utils'
import type { Col, ResourceDef } from '@/lib/types'
import Badge from '@/components/ui/Badge.vue'
import Checkbox from '@/components/ui/Checkbox.vue'
import Button from '@/components/ui/Button.vue'
import { useAuthStore } from '@/stores/auth'

const props = defineProps<{
  resource: ResourceDef
  rows: Record<string, any>[]
  selected: Set<number>
}>()

const emit = defineEmits<{
  toggleRow: [i: number]
  toggleAll: []
  rowAction: [i: number, actionKey: string]
}>()

const auth = useAuthStore()

const selectable = computed(() => (props.resource.bulkActions ?? []).filter((a) => auth.can(a.cap)).length > 0)

function toneFor(v: unknown) {
  return TONE[String(v)] ?? 'neutral'
}

function labelFor(v: unknown) {
  const key = String(v)
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
  if (col.r === 'badge') return { type: 'badge' as const, value: v, tone: toneFor(v), label: labelFor(v) }
  if (col.r === 'number') {
    if (v === null || v === undefined) return { type: 'number' as const, value: '—' }
    const isMoney = col.k === 'price' || col.k === 'total' || col.k === 'total_spent'
    return { type: 'number' as const, value: isMoney ? formatNT(v) : String(v) }
  }
  if (col.r === 'datetime') return { type: 'datetime' as const, value: v }
  return { type: 'text' as const, value: absentDash(v) }
}

function visibleActions(row: Record<string, any>) {
  return props.resource.rowActions.filter((a) => evalShowWhen(a.showWhen, row))
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
  <table>
    <thead>
      <tr>
        <th v-if="selectable" style="width:34px">
          <Checkbox
            :checked="selected.size === rows.length && rows.length > 0"
            @click="$emit('toggleAll')"
          />
        </th>
        <th
          v-for="col in resource.cols"
          :key="col.k"
          :class="col.r === 'number' ? 'num' : ''"
        >
          {{ col.l }}
        </th>
        <th class="num">動作</th>
      </tr>
    </thead>
    <tbody>
      <tr
        v-for="(row, i) in rows"
        :key="i"
      >
        <td v-if="selectable">
          <Checkbox
            :checked="selected.has(i)"
            @click="$emit('toggleRow', i)"
          />
        </td>
        <td
          v-for="col in resource.cols"
          :key="col.k"
          :class="col.r === 'number' ? 'num' : ''"
        >
          <template v-if="cellContent(col, row).type === 'mono'">
            <span class="mono">{{ cellContent(col, row).value }}</span>
          </template>
          <template v-else-if="cellContent(col, row).type === 'badge'">
            <Badge
              v-if="cellContent(col, row).value !== '' && cellContent(col, row).value != null"
              :tone="cellContent(col, row).tone"
              :label="cellContent(col, row).label"
            />
            <span v-else class="muted">—</span>
          </template>
          <template v-else-if="cellContent(col, row).type === 'number'">
            {{ cellContent(col, row).value }}
          </template>
          <template v-else-if="cellContent(col, row).type === 'datetime'">
            <span v-if="cellContent(col, row).value" class="muted">{{ cellContent(col, row).value }}</span>
            <span v-else class="muted">無限</span>
          </template>
          <template v-else>
            {{ cellContent(col, row).value }}
          </template>
        </td>
        <td class="num">
          <div style="display:flex;flex-wrap:wrap;justify-content:flex-end;gap:5px">
            <template v-for="a in visibleActions(row)" :key="a.k">
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
                @click="$emit('rowAction', i, a.k)"
              >{{ a.l }}</Button>
            </template>
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
</template>
