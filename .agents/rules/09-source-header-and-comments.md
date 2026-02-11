# Rule 09: Source Header and Comments

## Trigger Conditions

- 任意源码文件新增或修改时（`*.go`, `*.ts`, `*.vue`, `*.js`, `*.py`, `*.java`, `*.dart`）。

## Hard Constraints (MUST)

- 非 Java 源码文件必须具备标准文件头，字段顺序固定：
  1. `SPDX-License-Identifier: Apache-2.0`
  2. `Copyright (c) 2026 Goya`
  3. `Author: Goya`
  4. `Created: 2026-02-11`
  5. `Version: v1.0.0`
  6. `Description: <一句话职责>`
- Java 源码文件必须具备如下头注释（首个块注释）：
  1. `SPDX-License-Identifier: Apache-2.0`
  2. `<p>...</p>`
  3. `@author Goya`
  4. `@since YYYY-MM-DD HH:MM:SS`
- 注释风格必须与语言一致：
  - Go/Dart: `//`
  - TS/JS/Java/Vue script: `/** ... */`
  - Python: `#`
- 实现内注释仅解释“为什么”，避免逐行翻译式注释。
- Java 代码必须为以下声明提供 JavaDoc（`/** ... */`）：
  - `public/protected class/interface/enum/record`
  - `public/protected` 方法与构造器
  - `public/protected` 字段
- Java 方法与构造器遵循 JDK/Javadoc 标准：
  - 每个参数对应 `@param`
  - 非 `void` 方法必须有 `@return`
  - `throws` 子句中的每个异常对应 `@throws`
- CI 必须执行 `source_header_check.sh`，缺失或顺序错误直接失败。
- CI 必须执行 `java_javadoc_check.sh`，JavaDoc/标签缺失直接失败。

## Counterexamples

- Java 头注释缺失 `@since` 或时间格式错误。
- Java 方法缺失 `@param/@return/@throws`。
- 头信息存在但字段顺序错乱。
- 在普通流程中加入大量低价值注释噪音。

## Validation Commands

- `bash go_server/scripts/ci/source_header_check.sh`
- `bash java_server/scripts/ci/java_javadoc_check.sh`
- `bash go_server/scripts/ci/source_header_backfill.sh`
- `bash go_server/scripts/ci/contract_regression.sh`
