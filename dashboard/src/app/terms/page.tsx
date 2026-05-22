"use client";

import Link from "next/link";
import { TermsContent } from "@/components/legal/terms-content";
import { buttonVariants } from "@/components/ui/button";
import { cn } from "@/lib/utils";

export default function TermsPage() {
  return (
    <div className="min-h-screen bg-slate-50/80">
      <header className="border-b border-slate-200 bg-white/90 px-4 py-3">
        <div className="mx-auto flex max-w-3xl items-center justify-between">
          <Link href="/" className="text-sm font-semibold text-slate-900">
            Sub2API
          </Link>
          <Link href="/register" className={cn(buttonVariants({ variant: "outline", size: "sm" }))}>
            返回注册
          </Link>
        </div>
      </header>
      <main className="mx-auto max-w-3xl px-4 py-8">
        <h1 className="mb-6 text-2xl font-semibold text-slate-900">User Agreement &amp; Privacy Notice</h1>
        <TermsContent />
      </main>
    </div>
  );
}
