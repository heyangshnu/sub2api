"use client";

import Link from "next/link";
import { useAuth } from "@/lib/auth-context";
import { Button, buttonVariants } from "@/components/ui/button";
import { cn } from "@/lib/utils";

export function AppShell({ children }: { children: React.ReactNode }) {
  const { logout } = useAuth();

  return (
    <div className="flex min-h-screen flex-col">
      <header className="border-b border-slate-200/80 bg-white/70 backdrop-blur-xl">
        <div className="flex w-full items-center justify-between gap-4 px-4 py-3 md:px-6">
          <Link href="/" className="text-sm font-medium text-slate-900">
            Sub2API
          </Link>
          <div className="flex items-center gap-3">
            <Link href="/" className={cn(buttonVariants({ variant: "outline", size: "sm" }), "border-slate-200")}>
              Home
            </Link>
            <Link href="/account" className={cn(buttonVariants({ variant: "outline", size: "sm" }), "border-slate-200")}>
              Account
            </Link>
            <Link href="/profile" className={cn(buttonVariants({ variant: "outline", size: "sm" }), "border-slate-200")}>
              Profile
            </Link>
            <Button type="button" variant="ghost" size="sm" onClick={logout}>
              Sign out
            </Button>
          </div>
        </div>
      </header>
      <main className="flex min-h-0 flex-1 flex-col">{children}</main>
    </div>
  );
}
