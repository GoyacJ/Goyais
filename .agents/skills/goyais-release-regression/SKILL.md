---
name: goyais-release-regression
description: 统一执行仓库回归门禁与 single-binary 验收脚本。
---

# goyais-release-regression

## 适用场景

- 合并前、发布前、契约变更后回归。

## 输入

- `go_server/scripts/ci/contract_regression.sh`
- `scripts/verify_single_binary.sh`

## 输出

- 回归日志。
- 失败定位与修复建议。

## 严格步骤

1. 执行 `contract_regression.sh`。
2. 检查 single-binary 关键断言（no-store、404、Content-Type）。

## 验收

- 脚本退出码为 0。
