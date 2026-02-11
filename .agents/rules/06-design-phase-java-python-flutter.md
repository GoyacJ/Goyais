# Rule 06: Design Phase Gates (Java/Python/Flutter)

## Trigger Conditions

- `java_server`、`python_server`、`flutter_mobile` 需求推进。

## Hard Constraints (MUST)

- 先设计后编码：接口草案、数据模型、DoD、验收矩阵、回滚策略。
- 与 `docs/prd.md` 与 Go 契约语义对齐。
- 未过设计门禁不得进入实现阶段。

## Counterexamples

- 无契约草案直接搭框架。
- 跨端错误语义不一致。

## Validation Commands

- `rg -n 'Design Gate|DoD|验收' java_server python_server flutter_mobile -S`
