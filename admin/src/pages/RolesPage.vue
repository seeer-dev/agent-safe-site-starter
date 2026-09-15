<script setup lang="ts">
// 角色權限矩陣 — 唯讀檢視 admin/src/config/roles.ts 的設定。
// 這裡顯示的是「設定檔宣告的能力」，實際授權仍以伺服器為準。
import { computed } from 'vue'
import { Check, Minus } from 'lucide-vue-next'
import { ROLES } from '@/config/roles'

const roles = computed(() => Object.values(ROLES))

// 所有角色能力的聯集，依模組前綴分組排序（穩定、可預期）。
const capabilities = computed(() => {
  const all = new Set<string>()
  for (const r of roles.value) for (const c of r.caps) all.add(c)
  return [...all].sort((a, b) => a.localeCompare(b))
})

function allowed(roleCaps: string[], cap: string): boolean {
  return roleCaps.includes(cap)
}
</script>

<template>
  <div class="page">
    <div class="pagehd">
      <div>
        <h1>角色權限</h1>
        <div class="sub">各角色可執行的能力總表（唯讀；實際授權以伺服器為準）</div>
      </div>
    </div>

    <div class="panel matrix-wrap">
      <table class="matrix">
        <thead>
          <tr>
            <th class="cap-col">能力 ＼ 角色</th>
            <th v-for="r in roles" :key="r.key" class="role-col">{{ r.label }}<small class="mono">{{ r.key }}</small></th>
          </tr>
        </thead>
        <tbody>
          <tr v-for="cap in capabilities" :key="cap">
            <td class="cap-col mono">{{ cap }}</td>
            <td
              v-for="r in roles"
              :key="r.key"
              class="cell"
              :class="{ allow: allowed(r.caps, cap) }"
              :title="`${r.label} · ${cap}`"
            >
              <Check v-if="allowed(r.caps, cap)" class="ic-allow" aria-label="允許" />
              <Minus v-else class="ic-deny" aria-label="拒絕" />
            </td>
          </tr>
        </tbody>
      </table>
    </div>
  </div>
</template>

<style scoped>
.page { display: flex; flex-direction: column; gap: 14px; }
.matrix-wrap { overflow-x: auto; padding: 0; }
.matrix { width: 100%; border-collapse: collapse; font-size: 13px; }
.matrix th, .matrix td { padding: 8px 12px; border-bottom: 1px solid var(--border); text-align: center; }
.matrix .cap-col { text-align: left; font-weight: 500; position: sticky; left: 0; background: var(--card); }
.matrix .role-col { font-size: 12px; font-weight: 600; color: var(--text-2); }
.matrix .role-col small { display: block; font-weight: 400; color: var(--text-3); font-size: 11px; }
.matrix .cell .ic-allow { width: 15px; height: 15px; color: var(--green); }
.matrix .cell .ic-deny { width: 15px; height: 15px; color: var(--text-3); opacity: .45; }
.matrix .cell.allow { background: color-mix(in srgb, var(--green) 6%, transparent); }
</style>
