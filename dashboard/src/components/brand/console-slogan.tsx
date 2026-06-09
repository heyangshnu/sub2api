"use client";

import { useAuth } from "@/lib/auth-context";
import { ct } from "@/lib/console-typography";
import { useLocale } from "@/lib/i18n";
import { cn } from "@/lib/utils";

export function ConsoleSloganLayout({
  pageTitle,
  headerLeft,
  headerRight,
  mainClassName,
  children,
}: {
  pageTitle: string;
  headerLeft?: React.ReactNode;
  headerRight: React.ReactNode;
  mainClassName?: string;
  children: React.ReactNode;
}) {
  const { messages } = useLocale();
  const { isAuthenticated } = useAuth();
  const sloganLine = messages.slogan.words.join(" · ");
  const sub = messages.slogan.sub;

  return (
    <div className="flex min-h-0 min-w-0 flex-1 flex-col">
      <header className="sticky top-0 z-30 border-b border-slate-200/80 bg-white/75 backdrop-blur-xl">
        <div className="flex h-14 items-center justify-between gap-3 px-4 md:px-6 lg:px-8">
          <div className="relative flex min-w-0 flex-1 items-center gap-3">
            {headerLeft}
            {isAuthenticated ? (
              <p className={cn(ct.panelTitle, "min-w-0 flex-1 truncate text-teal-800")} aria-live="polite">
                <span className="font-semibold">{sloganLine}</span>
                <span className="mx-2 hidden font-normal text-slate-400 sm:inline">·</span>
                <span className={cn("hidden font-normal sm:inline", ct.panelDesc, "mt-0 inline text-slate-500")}>
                  {sub}
                </span>
              </p>
            ) : (
              <h1 className="min-w-0 truncate text-base font-semibold text-slate-900">{pageTitle}</h1>
            )}
          </div>
          {headerRight}
        </div>
      </header>

      <main
        className={cn(
          "console-main flex min-h-0 flex-1 flex-col px-4 py-6 md:px-6 lg:px-8 lg:py-8",
          mainClassName
        )}
      >
        {children}
      </main>
    </div>
  );
}
