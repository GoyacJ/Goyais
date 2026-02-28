import { computed, onBeforeUnmount, ref, watch, type Ref } from "vue";

import {
  applyRunningActionElapsed,
  buildRunningActionBaseViewModels,
  type RunningActionViewModel
} from "@/modules/conversation/views/runningActions";
import type { ConversationRuntime } from "@/modules/conversation/store/state";
import type { Locale } from "@/shared/i18n/messages";
import type { Execution } from "@/shared/types/api";

type RunningActionsViewOptions = {
  locale: Ref<Locale>;
  executionFilter?: (execution: Execution) => boolean;
};

export function useRunningActionsView(
  runtime: Ref<ConversationRuntime | undefined>,
  options: RunningActionsViewOptions
) {
  const nowTick = ref(Date.now());
  let timer: ReturnType<typeof setInterval> | undefined;

  const hasRunningExecutions = computed(() =>
    (runtime.value?.executions ?? []).some(
      (execution) => execution.state === "pending" || execution.state === "executing" || execution.state === "confirming"
    )
  );

  const baseRunningActions = computed(() => {
    const currentRuntime = runtime.value;
    if (!currentRuntime) {
      return [];
    }
    const executions = typeof options.executionFilter === "function"
      ? currentRuntime.executions.filter((execution) => options.executionFilter?.(execution) ?? true)
      : currentRuntime.executions;
    return buildRunningActionBaseViewModels(currentRuntime.events, executions, options.locale.value);
  });

  const runningActions = computed<RunningActionViewModel[]>(() => {
    return applyRunningActionElapsed(baseRunningActions.value, options.locale.value, new Date(nowTick.value));
  });

  watch(
    hasRunningExecutions,
    (running) => {
      if (running) {
        if (!timer) {
          timer = setInterval(() => {
            nowTick.value = Date.now();
          }, 1000);
        }
        return;
      }
      if (timer) {
        clearInterval(timer);
        timer = undefined;
      }
    },
    { immediate: true }
  );

  onBeforeUnmount(() => {
    if (timer) {
      clearInterval(timer);
      timer = undefined;
    }
  });

  return {
    runningActions
  };
}
