import { completeFullAdminThemeTokenMap, type AdminThemePack, type AdminThemeTokenMap } from '@sitecore/admin-ui/theme'

function completeAdminThemeTokenMap(values: Partial<AdminThemeTokenMap>): AdminThemeTokenMap {
  const dark = values['--admin-on-brand'] === 'var(--surface-950)'
  return completeFullAdminThemeTokenMap(values, {
    '--admin-overlay-backdrop': dark ? 'rgba(0, 0, 0, 0.55)' : 'rgba(17, 24, 39, 0.42)',
    '--admin-sidebar-bg': 'var(--sidebar-bg)',
    '--admin-sidebar-border': 'var(--sidebar-border)',
    '--admin-sidebar-text': 'var(--surface-600)',
    '--admin-sidebar-muted': dark ? 'var(--surface-500)' : 'var(--surface-400)',
    '--admin-sidebar-active-bg': 'var(--brand-50)',
    '--admin-sidebar-active-fg': dark ? 'var(--brand-400)' : 'var(--brand-600)',
    '--admin-sidebar-hover-bg': 'var(--surface-100)',
    '--admin-sidebar-hover-fg': 'var(--surface-800)',
  })
}

const presetRows: [string, string, string, string][] = [
  ['blue', 'Blue', '#2563eb', 'Tailwind blue-600（預設）'],
  ['indigo', 'Indigo', '#4f46e5', 'Tailwind indigo-600'],
  ['violet', 'Violet', '#7c3aed', 'Tailwind violet-600'],
  ['emerald', 'Emerald', '#059669', 'Tailwind emerald-600'],
  ['rose', 'Rose', '#e11d48', 'Tailwind rose-600'],
  ['amber', 'Amber', '#d97706', 'Tailwind amber-600'],
  ['cyan', 'Cyan', '#0891b2', 'Tailwind cyan-600'],
  ['slate', 'Slate', '#475569', 'Tailwind slate-600'],
]
const presets = presetRows.map(([key, label, hex, description]) => ({ key, label, hex, description }))

function tokens(dark = false): AdminThemeTokenMap {
  const values = dark
    ? ['var(--surface-50)', 'var(--surface-0)', 'var(--surface-100)', 'var(--surface-800)', 'var(--surface-600)', 'var(--surface-500)', 'var(--surface-50)', 'var(--surface-200)', 'var(--surface-300)', 'var(--brand-400)', 'var(--brand-400)', 'var(--surface-950)', '#052e2b', '#6ee7b7', '#065f46', '#451a03', '#fcd34d', '#92400e', '#4c0519', '#fda4af', '#9f1239', '#172554', '#93c5fd', '#1e40af', '2px', '6px', '8px', '0 1px 2px rgb(0 0 0 / .3)', '0 4px 12px rgb(0 0 0 / .35)', '0 4px 24px rgb(0 0 0 / .5)', '"Inter", "Noto Sans TC", system-ui, sans-serif', '"JetBrains Mono", ui-monospace, monospace', 'tabular-nums', '40px', '.625rem', '1.5rem', '1536px']
    : ['var(--surface-50)', 'var(--surface-0)', 'var(--surface-100)', 'var(--surface-800)', 'var(--surface-600)', 'var(--surface-500)', 'var(--surface-0)', 'var(--surface-200)', 'var(--surface-300)', 'var(--brand-500)', 'var(--brand-600)', 'var(--surface-0)', '#ecfdf5', '#047857', '#a7f3d0', '#fffbeb', '#b45309', '#fde68a', '#fff1f2', '#be123c', '#fecdd3', '#eff6ff', '#1d4ed8', '#bfdbfe', '2px', '6px', '8px', '0 1px 2px rgb(17 24 39 / .04)', '0 4px 12px rgb(17 24 39 / .06)', '0 4px 24px rgb(17 24 39 / .12)', '"Inter", "Noto Sans TC", system-ui, sans-serif', '"JetBrains Mono", ui-monospace, monospace', 'tabular-nums', '40px', '.625rem', '1.5rem', '1536px']
  const keys = ['--admin-canvas', '--admin-surface', '--admin-surface-subtle', '--admin-text-primary', '--admin-text-secondary', '--admin-text-muted', '--admin-text-inverse', '--admin-border', '--admin-border-strong', '--admin-focus-ring', '--admin-brand', '--admin-on-brand', '--admin-success-bg', '--admin-success-fg', '--admin-success-border', '--admin-warning-bg', '--admin-warning-fg', '--admin-warning-border', '--admin-danger-bg', '--admin-danger-fg', '--admin-danger-border', '--admin-info-bg', '--admin-info-fg', '--admin-info-border', '--admin-radius-sm', '--admin-radius-md', '--admin-radius-lg', '--admin-shadow-sm', '--admin-shadow-md', '--admin-shadow-lg', '--admin-font-sans', '--admin-font-mono', '--admin-numeric-variant', '--admin-control-height', '--admin-row-padding-y', '--admin-page-gap', '--admin-content-width']
  return completeAdminThemeTokenMap(Object.fromEntries(keys.map((key, index) => [key, values[index]])))
}

export const tailwindThemePack: AdminThemePack = {
  key: 'tailwind',
  label: 'Tailwind',
  description: '標準 white / zinc-950 + 常用主色',
  compatibilityVersion: 1,
  defaultBrand: 'blue',
  brandPolicy: 'preset-list',
  brandPresets: presets,
  defaultDensity: 'comfortable',
  preview: 'linear-gradient(135deg, #fff, #f9fafb, #09090b)',
  tokens: { light: tokens(), dark: tokens(true) },
}

export default tailwindThemePack
