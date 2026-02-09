# Goyais 国际化（i18n）设计

> 本文档定义 Goyais 的产品级国际化能力（API、前端、通知、错误消息与治理流程），用于指导实现“中英文可切换、可扩展到更多语言”的统一方案。

最后更新：2026-02-09

---

## 1. 目标与范围

### 1.1 目标

1. 支持用户在 UI/API 层按语言环境获取文本内容。
2. 支持 `zh-CN` 与 `en` 双语作为首批语言。
3. 保证错误码稳定，文本可本地化替换。
4. 保证审批、通知、系统提示可按 locale 渲染。

### 1.2 非目标

1. 不在 v1 做自动机器翻译。
2. 不在 v1 做多区域法规内容差异化。
3. 不在 v1 做多时区之外的复杂地区格式自定义模板。

---

## 2. 术语

- `locale`: 语言区域标识（如 `zh-CN`, `en`）。
- `message_key`: 稳定消息键（如 `error.policy.blocked`）。
- `fallback_locale`: 回退语言，默认 `en`。

---

## 3. 架构方案

### 3.1 语言协商顺序

1. `X-Locale` 请求头（显式覆盖）
2. `Accept-Language` 请求头
3. 用户偏好（UserProfile.PreferredLocale）
4. 系统默认（`en`）

### 3.2 文案输出策略

1. API 响应稳定字段保留：`code`、`error.type`、`error.reason`。
2. `message` 与 `error.localized_message` 按 locale 输出。
3. 若无对应翻译，回退到 `fallback_locale`。

### 3.3 前端策略

1. 前端内置 i18n 资源包（`zh-CN`, `en`）。
2. 全局语言切换器写入用户偏好与本地缓存。
3. 运行时 UI 文案、状态文案、审批提示全部使用 translation key。

---

## 4. 领域模型扩展建议

### 4.1 用户偏好

```go
type UserProfile struct {
    UserID           uuid.UUID
    PreferredLocale  string // "zh-CN" | "en"
    Timezone         string
}
```

### 4.2 本地化消息对象

```go
type LocalizedMessage struct {
    Key       string            // error.policy.blocked
    Locale    string            // zh-CN / en
    Text      string
    Params    map[string]string // 模板参数
    UpdatedAt time.Time
}
```

---

## 5. API 约定

### 5.1 请求头

- `Accept-Language`: 标准语言协商头
- `X-Locale`: 显式 locale 覆盖头

### 5.2 响应字段（建议）

```json
{
  "code": "POLICY_BLOCKED",
  "message": "策略拒绝本次操作",
  "error": {
    "type": "policy_violation",
    "reason": "missing_permission",
    "message_key": "error.policy.blocked",
    "localized_message": "当前角色缺少所需权限"
  },
  "meta": {
    "locale": "zh-CN",
    "fallback": false
  }
}
```

---

## 6. 前端设计约束

1. 禁止在组件中硬编码可见文案。
2. 所有用户可见文本必须走 i18n key。
3. 错误提示优先使用服务端 `message_key` + 参数渲染。
4. 页面级 locale 切换不触发全局登出。

---

## 7. 质量与验收

### 7.1 测试要求

1. API 双语快照测试（`zh-CN`, `en`）。
2. 前端关键页面双语渲染测试。
3. 回退策略测试（缺翻译 -> fallback）。

### 7.2 发布门禁

1. 新增用户可见文案必须附中英文翻译。
2. 新增错误码必须附 `message_key` 与双语文本。
3. PR 必须说明 i18n 影响与验证结果。

---

## 8. 实施计划映射

- `S1-I18N-001`: API locale 协商与响应字段扩展
- `S1-I18N-002`: 前端 i18n 基础设施（store/composable/route）
- `S2-I18N-001`: 审批与通知双语渲染
- `S2-I18N-002`: 错误码消息模板中心化
- `S3-I18N-001`: 语音意图输出的双语提示与确认

