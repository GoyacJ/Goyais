import { enUSMessages } from "@/shared/i18n/messages.en-US";
import { zhCNMessages } from "@/shared/i18n/messages.zh-CN";
import type { MessageMap } from "@/shared/i18n/messages.types";

export type Locale = "zh-CN" | "en-US";

export const messages: Record<Locale, MessageMap> = {
  "zh-CN": zhCNMessages,
  "en-US": enUSMessages
};
