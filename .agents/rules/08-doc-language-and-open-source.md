# Rule 08: Documentation Language & OSS Quality

## Trigger Conditions

- 新增/修改治理文档、rules、skills、技术方案文档。

## Hard Constraints (MUST)

- 中英双语风格：中文主叙述 + 英文术语/命令。
- 表达可审计、可复现、可移交。
- 术语统一，避免同义词漂移。

## Counterexamples

- 文档只有结论没有验证命令。
- 同一概念在不同文档使用冲突术语。

## Validation Commands

- `rg -n 'MUST|Trigger Conditions|Validation Commands|DoD' .agents/rules .agents/skills -S`
