"use client";

import { useLocale, type Locale } from "@/lib/i18n";
import { cn } from "@/lib/utils";

export function LocaleToggle({ className }: { className?: string }) {
  const { locale, setLocale, t } = useLocale();

  const btn = (code: Locale, label: string) => (
    <button
      type="button"
      onClick={() => setLocale(code)}
      className={cn(
        "rounded-md px-2.5 py-1 text-xs font-medium transition-colors",
        locale === code
          ? "bg-teal-600 text-white shadow-sm"
          : "text-slate-600 hover:bg-slate-100 hover:text-slate-900"
      )}
      aria-pressed={locale === code}
    >
      {label}
    </button>
  );

  return (
    <div
      className={cn(
        "inline-flex items-center gap-0.5 rounded-lg border border-slate-200/90 bg-white/80 p-0.5",
        className
      )}
      role="group"
      aria-label="Language"
    >
      {btn("en", t("locale.en"))}
      {btn("zh", t("locale.zh"))}
    </div>
  );
}
