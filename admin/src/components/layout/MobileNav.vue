<script setup lang="ts">
import { computed } from 'vue'
import { useRoute } from 'vue-router'
import {
  LayoutDashboard, Package, ShoppingBag, Users,
} from 'lucide-vue-next'
import { useAuthStore } from '@/stores/auth'
import { MOBILE_KEYS, navLeaves, hrefFor, type NavLeaf } from '@/config/profile'

const route = useRoute()
const auth = useAuthStore()

const ICON_MAP: Record<string, any> = {
  LayoutDashboard, Package, ShoppingBag, Users,
}

// Mobile items resolve against the flattened, capability-gated leaf set —
// grouped profile children stay reachable on mobile.
const items = computed(() => {
  const leaves = navLeaves()
  return MOBILE_KEYS
    .map((k) => leaves.find((x) => x.key === k))
    .filter((r): r is NavLeaf => !!r && r.caps.every((c) => auth.can(c)))
    .slice(0, 4)
})

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
