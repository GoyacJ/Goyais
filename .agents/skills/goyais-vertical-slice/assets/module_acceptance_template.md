# 模块验收模板

## 用例矩阵

| 编号 | 场景 | 输入 | 预期输出 | 结果 |
|---|---|---|---|---|
| 1 | 正向主路径 |  |  |  |
| 2 | 权限拒绝路径 |  |  |  |
| 3 | 参数非法路径 |  |  |  |
| 4 | 边界条件 |  |  |  |

## Header/协议检查（如适用）

- [ ] Content-Type 正确
- [ ] Cache-Control 符合约束
- [ ] 错误结构为 `error: { code, messageKey, details }`

## 文档同步检查

- [ ] openapi
- [ ] data-model
- [ ] state-machines
- [ ] overview
- [ ] acceptance

## 结论

- 是否通过：
- 阻断问题：
- 后续动作：
