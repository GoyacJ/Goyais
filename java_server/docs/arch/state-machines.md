# Java Server State Machines Draft (v0.1)

## 1. Command

- States: `accepted -> running -> succeeded|failed|canceled`
- Required audits: `command.authorize`, `command.execute`, `command.egress`

## 2. WorkflowRun

- States: `pending -> running -> succeeded|failed|canceled`
- Retry creates new run with `retry_of_run_id`

## 3. StepRun

- States: `pending -> running -> succeeded|failed|canceled|skipped`

## 4. PluginInstall

- States: `uploaded -> validating -> installing -> enabled|failed`
- Additional: `disabled`, `rolled_back`

## 5. Stream Recording

- Stream states: `offline|online|recording|error`
- Recording states: `starting|recording|stopping|succeeded|failed|canceled`

## 6. AI Session

- Session states: `active|archived`
- Turn via command types: `ai.intent.plan` / `ai.command.execute`
