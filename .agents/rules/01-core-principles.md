# Rule 01: Core Principles

## Trigger Conditions

- 涉及副作用动作、授权、共享、外发、可见性时。

## Hard Constraints (MUST)

- 副作用必须经 Command（Command-first）。
- AI 仅以当前用户身份执行（Agent-as-User）。
- 统一 visibility/ACL。
- 外发必须经 Egress Gate 且可审计。

## Counterexamples

- AI 直接调用内部 service 执行写操作。
- 在 step 执行前省略授权复检。

## Validation Commands

- `rg -n 'POST /api/v1/commands|Command-first|Agent-as-User' AGENTS.md go_server/docs -S`
