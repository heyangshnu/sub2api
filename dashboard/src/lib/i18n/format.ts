import type { Locale } from "./locale-provider";

export function formatLocaleDateTime(dateStr: string, locale: Locale): string {
  return new Date(dateStr).toLocaleString(locale === "zh" ? "zh-CN" : "en-US");
}

export function formatLocaleDate(dateStr: string, locale: Locale): string {
  return new Date(dateStr).toLocaleDateString(locale === "zh" ? "zh-CN" : "en-US");
}
