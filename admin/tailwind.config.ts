import type { Config } from 'tailwindcss'

/**
 * Tailwind config derived from admin/dashboard.html mockup CSS variables.
 * The mockup uses a warm, muted palette with brand brown — we map each
 * CSS variable to a Tailwind color token so components can use
 * bg-surface, text-text-2, border-border, etc.
 */
export default {
  content: ['./index.html', './src/**/*.{vue,ts,tsx}'],
  darkMode: ['class', '[data-theme="dark"]'],
  corePlugins: {
    preflight: false,
  },
  theme: {
    extend: {
      // Fractional spacing values used throughout the admin UI.
      // Tailwind's default scale only has 0.5, 1.5, 2.5, 3.5 —
      // we add the rest so classes like h-4.5, py-1.75, px-4.5 work.
      spacing: {
        0.75: '0.1875rem',
        1.25: '0.3125rem',
        1.75: '0.4375rem',
        2.25: '0.5625rem',
        2.75: '0.6875rem',
        3.75: '0.9375rem',
        4.5: '1.125rem',
        5.5: '1.375rem',
        7.5: '1.875rem',
        8.5: '2.125rem',
        10.5: '2.625rem',
      },
      colors: {
        bg: 'rgb(var(--bg-rgb) / <alpha-value>)',
        surface: {
          DEFAULT: 'rgb(var(--surface-rgb) / <alpha-value>)',
          2: 'rgb(var(--surface-2-rgb) / <alpha-value>)',
          3: 'rgb(var(--surface-3-rgb) / <alpha-value>)',
        },
        border: {
          DEFAULT: 'rgb(var(--border-rgb) / <alpha-value>)',
          2: 'rgb(var(--border-2-rgb) / <alpha-value>)',
        },
        text: {
          DEFAULT: 'rgb(var(--text-rgb) / <alpha-value>)',
          2: 'rgb(var(--text-2-rgb) / <alpha-value>)',
          3: 'rgb(var(--text-3-rgb) / <alpha-value>)',
        },
        brand: {
          DEFAULT: 'rgb(var(--brand-rgb) / <alpha-value>)',
          600: 'rgb(var(--brand-600-rgb) / <alpha-value>)',
          50: 'rgb(var(--brand-50-rgb) / <alpha-value>)',
          100: 'rgb(var(--brand-100-rgb) / <alpha-value>)',
        },
        amber: {
          DEFAULT: 'rgb(var(--amber-rgb) / <alpha-value>)',
          bg: 'rgb(var(--amber-bg-rgb) / <alpha-value>)',
        },
        red: {
          DEFAULT: 'rgb(var(--red-rgb) / <alpha-value>)',
          bg: 'rgb(var(--red-bg-rgb) / <alpha-value>)',
        },
        blue: {
          DEFAULT: 'rgb(var(--blue-rgb) / <alpha-value>)',
          bg: 'rgb(var(--blue-bg-rgb) / <alpha-value>)',
        },
        green: {
          DEFAULT: 'rgb(var(--green-rgb) / <alpha-value>)',
          bg: 'rgb(var(--green-bg-rgb) / <alpha-value>)',
        },
      },
      boxShadow: {
        panel: '0 1px 2px rgba(30,28,25,.04), 0 4px 16px rgba(30,28,25,.05)',
        lg: '0 8px 30px rgba(30,28,25,.14)',
      },
      borderRadius: {
        DEFAULT: '10px',
      },
      fontSize: {
        base: ['14px', '1.5'],
      },
    },
  },
  plugins: [],
} satisfies Config
