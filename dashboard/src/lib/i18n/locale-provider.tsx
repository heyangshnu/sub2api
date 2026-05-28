"use client";

import { createContext, useCallback, useContext, useEffect, useMemo, useState } from "react";
import { en, type Messages } from "./locales/en";
import { zh } from "./locales/zh";

export type Locale = "en" | "zh";

const LOCALE_KEY = "sub2api_locale";

const dictionaries: Record<Locale, Messages> = { en, zh };

function getInitialLocale(): Locale {
  if (typeof window === "undefined") return "en";
  const saved = localStorage.getItem(LOCALE_KEY);
  if (saved === "en" || saved === "zh") return saved;
  return "en";
}

function getByPath(obj: Messages, path: string): string | readonly string[] | undefined {
  const parts = path.split(".");
  let cur: unknown = obj;
  for (const p of parts) {
    if (cur == null || typeof cur !== "object") return undefined;
    cur = (cur as Record<string, unknown>)[p];
  }
  return cur as string | readonly string[] | undefined;
}

export function interpolate(
  template: string,
  vars?: Record<string, string | number>
): string {
  if (!vars) return template;
  return template.replace(/\{\{(\w+)\}\}/g, (_, key: string) =>
    vars[key] != null ? String(vars[key]) : `{{${key}}}`
  );
}

type LocaleContextValue = {
  locale: Locale;
  setLocale: (locale: Locale) => void;
  t: (key: string, vars?: Record<string, string | number>) => string;
  messages: Messages;
};

const LocaleContext = createContext<LocaleContextValue | null>(null);

export function LocaleProvider({ children }: { children: React.ReactNode }) {
  const [locale, setLocaleState] = useState<Locale>("en");
  const [ready, setReady] = useState(false);

  useEffect(() => {
    setLocaleState(getInitialLocale());
    setReady(true);
  }, []);

  useEffect(() => {
    if (!ready) return;
    localStorage.setItem(LOCALE_KEY, locale);
    document.documentElement.lang = locale === "zh" ? "zh-CN" : "en";
    document.title = dictionaries[locale].meta.title;
  }, [locale, ready]);

  const setLocale = useCallback((next: Locale) => {
    setLocaleState(next);
  }, []);

  const messages = dictionaries[locale];

  const t = useCallback(
    (key: string, vars?: Record<string, string | number>) => {
      const raw = getByPath(messages, key);
      if (typeof raw === "string") return interpolate(raw, vars);
      return key;
    },
    [messages]
  );

  const value = useMemo(
    () => ({ locale, setLocale, t, messages }),
    [locale, setLocale, t, messages]
  );

  return <LocaleContext.Provider value={value}>{children}</LocaleContext.Provider>;
}

export function useLocale() {
  const ctx = useContext(LocaleContext);
  if (!ctx) throw new Error("useLocale must be used within LocaleProvider");
  return ctx;
}

export function useT() {
  return useLocale().t;
}

export function usePageTitle(pathname: string): string {
  const { t } = useLocale();
  const map: Record<string, string> = {
    "/": "pages.usage",
    "/chat": "pages.chat",
    "/keys": "pages.keys",
    "/topup": "pages.topup",
    "/subscription": "pages.subscription",
    "/billing": "pages.billing",
    "/profile": "pages.profile",
    "/logs": "pages.logs",
    "/account/logs": "pages.logs",
    "/terms": "pages.terms",
  };
  const key = map[pathname] ?? "pages.usage";
  return t(key);
}
