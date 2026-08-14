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

export default defineConfig(({ mode }) => {
  const repoRoot = fileURLToPath(new URL('../../..', import.meta.url))
  const env = loadEnv(mode, repoRoot, '')

  return {
    plugins: [vue()],
    resolve: {
      alias: {
        '@': fileURLToPath(new URL('.', import.meta.url)),
      },
    },
    define: {
      'import.meta.env.AUTH_MODE': JSON.stringify(env.AUTH_MODE ?? ''),
      'import.meta.env.SUPABASE_URL': JSON.stringify(env.SUPABASE_URL ?? ''),
      'import.meta.env.SUPABASE_PUBLISHABLE_KEY': JSON.stringify(env.SUPABASE_PUBLISHABLE_KEY ?? ''),
      'import.meta.env.AUTH_GOOGLE_ENABLED': JSON.stringify(env.AUTH_GOOGLE_ENABLED ?? ''),
      'import.meta.env.AUTH_LINE_ENABLED': JSON.stringify(env.AUTH_LINE_ENABLED ?? ''),
    },
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
