"use client";

import Link from "next/link";
import { LocaleToggle } from "@/components/locale-toggle";
import { TermsContent } from "@/components/legal/terms-content";
import { useT } from "@/lib/i18n";
import { buttonVariants } from "@/components/ui/button";
import { cn } from "@/lib/utils";

export default function TermsPage() {
  const t = useT();

  return (
    <div className="min-h-screen bg-slate-50/80">
      <header className="border-b border-slate-200 bg-white/90 px-4 py-3">
        <div className="mx-auto flex max-w-3xl items-center justify-between gap-3">
          <Link href="/" className="text-sm font-semibold text-slate-900">
            {t("brand.name")}
          </Link>
          <div className="flex items-center gap-2">
            <LocaleToggle />
            <Link href="/register" className={cn(buttonVariants({ variant: "outline", size: "sm" }))}>
              {t("terms.backRegister")}
            </Link>
          </div>
        </div>
      </header>
      <main className="mx-auto max-w-3xl px-4 py-8">
        <h1 className="mb-6 text-2xl font-semibold text-slate-900">{t("terms.pageTitle")}</h1>
        <TermsContent />
      </main>
    </div>
  );
}
