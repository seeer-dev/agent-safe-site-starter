import { defineStore } from 'pinia'
import { ref, watch } from 'vue'

export type AccentColor = 'amber' | 'emerald' | 'rose' | 'sky' | 'violet'
export type Mode = 'light' | 'dark'

export interface AccentPreset {
  id: AccentColor
  label: string
  cta: string
  ctaForeground: string
  ring: string
}

const ACCENTS: Record<AccentColor, AccentPreset> = {
  amber:   { id: 'amber',   label: '暖橘',   cta: '38 76% 62.8%',  ctaForeground: '0 0% 98.5%', ring: '38 76% 62.8%' },
  emerald: { id: 'emerald', label: '翡翠綠', cta: '152 56% 40%',   ctaForeground: '0 0% 98.5%', ring: '152 56% 40%' },
  rose:    { id: 'rose',    label: '玫瑰粉', cta: '346 77% 60%',   ctaForeground: '0 0% 98.5%', ring: '346 77% 60%' },
  sky:     { id: 'sky',     label: '天空藍', cta: '199 89% 52%',   ctaForeground: '0 0% 98.5%', ring: '199 89% 52%' },
  violet:  { id: 'violet',  label: '紫羅蘭', cta: '262 70% 58%',   ctaForeground: '0 0% 98.5%', ring: '262 70% 58%' },
}

const STORAGE_KEY = 'monolith-theme'

interface StoredTheme {
  accent: AccentColor
  mode: Mode
}

export const useThemeStore = defineStore('theme', () => {
  const accent = ref<AccentColor>('amber')
  const mode = ref<Mode>('light')
  const accents = ACCENTS

  function applyAccent(id: AccentColor) {
    const preset = ACCENTS[id]
    const root = document.documentElement
    root.style.setProperty('--cta', preset.cta)
    root.style.setProperty('--cta-foreground', preset.ctaForeground)
    root.style.setProperty('--ring', preset.ring)
  }

  function applyMode(m: Mode) {
    const root = document.documentElement
    if (m === 'dark') root.classList.add('dark')
    else root.classList.remove('dark')
  }

  function setAccent(id: AccentColor) {
    accent.value = id
    applyAccent(id)
    persist()
  }

  function setMode(m: Mode) {
    mode.value = m
    applyMode(m)
    persist()
  }

  function toggleMode() {
    setMode(mode.value === 'light' ? 'dark' : 'light')
  }

  function persist() {
    const data: StoredTheme = { accent: accent.value, mode: mode.value }
    localStorage.setItem(STORAGE_KEY, JSON.stringify(data))
  }

  function restore() {
    try {
      const raw = localStorage.getItem(STORAGE_KEY)
      if (raw) {
        const data = JSON.parse(raw) as StoredTheme
        accent.value = data.accent
        mode.value = data.mode
      }
    } catch {
      // use defaults
    }
    applyAccent(accent.value)
    applyMode(mode.value)
  }

  return {
    accent,
    mode,
    accents,
    setAccent,
    setMode,
    toggleMode,
    restore,
  }
})
