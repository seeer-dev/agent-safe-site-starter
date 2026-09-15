<script setup lang="ts">
// ⌘K / Ctrl+K 全域搜尋 — 索引只收「能力篩過的可見導覽項目」。
// 這是 UI 可達性提示，不是授權：真正的 gate 仍在 router beforeEach
// 與伺服器端；不可見的目的地不會出現在結果裡。
import { computed, nextTick, onMounted, ref, watch } from 'vue'
import { useRouter } from 'vue-router'
import { Search } from 'lucide-vue-next'
import { useAuthStore } from '@/stores/auth'
import { useLayoutStore } from '@/stores/layout'
import { navLeaves, hrefFor, type NavLeaf } from '@/config/profile'

const router = useRouter()
const auth = useAuthStore()
const layout = useLayoutStore()

const query = ref('')
const activeIdx = ref(0)
const inputEl = ref<HTMLInputElement | null>(null)
const listEl = ref<HTMLElement | null>(null)

interface Entry extends NavLeaf {
  group?: string
}

// Capability-filtered destinations only — an entry the principal cannot
// open never appears, so search can't leak gated routes.
const index = computed<Entry[]>(() => [
  ...navLeaves().filter((l) => l.caps.every((c) => auth.can(c))),
  { key: 'states', label: '五狀態參考', icon: 'HelpCircle', caps: [] },
])

const results = computed<Entry[]>(() => {
  const q = query.value.trim().toLowerCase()
  if (!q) return index.value
  return index.value.filter((e) => e.label.toLowerCase().includes(q) || e.key.toLowerCase().includes(q))
})

watch(results, () => { activeIdx.value = 0 })

function pathFor(key: string): string {
  if (key === 'states') return '/states'
  return hrefFor(key)
}

function choose(e: Entry) {
  layout.closePalette()
  router.push(pathFor(e.key))
}

function onKeydown(e: KeyboardEvent) {
  if (e.key === 'Escape') {
    e.preventDefault()
    layout.closePalette()
    return
  }
  if (e.key === 'ArrowDown') { e.preventDefault(); moveActive(1); return }
  if (e.key === 'ArrowUp') { e.preventDefault(); moveActive(-1); return }
  if (e.key === 'Enter') {
    e.preventDefault()
    const it = results.value[activeIdx.value]
    if (it) choose(it)
  }
}

function moveActive(dir: 1 | -1) {
  const n = results.value.length
  if (!n) return
  activeIdx.value = (activeIdx.value + dir + n) % n
  nextTick(() => {
    listEl.value
      ?.querySelector(`[data-idx="${activeIdx.value}"]`)
      ?.scrollIntoView({ block: 'nearest' })
  })
}

onMounted(() => {
  nextTick(() => inputEl.value?.focus())
})
</script>

<template>
  <div class="palette-overlay" @click.self="layout.closePalette()">
    <div class="palette" role="dialog" aria-modal="true" aria-label="全域搜尋">
      <div class="palette-input">
        <Search class="palette-ic" aria-hidden="true" />
        <input
          ref="inputEl"
          v-model="query"
          type="text"
          placeholder="搜尋頁面或資源…"
          aria-label="搜尋頁面或資源"
          @keydown="onKeydown"
        />
        <kbd>Esc</kbd>
      </div>
      <div ref="listEl" class="palette-list" role="listbox" aria-label="搜尋結果">
        <button
          v-for="(e, i) in results"
          :key="e.key"
          type="button"
          role="option"
          :data-idx="i"
          class="palette-item"
          :class="{ active: i === activeIdx }"
          :aria-selected="i === activeIdx"
          @click="choose(e)"
          @mouseenter="activeIdx = i"
        >
          <span class="palette-item-label">{{ e.label }}</span>
          <span class="palette-item-path mono">{{ pathFor(e.key) }}</span>
        </button>
        <div v-if="results.length === 0" class="palette-empty">沒有符合的頁面</div>
      </div>
      <div class="palette-foot">
        <span><kbd>↑</kbd><kbd>↓</kbd> 移動</span>
        <span><kbd>Enter</kbd> 前往</span>
        <span><kbd>Esc</kbd> 關閉</span>
      </div>
    </div>
  </div>
</template>

<style scoped>
.palette-overlay {
  position: fixed;
  inset: 0;
  z-index: 90;
  background: rgba(0, 0, 0, 0.35);
  display: flex;
  align-items: flex-start;
  justify-content: center;
  padding-top: 14vh;
}
.palette {
  width: min(520px, calc(100vw - 32px));
  background: var(--card);
  border: 1px solid var(--border);
  border-radius: 12px;
  box-shadow: 0 18px 50px rgba(0, 0, 0, 0.25);
  overflow: hidden;
}
.palette-input {
  display: flex;
  align-items: center;
  gap: 10px;
  padding: 12px 14px;
  border-bottom: 1px solid var(--border);
}
.palette-ic { width: 15px; height: 15px; color: var(--text-3); flex: none; }
.palette-input input {
  flex: 1;
  border: none;
  background: none;
  font: inherit;
  font-size: 14px;
  color: var(--text);
  outline: none;
}
.palette-list {
  max-height: 320px;
  overflow-y: auto;
  padding: 6px;
}
.palette-item {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 10px;
  width: 100%;
  padding: 8px 10px;
  border: none;
  background: none;
  border-radius: 8px;
  font: inherit;
  font-size: 13.5px;
  color: var(--text-2);
  cursor: pointer;
  text-align: left;
}
.palette-item.active { background: var(--brand-50); color: var(--text); }
.palette-item-path { font-size: 11.5px; color: var(--text-3); }
.palette-empty { padding: 20px; text-align: center; color: var(--text-3); font-size: 13px; }
.palette-foot {
  display: flex;
  gap: 14px;
  padding: 8px 14px;
  border-top: 1px solid var(--border);
  font-size: 11.5px;
  color: var(--text-3);
}
.palette kbd {
  padding: 1px 5px;
  border: 1px solid var(--border);
  border-bottom-width: 2px;
  border-radius: 4px;
  background: var(--surface-2);
  font-family: inherit;
  font-size: 10.5px;
}
</style>
