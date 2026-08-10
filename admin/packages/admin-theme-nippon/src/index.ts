import { completeFullAdminThemeTokenMap, type AdminThemePack, type AdminThemeTokenMap } from '@sitecore/admin-ui/theme'

function completeAdminThemeTokenMap(values: Partial<AdminThemeTokenMap>): AdminThemeTokenMap {
  const dark = values['--admin-on-brand'] === 'var(--surface-950)'
  return completeFullAdminThemeTokenMap(values, {
    '--admin-overlay-backdrop': dark ? 'rgba(0, 0, 0, 0.55)' : 'rgba(12, 11, 8, 0.42)',
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
  ['ai', '藍 Ai', '#2c5784', '群青 · 深靛藍（預設）'],
  ['tokiwa', '常磐 Tokiwa', '#1a6e4a', '常綠深綠'],
  ['beni', '紅 Beni', '#ce3a4e', '傳統緋紅'],
  ['yamabuki', '山吹 Yamabuki', '#e8a33d', '山吹金色'],
  ['tsuyukusa', '露草 Tsuyukusa', '#2ea9df', '鴨跖草藍'],
  ['kikyo', '桔梗 Kikyo', '#6a4c9c', '桔梗紫'],
  ['fujinezumi', '藤鼠 Fujinezumi', '#6e75a4', '藤紫灰'],
  ['shu', '朱 Shu', '#d34a38', '朱紅'],
]
const presets = presetRows.map(([key, label, hex, description]) => ({ key, label, hex, description }))

function tokens(dark = false): AdminThemeTokenMap {
  const values = dark
    ? ['var(--surface-50)', 'var(--surface-0)', 'var(--surface-100)', 'var(--surface-800)', 'var(--surface-600)', 'var(--surface-500)', 'var(--surface-50)', 'var(--surface-200)', 'var(--surface-300)', 'var(--brand-400)', 'var(--brand-400)', 'var(--surface-950)', '#163328', '#92c5a8', '#286747', '#3a2b16', '#f4c577', '#6d4f1f', '#3b1e23', '#ee9bab', '#71333f', '#18303c', '#88d3f4', '#2b5f78', '2px', '6px', '8px', '0 1px 2px rgb(0 0 0 / .3)', '0 4px 12px rgb(0 0 0 / .35)', '0 4px 24px rgb(0 0 0 / .5)', '"Inter", "Noto Sans TC", system-ui, sans-serif', '"JetBrains Mono", ui-monospace, monospace', 'tabular-nums', '40px', '.625rem', '1.5rem', '1536px']
    : ['var(--surface-50)', 'var(--surface-0)', 'var(--surface-100)', 'var(--surface-800)', 'var(--surface-600)', 'var(--surface-500)', 'var(--surface-0)', 'var(--surface-200)', 'var(--surface-300)', 'var(--brand-500)', 'var(--brand-600)', 'var(--surface-0)', '#e6f1eb', '#1a6e4a', '#92c5a8', '#fdf3e1', '#754d13', '#f4c577', '#fbeaee', '#80222e', '#ee9bab', '#e6f5fd', '#245a78', '#88d3f4', '2px', '6px', '8px', '0 1px 2px rgb(40 39 31 / .04)', '0 4px 12px rgb(40 39 31 / .06)', '0 4px 24px rgb(40 39 31 / .12)', '"Inter", "Noto Sans TC", system-ui, sans-serif', '"JetBrains Mono", ui-monospace, monospace', 'tabular-nums', '40px', '.625rem', '1.5rem', '1536px']
  const keys = ['--admin-canvas', '--admin-surface', '--admin-surface-subtle', '--admin-text-primary', '--admin-text-secondary', '--admin-text-muted', '--admin-text-inverse', '--admin-border', '--admin-border-strong', '--admin-focus-ring', '--admin-brand', '--admin-on-brand', '--admin-success-bg', '--admin-success-fg', '--admin-success-border', '--admin-warning-bg', '--admin-warning-fg', '--admin-warning-border', '--admin-danger-bg', '--admin-danger-fg', '--admin-danger-border', '--admin-info-bg', '--admin-info-fg', '--admin-info-border', '--admin-radius-sm', '--admin-radius-md', '--admin-radius-lg', '--admin-shadow-sm', '--admin-shadow-md', '--admin-shadow-lg', '--admin-font-sans', '--admin-font-mono', '--admin-numeric-variant', '--admin-control-height', '--admin-row-padding-y', '--admin-page-gap', '--admin-content-width']
  return completeAdminThemeTokenMap(Object.fromEntries(keys.map((key, index) => [key, values[index]])))
}

export const nipponThemePack: AdminThemePack = {
  key: 'nippon',
  label: 'Nippon',
  description: '和紙 Kinari 暖白 + Gruvbox Glass 暗色',
  compatibilityVersion: 1,
  defaultBrand: 'ai',
  brandPolicy: 'preset-list',
  brandPresets: presets,
  defaultDensity: 'comfortable',
  preview: 'linear-gradient(135deg, #fdfcf8, #f6f4ee, #1d2021)',
  tokens: { light: tokens(), dark: tokens(true) },
}

export default nipponThemePack
