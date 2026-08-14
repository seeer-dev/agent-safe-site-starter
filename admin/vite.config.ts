import { defineConfig, loadEnv } from 'vite'
import vue from '@vitejs/plugin-vue'
import { fileURLToPath, URL } from 'node:url'

// Values allowed to cross the browser build boundary. Anything not named
// here is never injected. SUPABASE_PUBLISHABLE_KEY is a public client
// identifier, not a server secret. Server-only values (DATABASE_URL,
// DEV_AUTH_TOKEN, R2_SECRET_ACCESS_KEY, RESEND_API_KEY, provider
// service-role keys, OAuth client secrets) must never appear in this list.
const BROWSER_SAFE_KEYS = [
  'AUTH_MODE',
  'SUPABASE_URL',
  'SUPABASE_PUBLISHABLE_KEY',
  'AUTH_GOOGLE_ENABLED',
  'AUTH_LINE_ENABLED',
  'ADMIN_API_BASE',
] as const

export default defineConfig(({ mode }) => {
  const repoRoot = fileURLToPath(new URL('..', import.meta.url))
  const isProduction = mode === 'production'

  // Production builds run on Cloudflare Pages or another provider that
  // supplies configuration through the build process environment. Repository
  // dotenv files are deliberately not read: a developer's local file must
  // never be able to reach a production bundle. In development, loadEnv
  // resolves the local profiles, with `.env.development.local` taking
  // precedence over the legacy `.env`.
  const env = isProduction ? process.env : loadEnv(mode, repoRoot, '')

  const define = Object.fromEntries(
    BROWSER_SAFE_KEYS.map((key) => [
      `import.meta.env.${key}`,
      JSON.stringify(env[key] ?? ''),
    ]),
  )

  return {
    plugins: [vue()],
    resolve: {
      alias: {
        '@': fileURLToPath(new URL('./src', import.meta.url)),
      },
    },
    define,
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
