import type { Config } from "tailwindcss";

const config: Config = {
  darkMode: ["class"],
  content: ["./index.html", "./src/**/*.{ts,tsx}"],
  theme: {
    extend: {
      colors: {
        background: "hsl(var(--color-background) / <alpha-value>)",
        foreground: "hsl(var(--color-foreground) / <alpha-value>)",
        muted: "hsl(var(--color-muted) / <alpha-value>)",
        "muted-foreground": "hsl(var(--color-muted-foreground) / <alpha-value>)",
        border: "hsl(var(--color-border) / <alpha-value>)",
        "border-subtle": "hsl(var(--color-border-subtle) / <alpha-value>)",
        accent: "hsl(var(--color-accent) / <alpha-value>)",
        success: "hsl(var(--color-success) / <alpha-value>)",
        warning: "hsl(var(--color-warning) / <alpha-value>)",
        info: "hsl(var(--color-info) / <alpha-value>)",
        destructive: "hsl(var(--color-destructive) / <alpha-value>)",
        risk: {
          write: "hsl(var(--risk-write) / <alpha-value>)",
          exec: "hsl(var(--risk-exec) / <alpha-value>)",
          network: "hsl(var(--risk-network) / <alpha-value>)",
          delete: "hsl(var(--risk-delete) / <alpha-value>)",
          exfil: "hsl(var(--risk-exfil) / <alpha-value>)"
        }
      },
      borderRadius: {
        control: "var(--radius-control)",
        panel: "var(--radius-panel)",
        overlay: "var(--radius-overlay)"
      },
      boxShadow: {
        panel: "none",
        sm: "var(--shadow-sm)",
        md: "var(--shadow-md)",
        lg: "var(--shadow-lg)"
      },
      fontFamily: {
        sans: ["var(--font-ui)"],
        mono: ["var(--font-mono)"]
      },
      fontSize: {
        h1: ["var(--text-h1-size)", { lineHeight: "var(--text-h1-lh)", fontWeight: "600" }],
        h2: ["var(--text-h2-size)", { lineHeight: "var(--text-h2-lh)", fontWeight: "600" }],
        body: ["var(--text-body-size)", { lineHeight: "var(--text-body-lh)" }],
        small: ["var(--text-small-size)", { lineHeight: "var(--text-small-lh)" }],
        code: ["var(--text-mono-size)", { lineHeight: "var(--text-mono-lh)" }]
      },
      spacing: {
        1: "var(--space-1)",
        2: "var(--space-2)",
        3: "var(--space-3)",
        4: "var(--space-4)",
        5: "var(--space-5)",
        6: "var(--space-6)",
        8: "var(--space-8)",
        10: "var(--space-10)",
        12: "var(--space-12)",
        page: "var(--page-padding)",
        panel: "var(--panel-gap)",
        content: "var(--content-padding)",
        form: "var(--form-gap)",
        toolbar: "var(--toolbar-height)",
        sidebar: "var(--sidebar-width)",
        "sidebar-collapsed": "var(--sidebar-width-collapsed)"
      },
      ringColor: {
        focus: "hsl(var(--focus-ring-color) / 1)"
      },
      ringWidth: {
        DEFAULT: "var(--focus-ring-width)"
      },
      ringOffsetWidth: {
        DEFAULT: "var(--focus-ring-offset)"
      }
    }
  },
  plugins: []
};

export default config;
