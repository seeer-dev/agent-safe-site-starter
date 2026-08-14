import { defineConfig, loadEnv } from 'vite'
import vue from '@vitejs/plugin-vue'
import { fileURLToPath, URL } from 'node:url'

// Vue Islands build for the minimal-cart theme.
//
// The Go renderer produces static HTML with <div data-vue-island="Name">
// mount points. This Vite build produces a single bootstrap entry that
// scans for those mount points and lazy-mounts the corresponding island
// component on each. Each island is a separate chunk (code-split via
// import.meta.glob), so only the islands actually on a page are loaded.
//
// Output: site/themes/minimal-cart/dist/
//   islands.js            — bootstrap entry (scans + mounts)
//   islands-[hash].css    — Tailwind output (imported by bootstrap)
//   chunks/[name]-[hash].js — per-island chunks (lazy-loaded)

// Values allowed to cross the browser build boundary. Anything not named
// here is never injected. SUPABASE_PUBLISHABLE_KEY is a public client
// identifier, not a server secret. The public theme has no admin API base:
// server-only values (DATABASE_URL, DEV_AUTH_TOKEN, R2_SECRET_ACCESS_KEY,
// RESEND_API_KEY, provider service-role keys, OAuth client secrets) must
// never appear in this list.
const BROWSER_SAFE_KEYS = [
  'AUTH_MODE',
  'SUPABASE_URL',
  'SUPABASE_PUBLISHABLE_KEY',
  'AUTH_GOOGLE_ENABLED',
  'AUTH_LINE_ENABLED',
] as const

export default defineConfig(({ mode }) => {
  const repoRoot = fileURLToPath(new URL('../../..', import.meta.url))
  const isProduction = mode === 'production'

  // Production builds run in the Cloudflare Pages build environment, which
  // also holds server-only values for the Go renderer. Repository dotenv
  // files are deliberately not read here, and only BROWSER_SAFE_KEYS are
  // injected, so a build-time secret cannot reach the public bundle.
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
        '@': fileURLToPath(new URL('.', import.meta.url)),
      },
    },
    define,
    build: {
      outDir: 'dist',
      emptyOutDir: true,
      target: 'es2020',
      cssCodeSplit: false,
      rollupOptions: {
        input: {
          islands: fileURLToPath(new URL('./islands/bootstrap.ts', import.meta.url)),
        },
        output: {
          entryFileNames: 'islands.js',
          chunkFileNames: 'chunks/[name]-[hash].js',
          assetFileNames: 'islands-[hash][extname]',
          manualChunks(id) {
            if (id.includes('node_modules')) {
              if (id.includes('@supabase')) return
              if (id.includes('vue') || id.includes('pinia')) return 'vendor-vue'
              if (id.includes('lucide')) return 'vendor-icons'
              return 'vendor'
            }
          },
        },
      },
      chunkSizeWarningLimit: 200,
    },
    server: {
      port: 5174,
      proxy: {
        '/assets/islands': {
          target: 'http://localhost:5174',
          changeOrigin: true,
          rewrite: (path) => path.replace('/assets/islands/', '/'),
        },
      },
    },
  }
})
