/* ============================================================
 * Admin Theme — Token Contract + Registry
 *
 * Framework-neutral. No React/Vue provider here.
 * Theme packs (admin-theme-mint / nippon / tailwind) import the
 * types and helpers from this module to declare their metadata.
 * Runtime application is done via DOM data attributes
 * (data-theme-style / data-brand / .dark / data-admin-density)
 * that the theme pack CSS files respond to.
 * ============================================================ */

export const ADMIN_THEME_COMPATIBILITY_VERSION = 1 as const

export const REQUIRED_ADMIN_THEME_TOKENS = [
  '--admin-canvas',
  '--admin-surface',
  '--admin-surface-subtle',
  '--admin-text-primary',
  '--admin-text-secondary',
  '--admin-text-muted',
  '--admin-text-inverse',
  '--admin-border',
  '--admin-border-strong',
  '--admin-focus-ring',
  '--admin-brand',
  '--admin-on-brand',
  '--admin-success-bg',
  '--admin-success-fg',
  '--admin-success-border',
  '--admin-warning-bg',
  '--admin-warning-fg',
  '--admin-warning-border',
  '--admin-danger-bg',
  '--admin-danger-fg',
  '--admin-danger-border',
  '--admin-info-bg',
  '--admin-info-fg',
  '--admin-info-border',
  '--admin-radius-sm',
  '--admin-radius-md',
  '--admin-radius-lg',
  '--admin-shadow-sm',
  '--admin-shadow-md',
  '--admin-shadow-lg',
  '--admin-font-sans',
  '--admin-font-mono',
  '--admin-numeric-variant',
  '--admin-control-height',
  '--admin-row-padding-y',
  '--admin-page-gap',
  '--admin-content-width',
] as const

export const OPTIONAL_ADMIN_THEME_TOKENS = [
  '--admin-overlay-backdrop',
  '--admin-sidebar-bg',
  '--admin-sidebar-border',
  '--admin-sidebar-text',
  '--admin-sidebar-muted',
  '--admin-sidebar-hover-bg',
  '--admin-sidebar-hover-fg',
  '--admin-sidebar-active-bg',
  '--admin-sidebar-active-fg',
  '--admin-transition-duration',
  '--admin-accent-fg',
  '--admin-accent-bg',
  '--admin-density-control-height',
  '--admin-density-row-padding-y',
  // Motion
  '--admin-duration-fast',
  '--admin-duration-normal',
  '--admin-duration-slow',
  '--admin-duration-slower',
  '--admin-anim-duration-fast',
  '--admin-anim-duration-normal',
  '--admin-anim-duration-slow',
  '--admin-anim-duration-slowest',
  '--admin-easing-default',
  '--admin-easing-ease',
  '--admin-easing-bounce',
  // Typography
  '--admin-letter-spacing-heading',
  '--admin-letter-spacing-body',
  '--admin-letter-spacing-snug',
  '--admin-font-feature-settings',
  // Focus ring
  '--admin-focus-ring-offset',
  '--admin-focus-ring-shadow',
  // Transform
  '--admin-scale-in',
  '--admin-scale-tab',
  '--admin-scale-pop',
  '--admin-scale-rest',
  '--admin-translate-tab-y',
  '--admin-translate-subtab-y',
  '--admin-translate-pop-y',
  '--admin-translate-news-x',
  '--admin-translate-content-x',
  '--admin-rotate-chevron',
  // Backdrop
  '--admin-backdrop-blur',
  // Gradient
  '--admin-gradient-brand',
  '--admin-gradient-placeholder',
  // Skeleton
  '--admin-skeleton-bg',
  '--admin-skeleton-shimmer-start',
  '--admin-skeleton-shimmer-mid',
  '--admin-skeleton-shimmer-end',
  '--admin-skeleton-bar-height',
  '--admin-skeleton-circle-size',
  '--admin-skeleton-line-height',
  '--admin-skeleton-duration',
  '--admin-skeleton-easing',
  '--admin-skeleton-shimmer-gradient',
  '--admin-skeleton-shimmer-size',
  // Progress
  '--admin-progress-height',
  '--admin-progress-max-width',
  '--admin-progress-track-bg',
  '--admin-progress-fill-bg',
] as const

export type RequiredAdminThemeToken = (typeof REQUIRED_ADMIN_THEME_TOKENS)[number]
export type OptionalAdminThemeToken = (typeof OPTIONAL_ADMIN_THEME_TOKENS)[number]
export type AdminThemeTokenMap = Record<RequiredAdminThemeToken, string> & Partial<Record<OptionalAdminThemeToken, string>>

export const ADMIN_THEME_TOKEN_FAMILIES = {
  canvas: ['--admin-canvas', '--admin-surface', '--admin-surface-subtle'],
  text: ['--admin-text-primary', '--admin-text-secondary', '--admin-text-muted', '--admin-text-inverse'],
  border: ['--admin-border', '--admin-border-strong'],
  focus: ['--admin-focus-ring'],
  brand: ['--admin-brand', '--admin-on-brand'],
  status: [
    '--admin-success-bg', '--admin-success-fg', '--admin-success-border',
    '--admin-warning-bg', '--admin-warning-fg', '--admin-warning-border',
    '--admin-danger-bg', '--admin-danger-fg', '--admin-danger-border',
    '--admin-info-bg', '--admin-info-fg', '--admin-info-border',
  ],
  radius: ['--admin-radius-sm', '--admin-radius-md', '--admin-radius-lg'],
  shadow: ['--admin-shadow-sm', '--admin-shadow-md', '--admin-shadow-lg'],
  type: ['--admin-font-sans', '--admin-font-mono', '--admin-numeric-variant'],
  density: ['--admin-control-height', '--admin-row-padding-y'],
  layout: ['--admin-page-gap', '--admin-content-width'],
} as const satisfies Record<string, readonly RequiredAdminThemeToken[]>

export function completeAdminThemeTokenMap(values: Partial<AdminThemeTokenMap>): AdminThemeTokenMap {
  const missing = REQUIRED_ADMIN_THEME_TOKENS.filter((token) => {
    const value = values[token]
    return typeof value !== 'string' || value.trim() === ''
  })
  if (missing.length > 0) {
    throw new Error(`Missing required Admin theme tokens: ${missing.join(', ')}`)
  }
  return values as AdminThemeTokenMap
}

/** Complete the shared optional runtime token set without embedding a pack's palette in foundation. */
export function completeFullAdminThemeTokenMap(
  values: Partial<AdminThemeTokenMap>,
  overrides: Partial<AdminThemeTokenMap> = {},
): AdminThemeTokenMap {
  const surface = values['--admin-surface']!
  const subtle = values['--admin-surface-subtle']!
  const border = values['--admin-border']!
  const text = values['--admin-text-primary']!
  const secondary = values['--admin-text-secondary']!
  const muted = values['--admin-text-muted']!
  return completeAdminThemeTokenMap({
    ...values,
    '--admin-overlay-backdrop': 'rgb(2 6 23 / 0.56)',
    '--admin-sidebar-bg': surface, '--admin-sidebar-border': border,
    '--admin-sidebar-text': secondary, '--admin-sidebar-muted': muted,
    '--admin-sidebar-hover-bg': subtle, '--admin-sidebar-hover-fg': text,
    '--admin-sidebar-active-bg': 'var(--brand-50)', '--admin-sidebar-active-fg': 'var(--admin-brand)',
    '--admin-transition-duration': '150ms', '--admin-accent-fg': '#7c3aed',
    '--admin-accent-bg': 'color-mix(in srgb, #7c3aed 9%, transparent)',
    '--admin-density-control-height': '32px', '--admin-density-row-padding-y': '10px',
    '--admin-duration-fast': 'var(--duration-fast)', '--admin-duration-normal': 'var(--duration-normal)', '--admin-duration-slow': 'var(--duration-slow)', '--admin-duration-slower': 'var(--duration-slower)',
    '--admin-anim-duration-fast': 'var(--anim-duration-fast)', '--admin-anim-duration-normal': 'var(--anim-duration-normal)', '--admin-anim-duration-slow': 'var(--anim-duration-slow)', '--admin-anim-duration-slowest': 'var(--anim-duration-slowest)',
    '--admin-easing-default': 'var(--easing-default)', '--admin-easing-ease': 'var(--easing-ease)', '--admin-easing-bounce': 'var(--easing-bounce)',
    '--admin-letter-spacing-heading': 'var(--letter-spacing-tight)', '--admin-letter-spacing-body': 'var(--letter-spacing-normal)', '--admin-letter-spacing-snug': 'var(--letter-spacing-snug)', '--admin-font-feature-settings': '"cv11", "ss01", "tnum"',
    '--admin-focus-ring-offset': '0', '--admin-focus-ring-shadow': '0 0 0 var(--admin-focus-ring-width) var(--admin-focus-ring)',
    '--admin-scale-in': 'var(--scale-97)', '--admin-scale-tab': 'var(--scale-96)', '--admin-scale-pop': 'var(--scale-92)', '--admin-scale-rest': 'var(--scale-100)',
    '--admin-translate-tab-y': 'calc(-1 * var(--translate-xs))', '--admin-translate-subtab-y': 'var(--translate-sm)', '--admin-translate-pop-y': 'var(--translate-md)', '--admin-translate-news-x': 'calc(-1 * var(--translate-lg))', '--admin-translate-content-x': 'var(--translate-xl)', '--admin-rotate-chevron': 'var(--rotate-180)', '--admin-backdrop-blur': 'var(--blur-md)',
    '--admin-gradient-brand': 'linear-gradient(var(--gradient-diagonal), var(--admin-brand), var(--brand-600))', '--admin-gradient-placeholder': 'linear-gradient(var(--gradient-diagonal), var(--surface-300, var(--surface-3)), var(--surface-200, var(--surface-2)))',
    '--admin-skeleton-bg': 'var(--surface-200)', '--admin-skeleton-shimmer-start': 'var(--surface-200)', '--admin-skeleton-shimmer-mid': 'var(--surface-100)', '--admin-skeleton-shimmer-end': 'var(--surface-200)', '--admin-skeleton-bar-height': 'var(--skeleton-bar-height)', '--admin-skeleton-circle-size': 'var(--skeleton-circle-size)', '--admin-skeleton-line-height': 'var(--skeleton-line-height)', '--admin-skeleton-duration': 'var(--anim-duration-slowest)', '--admin-skeleton-easing': 'var(--easing-default)', '--admin-skeleton-shimmer-gradient': 'linear-gradient(90deg, var(--admin-skeleton-shimmer-start) 0%, var(--admin-skeleton-shimmer-mid) 50%, var(--admin-skeleton-shimmer-end) 100%)', '--admin-skeleton-shimmer-size': 'var(--skeleton-shimmer-size)',
    '--admin-progress-height': 'var(--progress-height)', '--admin-progress-max-width': 'var(--progress-max-width)', '--admin-progress-track-bg': 'var(--surface-200)', '--admin-progress-fill-bg': 'var(--admin-brand)',
    ...overrides,
  })
}

// ---- Registry types ----

export type ThemeMode = 'light' | 'dark'
export type AdminDensity = 'comfortable' | 'compact'

export interface BrandPresetMeta {
  key: string
  label: string
  hex: string
  description: string
}

/** A consumer-installed theme pack. Concrete values live in admin-theme-* packages. */
export interface AdminThemePack {
  key: string
  label: string
  description: string
  compatibilityVersion: typeof ADMIN_THEME_COMPATIBILITY_VERSION
  defaultBrand: string
  brandPolicy: 'fixed' | 'preset-list'
  brandPresets: BrandPresetMeta[]
  defaultDensity: AdminDensity
  preview: string
  tokens: { light: AdminThemeTokenMap; dark: AdminThemeTokenMap }
}

export type AdminThemeRegistry = ReadonlyMap<string, AdminThemePack>

export function createAdminThemeRegistry(packs: readonly AdminThemePack[]): AdminThemeRegistry {
  if (packs.length === 0) {
    throw new Error('createAdminThemeRegistry requires at least one installed theme pack')
  }
  const registry = new Map<string, AdminThemePack>()
  for (const pack of packs) {
    if (!pack.key) throw new Error('Theme pack key must not be empty')
    if (registry.has(pack.key)) throw new Error(`Duplicate theme pack key: ${pack.key}`)
    if (pack.compatibilityVersion !== ADMIN_THEME_COMPATIBILITY_VERSION) {
      throw new Error(`Theme pack ${pack.key} has incompatible version ${pack.compatibilityVersion}`)
    }
    if (!pack.brandPresets.some((brand) => brand.key === pack.defaultBrand)) {
      throw new Error(`Theme pack ${pack.key} has an invalid default brand`)
    }
    for (const mode of ['light', 'dark'] as const) {
      const missing = REQUIRED_ADMIN_THEME_TOKENS.filter((token) => !pack.tokens[mode]?.[token]?.trim())
      if (missing.length) throw new Error(`Theme pack ${pack.key} ${mode} is missing tokens: ${missing.join(', ')}`)
    }
    registry.set(pack.key, pack)
  }
  return registry
}

// ---- DOM helper: apply theme selection to <html> ----

export interface AdminThemeSelection {
  packKey: string
  brand: string
  mode: ThemeMode
  density: AdminDensity
}

/** Apply a theme selection to document.documentElement via data attributes.
 *  The theme pack CSS files respond to these attributes. */
export function applyAdminThemeToDocument(selection: AdminThemeSelection, locale?: string): void {
  if (typeof document === 'undefined') return
  const root = document.documentElement
  root.dataset['themeStyle'] = selection.packKey
  root.dataset['brand'] = selection.brand
  root.dataset['adminDensity'] = selection.density
  root.classList.toggle('dark', selection.mode === 'dark')
  if (locale) {
    root.dataset['locale'] = locale
    root.lang = locale
  }
}
