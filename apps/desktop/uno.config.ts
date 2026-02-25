import { defineConfig, presetWind, transformerDirectives, transformerVariantGroup } from "unocss";

export default defineConfig({
  presets: [presetWind()],
  theme: {
    colors: {
      bg: "var(--semantic-bg)",
      surface: "var(--semantic-surface)",
      "surface-2": "var(--semantic-surface-2)",
      text: "var(--semantic-text)",
      muted: "var(--semantic-text-muted)",
      subtle: "var(--semantic-text-subtle)",
      border: "var(--semantic-border)",
      divider: "var(--semantic-divider)",
      primary: "var(--semantic-primary)",
      success: "var(--semantic-success)",
      warning: "var(--semantic-warning)",
      danger: "var(--semantic-danger)"
    },
    fontFamily: {
      sans: "var(--global-font-family-ui)",
      mono: "var(--global-font-family-code)"
    },
    borderRadius: {
      sm: "var(--global-radius-8)",
      md: "var(--global-radius-12)",
      lg: "var(--global-radius-16)",
      full: "var(--global-radius-pill)"
    }
  },
  shortcuts: [
    [
      "ui-surface-card",
      "bg-[var(--semantic-surface)] text-[var(--semantic-text)] border border-[var(--semantic-border)] rounded-[var(--global-radius-16)]"
    ],
    [
      "ui-input-shell",
      "bg-[var(--component-input-bg)] text-[var(--component-input-fg)] border border-[var(--component-input-border)] rounded-[var(--component-input-radius)]"
    ],
    [
      "ui-sidebar-item",
      "text-[var(--component-sidebar-item-fg)] hover:bg-[var(--component-sidebar-item-bg-hover)] hover:text-[var(--component-sidebar-item-fg-active)]"
    ]
  ],
  transformers: [transformerDirectives(), transformerVariantGroup()]
});
