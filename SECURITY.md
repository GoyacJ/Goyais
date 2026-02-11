# Security Policy

## 1. 支持版本 | Supported Versions

| Version | Supported |
|---|---|
| `v0.1.x` | ✅ |
| `< v0.1` | ❌ |

## 2. 漏洞报告方式 | How to Report a Vulnerability

请不要在公开 Issue 中直接披露漏洞细节。  
Please do not disclose vulnerability details in public issues.

推荐流程 | Preferred process:
1. 使用 GitHub Private Vulnerability Reporting（若仓库已启用）。
2. 若不可用，请通过 GitHub 私信联系维护者并提供复现信息。

请尽量包含：
- 影响范围（受影响模块/版本）
- 复现步骤（PoC）
- 风险等级与可能影响
- 建议修复方向（可选）

## 3. 响应时效 | Response Timeline (Target)

- 72 小时内确认收到（acknowledgement）
- 7 个工作日内完成初步分级（triage）
- 修复与披露时间根据风险级别协商

## 4. 披露原则 | Disclosure Policy

- 修复发布前不公开可利用细节
- 修复后发布安全公告（如适用）
- 对报告者给予致谢（可匿名）

## 5. 安全基线 | Security Baseline

与仓库强约束保持一致：
- Command-first
- Agent-as-User
- Visibility + ACL
- Egress Gate
- 全链路审计
