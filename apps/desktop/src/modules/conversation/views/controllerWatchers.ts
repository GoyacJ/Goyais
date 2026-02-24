import { ref, watch, type ComputedRef } from "vue";

import type { ConversationRuntime } from "@/modules/conversation/store/state";
import type { Conversation } from "@/shared/types/api";

type ModelOption = { value: string; label: string };

type AutoModelSyncWatcherInput = {
  activeConversation: ComputedRef<Conversation | undefined>;
  activeCount: ComputedRef<number>;
  modelOptions: ComputedRef<ModelOption[]>;
  resolveSemanticModelID: (raw: string) => string;
  runtime: ComputedRef<ConversationRuntime | undefined>;
  updateModel: (modelID: string) => Promise<void>;
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
