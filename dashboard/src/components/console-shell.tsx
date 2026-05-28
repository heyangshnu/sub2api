"use client";

import Link from "next/link";
import { usePathname } from "next/navigation";
import { useState } from "react";
import {
  BarChart3,
  CreditCard,
  KeyRound,
  Menu,
  MessageSquare,
  Receipt,
  ScrollText,
  Sparkles,
  Wallet,
  X,
} from "lucide-react";
import { ConsoleSloganLayout } from "@/components/brand/console-slogan";
import { LocaleToggle } from "@/components/locale-toggle";
import { AuthDialog } from "@/components/auth-dialog";
import { useAuth } from "@/lib/auth-context";
import { useLocale, usePageTitle } from "@/lib/i18n";
import { Button, buttonVariants } from "@/components/ui/button";
import { cn } from "@/lib/utils";

const NAV = [
  { href: "/chat", key: "nav.chat", icon: MessageSquare },
  { href: "/", key: "nav.usage", icon: BarChart3 },
  { href: "/keys", key: "nav.keys", icon: KeyRound },
  { href: "/topup", key: "nav.topup", icon: Wallet },
  { href: "/subscription", key: "nav.subscription", icon: Sparkles },
  { href: "/billing", key: "nav.billing", icon: Receipt },
  { href: "/logs", key: "nav.logs", icon: ScrollText },
] as const;

function NavLinks({ onNavigate }: { onNavigate?: () => void }) {
  const pathname = usePathname();
  const { t } = useLocale();

  return (
    <nav className="flex flex-col gap-0.5 px-3">
      {NAV.map((item) => {
        const Icon = item.icon;
        const active =
          item.href === "/"
            ? pathname === "/"
            : pathname === item.href || pathname.startsWith(item.href + "/");
        return (
          <Link
            key={item.href}
            href={item.href}
            onClick={onNavigate}
            className={cn(
              "flex items-center gap-3 rounded-xl px-3 py-2.5 text-sm font-medium transition-colors",
              active
                ? "bg-teal-500/12 text-teal-900 shadow-sm ring-1 ring-teal-500/20"
                : "text-slate-600 hover:bg-slate-100/90 hover:text-slate-900"
            )}
          >
            <Icon
              className={cn("size-[18px] shrink-0", active ? "text-teal-600" : "text-slate-400")}
              strokeWidth={active ? 2.25 : 2}
            />
            {t(item.key)}
          </Link>
        );
      })}
    </nav>
  );
}

export function ConsoleShell({ children }: { children: React.ReactNode }) {
  const pathname = usePathname();
  const pageTitle = usePageTitle(pathname);
  const { t } = useLocale();
  const { isAuthenticated, isGuest, user, logout, openAuthDialog } = useAuth();
  const [mobileOpen, setMobileOpen] = useState(false);
  const isChatRoute = pathname === "/chat";

  const sidebar = (
    <>
      <div className="flex h-14 items-center gap-2 border-b border-slate-200/80 px-4">
        <div className="flex size-8 items-center justify-center rounded-lg bg-gradient-to-br from-teal-500 to-teal-600 text-white shadow-sm">
          <CreditCard className="size-4" strokeWidth={2.25} />
        </div>
        <Link href="/" className="text-sm font-semibold tracking-tight text-slate-900" onClick={() => setMobileOpen(false)}>
          {t("brand.name")}
        </Link>
      </div>

      <div className="flex-1 overflow-y-auto py-4">
        <NavLinks onNavigate={() => setMobileOpen(false)} />
      </div>
    </>
  );

  return (
    <>
      <AuthDialog />
      <div className="console-layout flex h-screen max-h-[100dvh] overflow-hidden">
        {/* Desktop sidebar */}
        <aside className="console-sidebar hidden h-full w-[var(--console-sidebar-w)] shrink-0 flex-col border-r border-slate-200/90 bg-white/70 backdrop-blur-xl md:flex">
          {sidebar}
        </aside>

        {/* Mobile drawer */}
        {mobileOpen ? (
          <button
            type="button"
            className="fixed inset-0 z-40 bg-slate-900/30 backdrop-blur-[2px] md:hidden"
            aria-label="Close menu"
            onClick={() => setMobileOpen(false)}
          />
        ) : null}
        <aside
          className={cn(
            "console-sidebar fixed inset-y-0 left-0 z-50 flex w-[min(100%,var(--console-sidebar-w))] flex-col border-r border-slate-200/90 bg-white/95 shadow-xl backdrop-blur-xl transition-transform duration-200 md:hidden",
            mobileOpen ? "translate-x-0" : "-translate-x-full"
          )}
        >
          <button
            type="button"
            className="absolute right-3 top-3 rounded-lg p-1.5 text-slate-500 hover:bg-slate-100"
            onClick={() => setMobileOpen(false)}
            aria-label="Close"
          >
            <X className="size-5" />
          </button>
          {sidebar}
        </aside>

        <ConsoleSloganLayout
          pageTitle={pageTitle}
          mainClassName={
            isChatRoute && !isGuest
              ? "overflow-hidden p-0 md:p-0 lg:p-0"
              : undefined
          }
          headerLeft={
            <button
              type="button"
              className="rounded-lg p-2 text-slate-600 hover:bg-slate-100 md:hidden"
              onClick={() => setMobileOpen(true)}
              aria-label="Open menu"
            >
              <Menu className="size-5" />
            </button>
          }
          headerRight={
            <div className="flex items-center gap-2">
              <LocaleToggle />
              {isAuthenticated && user ? (
                <>
                  <Link
                    href="/profile"
                    className={cn(
                      buttonVariants({ variant: "outline", size: "sm" }),
                      "max-w-[200px] truncate border-slate-200 text-slate-700"
                    )}
                  >
                    {user.email}
                  </Link>
                  <Button type="button" variant="ghost" size="sm" onClick={logout}>
                    {t("shell.signOut")}
                  </Button>
                </>
              ) : (
                <>
                  <Button
                    type="button"
                    variant="outline"
                    size="sm"
                    className="border-slate-200"
                    onClick={() => openAuthDialog("register")}
                  >
                    {t("shell.signUp")}
                  </Button>
                  <Button
                    type="button"
                    size="sm"
                    className="bg-teal-600 text-white hover:bg-teal-500"
                    onClick={() => openAuthDialog("login")}
                  >
                    {t("shell.signIn")}
                  </Button>
                </>
              )}
            </div>
          }
        >
          {children}
          {isGuest && !isChatRoute ? (
            <p className="mt-6 text-center text-xs text-slate-500">{t("shell.guestHint")}</p>
          ) : null}
        </ConsoleSloganLayout>
      </div>
    </>
  );
}
