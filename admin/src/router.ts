import { createRouter, createWebHistory, type RouteRecordRaw } from 'vue-router'
import { useAuthStore, type AuthStatus } from '@/stores/auth'

const routes: RouteRecordRaw[] = [
  {
    path: '/',
    name: 'dashboard',
    component: () => import('@/pages/DashboardPage.vue'),
    meta: { caps: ['twcommerce.read'] },
  },
  {
    path: '/res/:resourceKey',
    name: 'resource',
    component: () => import('@/pages/ResourceListPage.vue'),
    meta: { caps: [] },
  },
  {
    path: '/states',
    name: 'states',
    component: () => import('@/pages/StatesPage.vue'),
  },
  {
    path: '/:pathMatch(.*)*',
    redirect: '/',
  },
]

export const router = createRouter({
  history: createWebHistory(),
  routes,
})

export type CapabilityGuardAuth = {
  status: AuthStatus
  can: (cap: string) => boolean
}

export function authCapabilityGuard(
  to: { path: string; name?: string | symbol | null | undefined; meta: { caps?: string[] } },
  auth: CapabilityGuardAuth,
): true | string {
  const required = to.meta.caps ?? []
  if (required.length === 0) return true
  // Capability hiding is only meaningful after Go has confirmed staff.
  // During connecting/unverified/failed/unavailable the App gate already
  // hides AdminShell; do not redirect and never self-redirect `/` -> `/`.
  if (auth.status !== 'verified') return true
  if (required.every((c) => auth.can(c))) return true
  if (to.path === '/' || to.name === 'dashboard') return true
  return '/'
}

router.beforeEach((to) => {
  return authCapabilityGuard(
    { path: to.path, name: to.name, meta: { caps: to.meta.caps as string[] | undefined } },
    useAuthStore(),
  )
})
