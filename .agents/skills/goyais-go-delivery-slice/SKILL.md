---
name: goyais-go-delivery-slice
description: Go 端垂直切片交付模板，覆盖目标/范围/约束/测试/验收/回滚。
---

# goyais-go-delivery-slice

## 适用场景

- Go API、domain、repository、workflow 等模块实施。

## 输入

- `docs/prd.md`
- `go_server/docs/api/openapi.yaml`
- `go_server/docs/arch/*`
- `go_server/docs/acceptance.md`

## 输出

- 切片实现清单。
- DoD 与回滚策略。

## 严格步骤

1. 先定义验收终点，再定义实现。
2. 对齐错误模型与 Command-first。
3. 同步契约文档。
4. 运行回归命令并记录证据。

## 验收

- 代码、契约、验收三者一致。
