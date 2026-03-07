import { buildRunTraceViewModelData } from "@/modules/session/trace/present";
import type { RunTraceStepViewModel, RunTraceViewModelData, TraceLocale } from "@/modules/session/trace/types";
import type { Run, RunLifecycleEvent } from "@/shared/types/api";

export type RunTraceStep = RunTraceStepViewModel;
export type RunTraceViewModel = RunTraceViewModelData;

export function buildRunTraceViewModels(
  events: RunLifecycleEvent[],
  executions: Run[],
  locale: TraceLocale,
  now: Date = new Date()
): RunTraceViewModel[] {
  return buildRunTraceViewModelData(events, executions, locale, now);
}
