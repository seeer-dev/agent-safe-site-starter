<script setup lang="ts">
import { onMounted } from 'vue'
import { useThemeStore } from '@/stores/theme'
import { useAuthStore } from '@/stores/auth'
import { useLayoutStore } from '@/stores/layout'
import AdminShell from '@/components/layout/AdminShell.vue'
import AuthGate from '@/components/auth/AuthGate.vue'

const theme = useThemeStore()
const auth = useAuthStore()
const layout = useLayoutStore()

void auth.initialize()
onMounted(() => {
  theme.init()
  layout.init()
})
</script>

<template>
  <AdminShell v-if="auth.isAuthenticated">
    <RouterView />
  </AdminShell>
  <AuthGate v-else />
</template>
