---
name: goyais-java-delivery-slice
description: Java 端垂直切片交付模板，覆盖单应用拓扑、动态权限、数据权限与回归门禁。
---

# goyais-java-delivery-slice

## 适用场景

- Java API、security、data permission、capability provider 的实现任务。

## 输入

- `docs/prd.md`
- `go_server/docs/api/openapi.yaml`
- `java_server/docs/*`
- `.agents/rules/*`

## 输出

- 可落地切片实现清单（代码 + 文档 + 回归证据）。
- DoD 与回滚策略。

## 严格步骤

1. 固定契约：确认 `/api/v1`、错误模型、分页语义不漂移。
2. 固定拓扑：默认 `single`，并保证 `resource-only` 可切换。
3. 固定权限：`policyVersion + Redis invalidation` 与 SQL 行级权限同时收敛。
4. 固定注释门禁：Java 文件头采用 `SPDX + <p> + @author + @since(yyyy-MM-dd HH:mm:ss)`，并为 `public/protected` type/method/ctor/field 提供符合 JDK/Javadoc 标准的注释（`@param/@return/@throws`）。
5. 运行回归并记录证据命令。

## 验收

- `mvn -f java_server/pom.xml -DskipTests verify`
- `mvn -f java_server/pom.xml test`
- `bash java_server/scripts/ci/java_javadoc_check.sh`
