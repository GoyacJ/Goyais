import { ref, watch, type ComputedRef, type Ref } from "vue";

import type { ConversationRuntime } from "@/modules/conversation/store/state";
import type { Conversation } from "@/shared/types/api";

type ModelOption = { value: string; label: string };

type RiskConfirmState = {
  open: boolean;
  executionId: string;
  summary: string;
  preview: string;
};

type AutoModelSyncWatcherInput = {
  activeConversation: ComputedRef<Conversation | undefined>;
  activeCount: ComputedRef<number>;
  modelOptions: ComputedRef<ModelOption[]>;
  resolveSemanticModelID: (raw: string) => string;
  runtime: ComputedRef<ConversationRuntime | undefined>;
  updateModel: (modelID: string) => Promise<void>;
};

type RiskConfirmWatcherInput = {
  runtime: ComputedRef<ConversationRuntime | undefined>;
  riskConfirm: Ref<RiskConfirmState>;
};

export function useAutoModelSyncWatcher(input: AutoModelSyncWatcherInput): void {
  const autoModelSyncingConversationID = ref("");

  watch(
    [() => input.activeConversation.value?.id ?? "", () => input.activeCount.value, () => input.modelOptions.value],
    async ([conversationID, activeExecutionCount, options]) => {
      if (conversationID === "" || activeExecutionCount > 0 || options.length === 0) {
        return;
      }
      if (autoModelSyncingConversationID.value === conversationID) {
        return;
      }

      const currentModelID = input.resolveSemanticModelID(
        input.runtime.value?.modelId ?? input.activeConversation.value?.model_id ?? ""
      );
      if (currentModelID !== "" && options.some((item) => item.value === currentModelID)) {
        return;
      }

      const targetModelID = options[0]?.value ?? "";
      if (targetModelID === "") {
        return;
      }

      autoModelSyncingConversationID.value = conversationID;
      try {
        await input.updateModel(targetModelID);
      } finally {
        autoModelSyncingConversationID.value = "";
      }
    },
    { deep: true }
  );
}

export function useRiskConfirmWatcher(input: RiskConfirmWatcherInput): void {
  watch(
    () => input.runtime.value?.events.length ?? 0,
    () => {
      const events = input.runtime.value?.events ?? [];
      const latest = events[events.length - 1];
      if (!latest) {
        return;
      }

      if (latest.type === "confirmation_required") {
        input.riskConfirm.value = {
          open: true,
          executionId: latest.execution_id,
          summary: typeof latest.payload.summary === "string" ? latest.payload.summary : "高风险操作需要确认",
          preview: typeof latest.payload.preview === "string" ? latest.payload.preview : ""
        };
        return;
      }

      if (
        latest.type === "confirmation_resolved" ||
        latest.type === "execution_done" ||
        latest.type === "execution_error" ||
        latest.type === "execution_stopped"
      ) {
        if (latest.execution_id === input.riskConfirm.value.executionId) {
          input.riskConfirm.value.open = false;
          input.riskConfirm.value.executionId = "";
        }
      }
    }
  );
}
