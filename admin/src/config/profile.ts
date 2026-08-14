import type { RouteDef, SectionKey } from '@/lib/types';

// profile routes：section / capabilityGate 逐欄對應真實 profile
export const PROFILE: RouteDef[] = [
  { key: 'dashboard', label: '總覽', icon: 'LayoutDashboard', section: 'primary', component: 'MinimalCartDashboardPage', caps: ['twcommerce.read'] },
  { key: 'minimal-cart-products', label: '商品', icon: 'Package', section: 'primary', component: 'ResourceListPage', caps: ['twcommerce.read'] },
  { key: 'minimal-cart-orders', label: '訂單', icon: 'ShoppingBag', section: 'primary', component: 'ResourceListPage', caps: ['twcommerce.read'] },
  { key: 'minimal-cart-members', label: '會員', icon: 'Users', section: 'primary', component: 'ResourceListPage', caps: ['twcommerce.read'] },
  { key: 'minimal-cart-promos', label: '優惠', icon: 'TicketPercent', section: 'primary', component: 'ResourceListPage', caps: ['twcommerce.read'] },
  { key: 'minimal-cart-content', label: '前台內容', icon: 'FileText', section: 'primary', component: 'ResourceListPage', caps: ['content.read'] },
  { key: 'tw-commerce.methods', label: '付款方式', icon: 'CreditCard', section: 'secondary', component: 'ResourceListPage', caps: ['twcommerce.read'] },
  { key: 'minimal-cart-shipping', label: '配送方式', icon: 'CreditCard', section: 'secondary', component: 'ResourceListPage', caps: ['twcommerce.read'] },
  { key: 'staff', label: '人員', icon: 'UserCog', section: 'settings', component: 'ResourceListPage', caps: ['staff.read'] },
];

export const SECTION_LABEL: Record<SectionKey, string> = {
  primary: '營運',
  secondary: '交易設定',
  settings: '系統',
};

export const MOBILE_KEYS: string[] = [
  'dashboard',
  'minimal-cart-products',
  'minimal-cart-orders',
  'minimal-cart-members',
];
