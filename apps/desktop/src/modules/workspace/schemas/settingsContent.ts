export const localThemeCards = [
  {
    title: "主题模式",
    lines: ["跟随系统 / Dark / Light", "dark-first, light 为对照模式"],
    tone: "default" as const
  },
  {
    title: "预设主题与排版",
    lines: [
      "预设: Graphite / Carbon / Slate",
      "字体大小: Small / Default / Large",
      "紧凑度: Compact / Comfortable"
    ],
    tone: "default" as const
  },
  {
    title: "预览提示",
    lines: ["切换后即时预览，不影响业务数据。"],
    tone: "info" as const
  }
];

export const localI18nCards = [
  {
    title: "语言切换",
    lines: ["zh-CN / en-US", "切换后即时生效"],
    tone: "default" as const
  },
  {
    title: "长文案容纳抽测",
    lines: [
      "English Stress Test: This is a deliberately long translation sample to validate wrapping, clipping, and line-height behavior across configuration cards."
    ],
    tone: "default" as const
  },
  {
    title: "i18n 说明",
    lines: ["中文主稿 + 英文抽测，确保关键控件不破版。"],
    tone: "info" as const
  }
];

export const localUpdatesCards = [
  {
    title: "版本信息",
    lines: ["当前版本: 自动读取", "最新版本: 按发布源检查", "操作: 获取最新版本 Check Update"],
    tone: "default" as const
  },
  {
    title: "诊断与日志",
    lines: ["Hub 连接诊断: healthy", "日志导出: export latest session log"],
    tone: "success" as const
  },
  {
    title: "更新策略",
    lines: ["仅检查客户端版本，不影响远程工作区状态。"],
    tone: "info" as const
  }
];

export const localGeneralCards = [
  {
    title: "软件通用项",
    lines: ["开机启动: enabled", "默认目录: ~/Workspace/Goyais"],
    tone: "default" as const
  },
  {
    title: "隐私与通知策略",
    lines: ["遥测: minimized", "通知: 断线重连 / 审批反馈 / 错误告警"],
    tone: "default" as const
  },
  {
    title: "作用域说明",
    lines: ["本页面配置与工作区无关，属于客户端级设置。"],
    tone: "info" as const
  }
];
