export type ThemeMode = 'system' | 'light' | 'dark'

export type DensityMode = 'compact' | 'comfortable'

export type SupportedLocale = 'zh-CN' | 'en-US'

export type CommandStatus =
  | 'accepted'
  | 'running'
  | 'succeeded'
  | 'failed'
  | 'canceled'
