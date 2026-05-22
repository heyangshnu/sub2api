"use client";

import Link from "next/link";
import { usePathname } from "next/navigation";
import { useAuth } from "@/lib/auth-context";
import { Button, buttonVariants } from "@/components/ui/button";
import { cn } from "@/lib/utils";
import { AuthDialog } from "@/components/auth-dialog";

const NAV = [
  { href: "/chat", label: "Chat" },
  { href: "/", label: "Usage" },
  { href: "/keys", label: "API Keys" },
  { href: "/topup", label: "Top-up" },
  { href: "/subscription", label: "Subscription" },
  { href: "/billing", label: "Billing" },
] as const;

export function ConsoleShell({ children }: { children: React.ReactNode }) {
  const pathname = usePathname();
  const { isAuthenticated, isGuest, user, logout, openAuthDialog } = useAuth();

  return (
    <>
      <AuthDialog />
      <div className="flex min-h-screen flex-col">
        <header className="sticky top-0 z-40 border-b border-slate-200/80 bg-white/80 backdrop-blur-xl">
          <div className="mx-auto flex max-w-6xl flex-wrap items-center justify-between gap-3 px-4 py-3 md:px-6">
            <div className="flex flex-wrap items-center gap-4">
              <Link href="/" className="text-sm font-semibold text-slate-900">
                Sub2API
              </Link>
              <nav className="flex flex-wrap items-center gap-1">
                {NAV.map((item) => {
                  const active =
                    item.href === "/"
                      ? pathname === "/"
                      : pathname === item.href || pathname.startsWith(item.href + "/");
                  return (
                    <Link
                      key={item.href}
                      href={item.href}
                      className={cn(
                        buttonVariants({ variant: "ghost", size: "sm" }),
                        "text-sm",
                        active
                          ? "bg-slate-100 text-slate-900"
                          : "text-slate-600 hover:bg-slate-50 hover:text-slate-900"
                      )}
                    >
                      {item.label}
                    </Link>
                  );
                })}
              </nav>
            </div>
            <div className="flex items-center gap-2">
              {isAuthenticated && user ? (
                <>
                  <Link
                    href="/profile"
                    className={cn(
                      buttonVariants({ variant: "outline", size: "sm" }),
                      "border-slate-200 text-slate-700"
                    )}
                  >
                    {user.email}
                  </Link>
                  <Button type="button" variant="ghost" size="sm" onClick={logout}>
                    Sign out
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
                    Sign up
                  </Button>
                  <Button
                    type="button"
                    size="sm"
                    className="bg-slate-900 text-white hover:bg-slate-800"
                    onClick={() => openAuthDialog("login")}
                  >
                    Sign in
                  </Button>
                </>
              )}
            </div>
          </div>
        </header>
        <main className="mx-auto w-full max-w-6xl flex-1 px-4 py-6 md:px-6">{children}</main>
        {isGuest ? (
          <p className="mx-auto max-w-6xl px-4 pb-4 text-center text-xs text-slate-500 md:px-6">
            You can browse all sections while signed out. Sign in to perform actions.
          </p>
        ) : null}
      </div>
    </>
  );
}
