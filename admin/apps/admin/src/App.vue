<script setup lang="ts">
import { computed } from 'vue'
import { RouterView, useRoute, useRouter } from 'vue-router'
import { AdminShell } from '@sitecore/admin-shell'
import type { ShellNavigationModel, ShellNavItem } from '@sitecore/admin-shell'
import {
  LayoutDashboard,
  ShoppingCart,
  Package,
  CircleDollarSign,
  Boxes,
  Megaphone,
  Mail,
  Users,
  CalendarPlus,
  FileText,
  Settings,
  UserCircle,
} from 'lucide-vue-next'

const route = useRoute()
const router = useRouter()

function isCurrent(href: string): boolean {
  if (href === '/') return route.path === '/'
  return route.path === href || route.path.startsWith(href + '/')
}

const commissionsChildren = computed<ShellNavItem[]>(() => [
  { key: 'commissions-lines', label: '分潤明細', href: '/commissions', icon: Boxes, current: isCurrent('/commissions') && !isCurrent('/payouts') },
  { key: 'commissions-payouts', label: '撥款紀錄', href: '/payouts', icon: CircleDollarSign, current: isCurrent('/payouts') },
])

const navigation = computed<ShellNavigationModel>(() => ({
  groups: [
    {
      key: 'operations',
      label: '營運',
      items: [
        { key: 'dashboard', label: '今日待辦', href: '/', icon: LayoutDashboard, current: isCurrent('/') },
        { key: 'orders', label: '訂單', href: '/orders', icon: ShoppingCart, current: isCurrent('/orders') },
        { key: 'campaigns-new', label: '新增檔期', href: '/campaigns/new', icon: CalendarPlus, current: isCurrent('/campaigns/new') },
        { key: 'products-new', label: '商品建立', href: '/products/new', icon: Package, current: isCurrent('/products/new') },
        { key: 'influencers', label: '網紅', href: '/influencers', icon: UserCircle, current: isCurrent('/influencers') },
        {
          key: 'commissions',
          label: '分潤與撥款',
          href: '/commissions',
          icon: CircleDollarSign,
          current: isCurrent('/commissions') || isCurrent('/payouts'),
          branchOpen: isCurrent('/commissions') || isCurrent('/payouts'),
          children: commissionsChildren.value,
        },
      ],
    },
    {
      key: 'engagement',
      label: '溝通',
      items: [
        { key: 'announcement', label: '公告', href: '/announcement', icon: Megaphone, current: isCurrent('/announcement') },
        { key: 'messaging', label: '通知', href: '/messaging', icon: Mail, current: isCurrent('/messaging') },
        { key: 'content', label: '內容', href: '/content', icon: FileText, current: isCurrent('/content') },
      ],
    },
    {
      key: 'system',
      label: '系統',
      items: [
        { key: 'staff', label: '員工管理', href: '/system/staff', icon: Users, current: isCurrent('/system/staff') },
        { key: 'settings', label: '賣點設定', href: '/settings', icon: Settings, current: isCurrent('/settings') },
      ],
    },
  ],
  footer: [],
  mobile: [
    { key: 'dashboard', label: '今日待辦', href: '/', icon: LayoutDashboard, current: isCurrent('/') },
    { key: 'orders', label: '訂單', href: '/orders', icon: ShoppingCart, current: isCurrent('/orders') },
    { key: 'commissions', label: '分潤', href: '/commissions', icon: CircleDollarSign, current: isCurrent('/commissions') },
    { key: 'payouts', label: '撥款', href: '/payouts', icon: Boxes, current: isCurrent('/payouts') },
  ],
}))

const breadcrumb = computed(() => {
  if (isCurrent('/')) return ['後台', '今日待辦']
  if (isCurrent('/orders')) return ['後台', '訂單']
  if (isCurrent('/commissions')) return ['後台', '分潤與撥款', '分潤明細']
  if (isCurrent('/payouts')) return ['後台', '分潤與撥款', '撥款紀錄']
  if (isCurrent('/campaigns/new')) return ['後台', '新增檔期']
  if (isCurrent('/products/new')) return ['後台', '商品建立']
  if (isCurrent('/influencers/new')) return ['後台', '網紅', '新增網紅']
  if (isCurrent('/influencers')) return ['後台', '網紅']
  if (isCurrent('/announcement')) return ['後台', '公告']
  if (isCurrent('/messaging')) return ['後台', '通知']
  if (isCurrent('/content')) return ['後台', '內容']
  if (isCurrent('/system/staff')) return ['後台', '系統', '員工管理']
  if (isCurrent('/settings')) return ['後台', '系統', '賣點設定']
  return ['後台']
})

function onNavigate(href: string) {
  router.push(href)
}
</script>

<template>
  <div data-surface-id="admin" data-candidate="vue">
    <AdminShell
      :navigation="navigation"
      :breadcrumb="breadcrumb"
      identity-label="陳怡君 · Owner"
      search-placeholder="搜尋訂單、商品與動作…"
      :brand="{ logo: '網', label: '網紅分潤後台', subtitle: 'admin app' }"
      @navigate="onNavigate"
    >
      <RouterView />
    </AdminShell>
  </div>
</template>
