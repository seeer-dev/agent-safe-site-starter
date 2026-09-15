import type { Capability, RouteDef } from '@/lib/types';

// Grouped navigation profile, ported from the reference mockup's IA and
// rearranged per owner review: high-frequency business leaves (總覽,
// 訂單, 商品, 分類) sit flat on top, the 顧客 group follows, a labeled
// divider ("通用後台") separates universal base groups. Groups are a data
// shape — the sidebar activates the existing children machinery (inline
// expand in expanded mode, hover flyout in collapsed mode). Capability
// gating is recursive: a gated child drops out, and a group with zero
// visible children drops out entirely. Groups carry caps:[] so a mixed-
// capability group (e.g. 商店 holds both twcommerce.read and content.read
// children) stays reachable by any principal that can see one child.
export const PROFILE: RouteDef[] = [
  { key: 'dashboard', label: '總覽', icon: 'LayoutDashboard', component: 'MinimalCartDashboardPage', caps: ['twcommerce.read'] },
  { key: 'minimal-cart-orders', label: '訂單', icon: 'ShoppingBag', caps: ['twcommerce.read'] },
  { key: 'minimal-cart-products', label: '商品', icon: 'Package', caps: ['twcommerce.read'] },
  { key: 'categories', label: '分類', icon: 'FolderTree', caps: ['twcommerce.read'] },
  {
    key: 'grp-customer', label: '顧客', icon: 'Users', caps: [],
    children: [
      { key: 'minimal-cart-members', label: '會員', icon: 'Users', caps: ['twcommerce.read'] },
      { key: 'comments', label: '評論', icon: 'MessageSquareText', caps: ['twcommerce.read'] },
    ],
  },
  {
    key: 'grp-content', label: '內容', icon: 'Newspaper', caps: [], dividerBefore: '通用後台',
    children: [
      { key: 'articles', label: '公告文章', icon: 'Newspaper', caps: ['content.publish'] },
      { key: 'minimal-cart-content', label: '前台內容', icon: 'FileText', caps: ['content.read'] },
    ],
  },
  {
    key: 'grp-notify', label: '通知', icon: 'Mail', caps: [], dividerBefore: '通用後台',
    children: [
      { key: 'notification-templates', label: '通知模板', icon: 'Mail', caps: ['twcommerce.read'] },
      { key: 'notification-logs', label: '通知日誌', icon: 'MailCheck', caps: ['twcommerce.read'] },
    ],
  },
  {
    key: 'grp-store', label: '商店', icon: 'Store', caps: [], dividerBefore: '通用後台',
    children: [
      { key: 'minimal-cart-promos', label: '優惠', icon: 'TicketPercent', caps: ['twcommerce.read'] },
      { key: 'store-settings', label: '商店設定', icon: 'Store', caps: ['content.read'] },
      { key: 'tw-commerce.methods', label: '付款方式', icon: 'CreditCard', caps: ['twcommerce.read'] },
      { key: 'minimal-cart-shipping', label: '配送方式', icon: 'Truck', caps: ['twcommerce.read'] },
    ],
  },
  {
    key: 'grp-system', label: '系統', icon: 'Settings', caps: [], dividerBefore: '通用後台',
    children: [
      { key: 'staff', label: '人員', icon: 'UserCog', caps: ['staff.read'] },
      { key: 'roles', label: '角色權限', icon: 'ShieldCheck', caps: ['staff.read'] },
    ],
  },
];

/** Profile key -> route path. Shared by Sidebar, MobileNav, and the
 *  search palette so every surface resolves destinations identically. */
export function hrefFor(key: string): string {
  if (key === 'dashboard') return '/';
  if (key === 'store-settings') return '/settings';
  if (key === 'roles') return '/roles';
  return `/res/${key}`;
}

export const MOBILE_KEYS: string[] = [
  'dashboard',
  'minimal-cart-products',
  'minimal-cart-orders',
  'minimal-cart-members',
];

export interface NavLeaf {
  key: string;
  label: string;
  icon: string;
  caps: Capability[];
}

/** Flattened routable leaves across the grouped profile — shared by
 *  MobileNav and the search palette so every surface derives from the
 *  same capability-filtered navigation set. */
export function navLeaves(): NavLeaf[] {
  const out: NavLeaf[] = [];
  for (const r of PROFILE) {
    if (r.children?.length) {
      for (const c of r.children) {
        out.push({ key: c.key, label: c.label, icon: c.icon ?? r.icon, caps: c.caps });
      }
    } else {
      out.push({ key: r.key, label: r.label, icon: r.icon, caps: r.caps });
    }
  }
  return out;
}
