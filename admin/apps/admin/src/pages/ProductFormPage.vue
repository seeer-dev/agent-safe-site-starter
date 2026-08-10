<script setup lang="ts">
import { ref } from 'vue'
import { useRouter } from 'vue-router'
import { PageHeader, Panel, TableViewport, Table, TableHeader, TableRow, TableHead, TableBody, TableCell, Button } from '@sitecore/admin-ui'
import { ArrowLeft, Plus, Check, Image as ImageIcon } from 'lucide-vue-next'

const router = useRouter()

const productName = ref('無袖小花 T-Shirt')
const sku = ref('TS-SU-WM')
const category = ref('上衣/T-Shirt')
const description = ref('無袖清爽小花紋，100% 棉，透氣舒適。')

const variants = [
  { id: 'v1', spec: '白 / M', sku: 'TS-SU-WM', price: '$1,280', stock: '24' },
  { id: 'v2', spec: '白 / L', sku: 'TS-SU-WL', price: '$1,280', stock: '18' },
  { id: 'v3', spec: '黑 / M', sku: 'TS-SU-BM', price: '$1,280', stock: '0' },
]

const campaigns = [
  { id: 'c1', label: '無袖小花', checked: true },
  { id: 'c2', label: '夏季新品活動', checked: false },
  { id: 'c3', label: '年度特賣', checked: false },
]

const listingEnabled = ref(true)
</script>

<template>
  <div class="content">
    <a class="back-link inline-flex items-center gap-1.5 text-text-primary text-fs-md mb-2.5 cursor-pointer" @click="router.push('/orders')">
      <ArrowLeft :size="15" />
      商品
    </a>

    <PageHeader title="新增商品" subtitle="無袖小花 T-Shirt" />

    <div class="grid lg:grid-cols-[1fr_280px] gap-4 max-lg:block mt-3.5">
      <div class="flex flex-col gap-4">
        <!-- Image upload -->
        <Panel class="overflow-hidden">
          <div class="panel-hd px-4 py-3 border-b border-border text-fs-lg font-semibold text-text-primary">商品圖片</div>
          <div class="p-4">
            <div class="grid grid-cols-3 sm:grid-cols-4 gap-2.5">
              <div class="prod-img-main aspect-square rounded-card bg-gradient-to-br from-brand-50 to-surface-200 border border-border grid place-items-center relative overflow-hidden cursor-pointer">
                <ImageIcon :size="28" class="text-text-muted" />
                <span class="absolute bottom-1 left-1 text-fs-xs font-medium text-text-muted bg-surface/80 px-1.5 rounded">主圖</span>
              </div>
              <div v-for="n in 3" :key="n" class="aspect-square rounded-card border-1.5 border-dashed border-border-strong grid place-items-center cursor-pointer">
                <Plus :size="22" class="text-text-muted" />
              </div>
            </div>
            <div class="text-text-muted text-fs-base mt-3 flex items-center gap-1.5">
              建議尺寸 800×800px · JPG/PNG/WebP · 每張最大 5MB · 最多 8 張
            </div>
          </div>
        </Panel>

        <!-- Basic info -->
        <Panel class="overflow-hidden">
          <div class="panel-hd px-4 py-3 border-b border-border text-fs-lg font-semibold text-text-primary">基本資料</div>
          <div class="p-4 flex flex-col gap-3.5">
            <div>
              <label class="block text-text-secondary text-fs-base font-semibold mb-1.5">商品名稱</label>
              <input v-model="productName" class="w-full px-3 py-2 rounded-input bg-surface border border-border text-fs-md text-text-primary" />
            </div>
            <div class="grid grid-cols-2 gap-3">
              <div>
                <label class="block text-text-secondary text-fs-base font-semibold mb-1.5">SKU</label>
                <input v-model="sku" class="w-full px-3 py-2 rounded-input bg-surface border border-border text-fs-md text-text-primary" />
              </div>
              <div>
                <label class="block text-text-secondary text-fs-base font-semibold mb-1.5">分類</label>
                <select v-model="category" class="w-full px-3 py-2 rounded-input bg-surface border border-border text-fs-md text-text-primary cursor-pointer">
                  <option>上衣/T-Shirt</option>
                  <option>上衣/外套</option>
                  <option>裙子</option>
                  <option>配件</option>
                </select>
              </div>
            </div>
            <div>
              <label class="block text-text-secondary text-fs-base font-semibold mb-1.5">商品描述</label>
              <textarea v-model="description" class="w-full min-h-[60px] px-3 py-2 rounded-input bg-surface border border-border text-fs-md text-text-primary resize-y" />
            </div>
          </div>
        </Panel>

        <!-- Variants -->
        <Panel class="overflow-hidden">
          <div class="panel-hd px-4 py-3 border-b border-border text-fs-lg font-semibold text-text-primary">規格與庫存</div>
          <TableViewport>
            <Table>
              <TableHeader>
                <TableRow>
                  <TableHead>規格</TableHead>
                  <TableHead>SKU</TableHead>
                  <TableHead align="end">價格</TableHead>
                  <TableHead align="end">庫存</TableHead>
                </TableRow>
              </TableHeader>
              <TableBody>
                <TableRow v-for="v in variants" :key="v.id">
                  <TableCell>{{ v.spec }}</TableCell>
                  <TableCell class="text-text-muted">{{ v.sku }}</TableCell>
                  <TableCell align="end" class="cell-num">{{ v.price }}</TableCell>
                  <TableCell align="end" class="cell-num">{{ v.stock }}</TableCell>
                </TableRow>
              </TableBody>
            </Table>
          </TableViewport>
          <div class="px-4 py-2.5 border-t border-border">
            <button class="text-fs-base-plus font-medium text-brand-700 cursor-pointer">+ 新增規格</button>
          </div>
        </Panel>

        <!-- Campaign binding -->
        <Panel class="overflow-hidden">
          <div class="panel-hd px-4 py-3 border-b border-border text-fs-lg font-semibold text-text-primary">檔期綁定</div>
          <div class="p-4 flex flex-col gap-2.5">
            <label v-for="c in campaigns" :key="c.id" class="flex items-center gap-2.5 cursor-pointer">
              <input type="checkbox" :checked="c.checked" class="w-4 h-4 accent-brand-600" />
              {{ c.label }}
            </label>
          </div>
        </Panel>
      </div>

      <!-- Side rail -->
      <aside class="flex flex-col gap-4 max-lg:mt-4">
        <Panel class="overflow-hidden">
          <div class="panel-hd px-4 py-3 border-b border-border text-fs-lg font-semibold text-text-primary">上架狀態</div>
          <div class="p-4 flex flex-col gap-3">
            <label class="flex items-center justify-between cursor-pointer">
              <span class="text-fs-md text-text-secondary">上架販售</span>
              <span class="inline-flex w-9 h-5 rounded-full items-center px-0.5 transition-colors" :class="listingEnabled ? 'bg-brand-600 justify-end' : 'bg-surface-200 justify-start'">
                <span class="w-4 h-4 rounded-full bg-white shadow" />
              </span>
            </label>
            <div class="text-text-muted text-fs-base">下架時買家無法瀏覽，已下訂單不受影響。</div>
          </div>
        </Panel>
        <Panel class="overflow-hidden">
          <div class="panel-hd px-4 py-3 border-b border-border text-fs-lg font-semibold text-text-primary">參與檔期</div>
          <div class="p-4 flex flex-col gap-2 text-fs-base-plus">
            <div class="flex items-center justify-between">
              <span class="text-text-primary">無袖小花</span>
              <span class="text-success-fg font-medium">可檔期</span>
            </div>
            <div class="text-text-muted">比例 15% + 固定 $80</div>
          </div>
        </Panel>
      </aside>
    </div>

    <!-- Save bar -->
    <div class="savebar sticky bottom-0 mt-4 flex items-center gap-2 px-4 py-3 bg-surface border-t border-border">
      <span class="text-fs-base-plus text-text-muted"><b class="text-warning-fg">●</b> 尚有未儲存的變更</span>
      <div class="savebar-spacer flex-1" />
      <Button variant="secondary">預覽</Button>
        <Button variant="primary">
        <template #default>
          <span class="inline-flex items-center gap-1">
            <Check :size="15" />
            儲存
          </span>
        </template>
      </Button>
    </div>
  </div>
</template>
