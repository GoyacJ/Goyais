# Rule 09: Source Header and Comments

## Trigger Conditions

- 任意源码文件新增或修改时（`*.go`, `*.ts`, `*.vue`, `*.js`, `*.py`, `*.java`, `*.dart`）。

## Hard Constraints (MUST)

- 每个源码文件必须具备标准文件头，字段顺序固定：
  1. `SPDX-License-Identifier: Apache-2.0`
  2. `Copyright (c) 2026 Goya`
  3. `Author: Goya`
  4. `Created: 2026-02-11`
  5. `Version: v1.0.0`
  6. `Description: <一句话职责>`
- 注释风格必须与语言一致：
  - Go/Dart: `//`
  - TS/JS/Java/Vue script: `/** ... */`
  - Python: `#`
- 实现内注释仅解释“为什么”，避免逐行翻译式注释。
- Java 代码必须为以下声明提供 JavaDoc（`/** ... */`）：
  - `public class/interface/enum/record`
  - `public` 方法与构造器
- CI 必须执行 `source_header_check.sh`，缺失或顺序错误直接失败。
- CI 必须执行 `java_javadoc_check.sh`，JavaDoc 缺失直接失败。

## Counterexamples

- 只有 `Copyright`，缺失 `SPDX/Author/Version`。
- 头信息存在但字段顺序错乱。
- 在普通流程中加入大量低价值注释噪音。

## Validation Commands

- `bash go_server/scripts/ci/source_header_check.sh`
- `bash java_server/scripts/ci/java_javadoc_check.sh`
- `bash go_server/scripts/ci/source_header_backfill.sh`
- `bash go_server/scripts/ci/contract_regression.sh`
