export default {
  nav: {
    home: 'Home',
    canvas: 'Canvas',
    commands: 'Commands',
    assets: 'Assets',
    plugins: 'Plugins',
    streams: 'Streams',
    settings: 'Settings',
  },
  common: {
    appName: 'Goyais Console',
    workspace: 'workspace-alpha',
    language: '语言',
    theme: '主题',
    density: '密度',
    system: '跟随系统',
    light: '浅色',
    dark: '深色',
    compact: '紧凑',
    comfortable: '舒适',
    searchPlaceholder: '全局搜索（Cmd+K）',
    loading: '加载中',
    cancel: '取消',
    confirm: '确认',
    close: '关闭',
    save: '保存',
    copy: '复制',
    copied: '已复制',
    empty: '暂无数据',
    refresh: '刷新',
    previous: '上一页',
    next: '下一页',
  },
  page: {
    home: {
      title: 'Console 基线',
      subtitle: '统一 UI 规范、主题、密度与 i18n 已完成基座化。',
    },
    canvas: {
      title: 'Canvas 占位页',
      subtitle: '仅提供布局与视觉基线，不接真实业务数据。',
    },
    commands: {
      title: 'Command Runtime 占位',
      subtitle: 'CommandCard / StatusBadge / LogPanel 仅展示结构。',
    },
    assets: {
      title: 'Assets 占位页',
      subtitle: '用于验证列表页的表格/分页/空状态规范。',
    },
    plugins: {
      title: 'Plugins 占位页',
      subtitle: '用于验证卡片页布局与状态标签规范。',
    },
    streams: {
      title: 'Streams 占位页',
      subtitle: '用于验证状态条、信息密度与单色语义。',
    },
    settings: {
      title: 'Settings 占位页',
      subtitle: '集中展示主题、语言与密度切换。',
    },
  },
  status: {
    accepted: '已接收',
    running: '执行中',
    succeeded: '成功',
    failed: '失败',
    canceled: '已取消',
  },
  error: {
    common: {
      unknown: '未知错误，请稍后重试。',
    },
    context: {
      missing: '请求上下文缺失。',
    },
    authz: {
      forbidden: '当前用户没有访问权限。',
    },
    command: {
      invalidPayload: '命令入参不合法。',
    },
  },
}
