import {
  buildRunningActionBaseViewModelData,
  hydrateRunningActionElapsed
} from "@/modules/session/trace/present";
import type {
  RunningActionBaseViewModel,
  RunningActionType,
  RunningActionViewModelData,
  TraceLocale
} from "@/modules/session/trace/types";
import type { Run, RunLifecycleEvent } from "@/shared/types/api";

export type RunningActionViewModel = RunningActionViewModelData;
export type RunningActionBase = RunningActionBaseViewModel;
export type { RunningActionType };

export function buildRunningActionBaseViewModels(
  events: RunLifecycleEvent[],
  executions: Run[],
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
  events: RunLifecycleEvent[],
  executions: Run[],
  locale: TraceLocale,
  now: Date = new Date()
): RunningActionViewModel[] {
  return applyRunningActionElapsed(buildRunningActionBaseViewModels(events, executions, locale), locale, now);
}
