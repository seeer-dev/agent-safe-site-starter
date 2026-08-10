import { createRouter, createWebHistory } from 'vue-router'
import DashboardPage from '../pages/DashboardPage.vue'
import OrdersPage from '../domains/orders/ui/OrdersPage.vue'
import OrderDetailPage from '../domains/order-detail/ui/OrderDetailPage.vue'
import CommissionsPayoutsPage from '../domains/commissions/ui/CommissionsPayoutsPage.vue'
import MessagingPage from '../domains/messaging/ui/MessagingPage.vue'
import ContentListPage from '../domains/content/ui/ContentListPage.vue'
import InfluencersListPage from '../domains/influencers/ui/InfluencersListPage.vue'
import InfluencerDetailPage from '../domains/influencers/ui/InfluencerDetailPage.vue'
import InfluencerCreatePage from '../domains/influencers/ui/InfluencerCreatePage.vue'
import AnnouncementPage from '../pages/AnnouncementPage.vue'
import StaffPage from '../pages/StaffPage.vue'
import SettingsPage from '../pages/SettingsPage.vue'
import CampaignNewPage from '../pages/CampaignNewPage.vue'
import ProductFormPage from '../pages/ProductFormPage.vue'
import ForbiddenPage from '../pages/ForbiddenPage.vue'
import { useSessionStore } from '@sitecore/admin-auth'

export const router = createRouter({ history: createWebHistory('/admin/'), routes: [
  { path: '/', component: DashboardPage },
  { path: '/orders', component: OrdersPage, meta: { capability: 'orders.read' } },
  { path: '/orders/:orderId', component: OrderDetailPage, props: true, meta: { capability: 'orders.read' } },
  { path: '/orders/:orderId/review-history', component: OrderDetailPage, props: true, meta: { capability: 'orders.read' } },
  { path: '/orders/:orderId/commission', component: OrderDetailPage, props: true, meta: { capability: 'orders.read' } },
  { path: '/orders/:orderId/fulfillment-documents', component: OrderDetailPage, props: true, meta: { capability: 'orders.read' } },
  { path: '/commissions', component: CommissionsPayoutsPage, props: { defaultTab: 'commissions' }, meta: { capability: 'commissions.read' } },
  { path: '/payouts', component: CommissionsPayoutsPage, props: { defaultTab: 'payouts' }, meta: { capability: 'commissions.read' } },
  { path: '/messaging', component: MessagingPage, meta: { capability: 'messaging.read' } },
  { path: '/messaging/templates', component: MessagingPage, meta: { capability: 'messaging.read' } },
  { path: '/messaging/attempts', component: MessagingPage, meta: { capability: 'messaging.read' } },
  { path: '/content', component: ContentListPage, meta: { capability: 'content.read' } },
  { path: '/influencers', component: InfluencersListPage, meta: { capability: 'influencers.read' } },
  { path: '/influencers/new', component: InfluencerCreatePage, meta: { capability: 'influencers.write' } },
  { path: '/influencers/:influencerId', component: InfluencerDetailPage, props: true, meta: { capability: 'influencers.read' } },
  { path: '/announcement', component: AnnouncementPage, meta: { capability: 'announcement.write' } },
  { path: '/system/staff', component: StaffPage, meta: { capability: 'system.read' } },
  { path: '/settings', component: SettingsPage, meta: { capability: 'settings.read' } },
  { path: '/campaigns/new', component: CampaignNewPage, meta: { capability: 'campaigns.write' } },
  { path: '/products/new', component: ProductFormPage, meta: { capability: 'products.write' } },
  { path: '/forbidden', component: ForbiddenPage },
] })
router.beforeEach(to => { const required = to.meta['capability'] as string | undefined; if (!required) return true; const session = useSessionStore(); return session.has(required) ? true : '/forbidden' })
