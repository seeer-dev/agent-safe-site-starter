/// <reference types="vite/client" />

interface ImportMetaEnv {
  readonly AUTH_MODE: string
  readonly SUPABASE_URL: string
  readonly SUPABASE_PUBLISHABLE_KEY: string
  readonly ADMIN_API_BASE?: string
}

interface ImportMeta {
  readonly env: ImportMetaEnv
}
