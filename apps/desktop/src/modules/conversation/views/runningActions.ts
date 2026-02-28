import {
  buildRunningActionBaseViewModelData,
  hydrateRunningActionElapsed
} from "@/modules/conversation/trace/present";
import type {
  RunningActionBaseViewModel,
  RunningActionType,
  RunningActionViewModelData,
  TraceLocale
} from "@/modules/conversation/trace/types";
import type { Execution, ExecutionEvent } from "@/shared/types/api";

export type RunningActionViewModel = RunningActionViewModelData;
export type RunningActionBase = RunningActionBaseViewModel;
export type { RunningActionType };

export function buildRunningActionBaseViewModels(
  events: ExecutionEvent[],
  executions: Execution[],
  locale: TraceLocale
): RunningActionBase[] {
  return buildRunningActionBaseViewModelData(events, executions, locale);
}

export function applyRunningActionElapsed(
  actions: RunningActionBase[],
  locale: TraceLocale,
  now: Date = new Date()
): RunningActionViewModel[] {
  return hydrateRunningActionElapsed(actions, locale, now);
}

export function buildRunningActionViewModels(
  events: ExecutionEvent[],
  executions: Execution[],
  locale: TraceLocale,
  now: Date = new Date()
): RunningActionViewModel[] {
  return applyRunningActionElapsed(buildRunningActionBaseViewModels(events, executions, locale), locale, now);
}
