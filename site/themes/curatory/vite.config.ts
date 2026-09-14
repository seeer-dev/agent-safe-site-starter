import { defineConfig } from 'vite'
import vue from '@vitejs/plugin-vue'
import tailwindcss from '@tailwindcss/vite'
import { fileURLToPath, URL } from 'node:url'

// Vue Islands build for the curatory (質選所) theme.
//
// The Go renderer produces static HTML with <div data-vue-island="Name">
// mount points. This Vite build produces:
//   islands.js          — bootstrap entry (scans + lazy-mounts islands)
//   theme-init.js       — tiny pre-paint script (dark-mode class restore;
//                         external file because CSP is script-src 'self')
//   islands-[hash].css  — Tailwind v4 output (imported by bootstrap)
//   chunks/[name]-[hash].js — per-island chunks (lazy-loaded)
//
// Output: site/themes/curatory/dist/ → copied to dist/assets/islands/
// by the Go renderer.
export default defineConfig({
  plugins: [vue(), tailwindcss()],
  resolve: {
    alias: {
      '@': fileURLToPath(new URL('.', import.meta.url)),
    },
  },
  build: {
    outDir: 'dist',
    emptyOutDir: true,
    target: 'es2020',
    cssCodeSplit: false,
    rollupOptions: {
      input: {
        islands: fileURLToPath(new URL('./islands/bootstrap.ts', import.meta.url)),
        'theme-init': fileURLToPath(new URL('./shared/theme-init.ts', import.meta.url)),
      },
      output: {
        entryFileNames: '[name].js',
        chunkFileNames: 'chunks/[name]-[hash].js',
        assetFileNames: (info) => {
          const name = info.names?.[0] ?? ''
          if (name.endsWith('.css')) return 'islands-[hash][extname]'
          return 'assets/[name]-[hash][extname]'
        },
        manualChunks(id) {
          if (id.includes('node_modules')) {
            if (id.includes('vue')) return 'vendor-vue'
            if (id.includes('lucide')) return 'vendor-icons'
            return 'vendor'
          }
        },
      },
    },
    chunkSizeWarningLimit: 200,
  },
})
