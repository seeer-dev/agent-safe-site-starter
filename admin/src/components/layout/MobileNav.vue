<script setup lang="ts">
import { computed } from 'vue'
import { useRoute } from 'vue-router'
import {
  LayoutDashboard, Package, ShoppingBag, Users,
} from 'lucide-vue-next'
import { useAuthStore } from '@/stores/auth'
import { PROFILE, MOBILE_KEYS } from '@/config/profile'

const route = useRoute()
const auth = useAuthStore()

const ICON_MAP: Record<string, any> = {
  LayoutDashboard, Package, ShoppingBag, Users,
}

const items = computed(() =>
  MOBILE_KEYS
    .map((k) => PROFILE.find((x) => x.key === k))
    .filter((r): r is typeof PROFILE[number] => !!r && r.caps.every((c) => auth.can(c)))
    .slice(0, 4),
)

function hrefFor(key: string): string {
  if (key === 'dashboard') return '/'
  return `/res/${key}`
}

function isActive(key: string): boolean {
  if (key === 'dashboard') return route.name === 'dashboard'
  return route.path === `/res/${key}`
}
</script>

<template>
  <nav class="mobilenav">
    <RouterLink
      v-for="r in items"
      :key="r.key"
      :to="hrefFor(r.key)"
      :class="{ active: isActive(r.key) }"
    >
      <component :is="ICON_MAP[r.icon]" />
      {{ r.label }}
    </RouterLink>
  </nav>
</template>
