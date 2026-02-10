import type { Config } from 'tailwindcss'

export default {
  darkMode: 'class',
  content: ['./index.html', './src/**/*.{vue,ts,js}'],
  theme: {
    extend: {
      colors: {
        ui: {
          bg: 'rgb(var(--ui-neutral-bg) / <alpha-value>)',
          fg: 'rgb(var(--ui-neutral-fg) / <alpha-value>)',
          muted: 'rgb(var(--ui-neutral-muted) / <alpha-value>)',
          border: 'rgb(var(--ui-neutral-border) / <alpha-value>)',
          hover: 'rgb(var(--ui-neutral-hover) / <alpha-value>)',
          pressed: 'rgb(var(--ui-neutral-pressed) / <alpha-value>)',
          panel: 'rgb(var(--ui-neutral-panel) / <alpha-value>)',
        },
        primary: {
          500: 'rgb(var(--ui-primary-500) / <alpha-value>)',
          600: 'rgb(var(--ui-primary-600) / <alpha-value>)',
          700: 'rgb(var(--ui-primary-700) / <alpha-value>)',
        },
        success: 'rgb(var(--ui-success) / <alpha-value>)',
        warn: 'rgb(var(--ui-warn) / <alpha-value>)',
        error: 'rgb(var(--ui-error) / <alpha-value>)',
        info: 'rgb(var(--ui-info) / <alpha-value>)',
      },
      borderRadius: {
        card: 'var(--ui-radius-card)',
        button: 'var(--ui-radius-button)',
        canvas: 'var(--ui-radius-canvas-node)',
      },
      boxShadow: {
        overlay: 'var(--ui-shadow-overlay)',
      },
      fontFamily: {
        sans: ['var(--ui-font-sans)', 'ui-sans-serif', 'system-ui'],
        mono: ['var(--ui-font-mono)', 'ui-monospace', 'SFMono-Regular'],
      },
      spacing: {
        'page-gap': 'var(--ui-page-gap)',
      },
    },
  },
  plugins: [],
} satisfies Config
