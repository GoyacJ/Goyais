import type { SupportedLocale } from "@/i18n/types";

export function formatTimeByLocale(isoTs: string, locale: SupportedLocale): string {
  const ts = new Date(isoTs);
  if (Number.isNaN(ts.getTime())) {
    return "";
  }

  return new Intl.DateTimeFormat(locale, {
    hour: "2-digit",
    minute: "2-digit",
    second: "2-digit"
  }).format(ts);
}
