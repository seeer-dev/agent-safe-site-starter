import { completeFullAdminThemeTokenMap, type AdminThemePack, type AdminThemeTokenMap } from '@sitecore/admin-ui/theme'

function completeAdminThemeTokenMap(values: Partial<AdminThemeTokenMap>): AdminThemeTokenMap {
  const dark = values['--admin-canvas'] === '#0e1417'
  return completeFullAdminThemeTokenMap(values, {
    '--admin-overlay-backdrop': dark ? 'rgba(0, 0, 0, 0.55)' : 'rgba(12, 18, 22, 0.42)',
    '--admin-sidebar-bg': dark ? '#151d21' : '#ffffff',
    '--admin-sidebar-border': dark ? '#26333a' : '#e2e7ea',
    '--admin-sidebar-text': dark ? '#9fb0ba' : '#57636e',
    '--admin-sidebar-muted': dark ? '#6f818c' : '#8b96a0',
    '--admin-sidebar-active-bg': 'var(--brand-50)',
    '--admin-sidebar-active-fg': dark ? 'var(--brand-400)' : 'var(--brand-600)',
    '--admin-sidebar-hover-bg': dark ? '#1c262b' : '#f1f4f5',
    '--admin-sidebar-hover-fg': dark ? '#e8edf0' : '#182027',
  })
}

const presetRows: [string, string, string, string][] = [
  ['teal', 'Teal', '#0f766e', '冷調專業青綠（預設）'], ['ocean', 'Ocean', '#2563eb', '深藍，值得信賴'],
  ['forest', 'Forest', '#059669', '深綠，沉穩'], ['sunset', 'Sunset', '#ea580c', '暖橘，有活力'],
  ['berry', 'Berry', '#7c3aed', '紫色，具創意感'], ['slate', 'Slate', '#475569', '中性藍灰，安定'],
  ['cocoa', 'Cocoa', '#7a4f1e', '暖棕，樸實'], ['indigo', 'Indigo', '#4f46e5', '深靛藍，高級感'],
]
const presets = presetRows.map(([key, label, hex, description]) => ({ key, label, hex, description }))

function tokens(dark = false): AdminThemeTokenMap {
  const values = dark
    ? ['#0e1417', '#151d21', '#1c262b', '#e8edf0', '#9fb0ba', '#6f818c', '#0e1417', '#26333a', '#33434c', 'var(--brand-400)', 'var(--brand-400)', '#0e1417', '#12291a', '#86efac', '#27683a', '#33260f', '#fcd34d', '#70551d', '#331810', '#fdba9a', '#783921', '#13223f', '#93c5fd', '#294e85', '6px', '9px', '12px', '0 1px 2px rgb(0 0 0 / .3)', '0 8px 24px rgb(0 0 0 / .36)', '0 10px 34px rgb(0 0 0 / .55)', '"Inter", "Noto Sans TC", system-ui, sans-serif', '"JetBrains Mono", ui-monospace, monospace', 'tabular-nums', '40px', '.625rem', '1.5rem', '1440px']
    : ['#f5f7f8', '#fff', '#f1f4f5', '#182027', '#57636e', '#8b96a0', '#fff', '#e2e7ea', '#d3dade', 'var(--brand-500)', 'var(--brand-600)', '#fff', '#e7f5ec', '#15803d', '#b7dfc4', '#fef3e2', '#b45309', '#f2cf95', '#fdece5', '#c2410c', '#efb8a5', '#e8effd', '#1d4ed8', '#b6c9f2', '6px', '9px', '12px', '0 1px 2px rgb(16 24 32 / .04)', '0 4px 16px rgb(16 24 32 / .05)', '0 8px 30px rgb(16 24 32 / .14)', '"Inter", "Noto Sans TC", system-ui, sans-serif', '"JetBrains Mono", ui-monospace, monospace', 'tabular-nums', '40px', '.625rem', '1.5rem', '1440px']
  const keys = ['--admin-canvas', '--admin-surface', '--admin-surface-subtle', '--admin-text-primary', '--admin-text-secondary', '--admin-text-muted', '--admin-text-inverse', '--admin-border', '--admin-border-strong', '--admin-focus-ring', '--admin-brand', '--admin-on-brand', '--admin-success-bg', '--admin-success-fg', '--admin-success-border', '--admin-warning-bg', '--admin-warning-fg', '--admin-warning-border', '--admin-danger-bg', '--admin-danger-fg', '--admin-danger-border', '--admin-info-bg', '--admin-info-fg', '--admin-info-border', '--admin-radius-sm', '--admin-radius-md', '--admin-radius-lg', '--admin-shadow-sm', '--admin-shadow-md', '--admin-shadow-lg', '--admin-font-sans', '--admin-font-mono', '--admin-numeric-variant', '--admin-control-height', '--admin-row-padding-y', '--admin-page-gap', '--admin-content-width']
  return completeAdminThemeTokenMap(Object.fromEntries(keys.map((key, index) => [key, values[index]])))
}

export const adminMintThemePack: AdminThemePack = {
  key: 'admin-mint',
  label: 'Admin Mint',
  description: 'Teal / slate、清楚層級與高密度營運介面',
  compatibilityVersion: 1,
  defaultBrand: 'teal',
  brandPolicy: 'preset-list',
  brandPresets: presets,
  defaultDensity: 'comfortable',
  preview: 'linear-gradient(135deg, #fff, #e8f7f4, #0e1417)',
  tokens: { light: tokens(), dark: tokens(true) },
}

export default adminMintThemePack
