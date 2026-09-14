<script setup lang="ts">
// 商品 variants 子表格編輯器：每列一個可購買規格
// （名稱 / SKU / 價差 / 庫存 / 排序），支援新增與移除。
import Button from '@/components/ui/Button.vue'
import Input from '@/components/ui/Input.vue'

export interface VariantRow {
  name: string
  sku: string
  price_delta: number | string
  stock: number | string
  sort_order: number | string
}

const model = defineModel<VariantRow[]>({ default: () => [] })
defineProps<{ readOnly?: boolean; labelledby?: string }>()

function addRow() {
  model.value = [...model.value, { name: '', sku: '', price_delta: 0, stock: 0, sort_order: model.value.length }]
}
function removeRow(i: number) {
  model.value = model.value.filter((_, idx) => idx !== i)
}
</script>

<template>
  <div class="variants-editor" :aria-labelledby="labelledby" role="group">
    <table v-if="model.length > 0" class="vt">
      <thead>
        <tr>
          <th>名稱 <span class="req">*</span></th>
          <th>SKU</th>
          <th>價差</th>
          <th>庫存</th>
          <th>排序</th>
          <th v-if="!readOnly" class="vt-act"></th>
        </tr>
      </thead>
      <tbody>
        <tr v-for="(v, i) in model" :key="i">
          <td><Input v-model="v.name" :disabled="readOnly" placeholder="例如 白色" /></td>
          <td><Input v-model="v.sku" :disabled="readOnly" placeholder="選填，空白沿用主 SKU" /></td>
          <td><Input :model-value="String(v.price_delta)" type="number" :disabled="readOnly" @update:model-value="v.price_delta = $event" /></td>
          <td><Input :model-value="String(v.stock)" type="number" :disabled="readOnly" @update:model-value="v.stock = $event" /></td>
          <td><Input :model-value="String(v.sort_order)" type="number" :disabled="readOnly" @update:model-value="v.sort_order = $event" /></td>
          <td v-if="!readOnly" class="vt-act">
            <Button size="sm" variant="danger" @click="removeRow(i)">移除</Button>
          </td>
        </tr>
      </tbody>
    </table>
    <p v-else class="muted" style="margin:0 0 8px">尚無規格——此商品以單一規格販售。</p>
    <Button v-if="!readOnly" size="sm" variant="sec" @click="addRow">+ 新增規格</Button>
  </div>
</template>

<style scoped>
.variants-editor {
  grid-column: 1 / -1;
}
.vt {
  width: 100%;
  border-collapse: collapse;
  margin-bottom: 8px;
}
.vt th {
  text-align: left;
  font-size: 11px;
  color: var(--muted);
  font-weight: 500;
  padding: 4px 6px;
  border-bottom: 1px solid var(--line);
}
.vt td {
  padding: 4px 6px 4px 0;
}
.vt td:last-child {
  padding-right: 0;
}
.vt-act {
  width: 64px;
  text-align: right;
}
.req {
  color: var(--danger);
}
</style>
