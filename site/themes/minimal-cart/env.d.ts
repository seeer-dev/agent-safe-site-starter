/// <reference types="vite/client" />

interface ImportMetaEnv {
  readonly AUTH_MODE: string
  readonly SUPABASE_URL: string
  readonly SUPABASE_PUBLISHABLE_KEY: string
  readonly AUTH_GOOGLE_ENABLED: string
  readonly AUTH_LINE_ENABLED: string
}

interface ImportMeta {
  readonly env: ImportMetaEnv
}

declare module '*.vue' {
  import type { DefineComponent } from 'vue'
  const component: DefineComponent<{}, {}, any>
  export default component
}
