import { defineConfig, loadEnv } from 'vite'
import vue from '@vitejs/plugin-vue'
import { fileURLToPath, URL } from 'node:url'

export default defineConfig(({ mode }) => {
  const repoRoot = fileURLToPath(new URL('..', import.meta.url))
  const env = loadEnv(mode, repoRoot, '')

  return {
    plugins: [vue()],
    resolve: {
      alias: {
        '@': fileURLToPath(new URL('./src', import.meta.url)),
      },
    },
    define: {
      'import.meta.env.AUTH_MODE': JSON.stringify(env.AUTH_MODE ?? ''),
      'import.meta.env.SUPABASE_URL': JSON.stringify(env.SUPABASE_URL ?? ''),
      'import.meta.env.SUPABASE_PUBLISHABLE_KEY': JSON.stringify(env.SUPABASE_PUBLISHABLE_KEY ?? ''),
      'import.meta.env.AUTH_GOOGLE_ENABLED': JSON.stringify(env.AUTH_GOOGLE_ENABLED ?? ''),
      'import.meta.env.AUTH_LINE_ENABLED': JSON.stringify(env.AUTH_LINE_ENABLED ?? ''),
      'import.meta.env.ADMIN_API_BASE': JSON.stringify(env.ADMIN_API_BASE ?? ''),
    },
    server: {
      port: 5174,
      proxy: {
        '/api': 'http://localhost:8080',
      },
    },
    build: {
      outDir: 'dist',
      emptyOutDir: true,
      target: 'es2020',
    },
  }
})
