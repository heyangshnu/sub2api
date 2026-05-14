"use client";

import { useAuth } from "@/lib/auth-context";
import { LoginForm } from "@/components/login-form";
import { Dashboard } from "@/components/dashboard";
import { Skeleton } from "@/components/ui/skeleton";

export default function Home() {
  const { isAuthenticated, isLoading } = useAuth();

  if (isLoading) {
    return (
      <div className="flex min-h-screen items-center justify-center px-4">
        <div className="w-full max-w-md space-y-6">
          <Skeleton className="mx-auto h-14 w-14 rounded-2xl bg-slate-200/80" />
          <div className="space-y-3 rounded-2xl border border-slate-200/90 bg-white/75 p-8 shadow-lg shadow-slate-200/40 ring-1 ring-slate-200/50 backdrop-blur-xl">
            <Skeleton className="h-5 w-24 rounded-md bg-slate-200/80" />
            <Skeleton className="h-11 w-full rounded-xl bg-slate-200/70" />
            <Skeleton className="h-11 w-full rounded-xl bg-slate-200/70" />
            <Skeleton className="h-11 w-full rounded-xl bg-slate-200/70" />
          </div>
        </div>
      </div>
    );
  }

  if (!isAuthenticated) {
    return <LoginForm />;
  }

  return <Dashboard />;
}
