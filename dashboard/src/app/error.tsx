"use client";

import { useEffect } from "react";
import { LocaleToggle } from "@/components/locale-toggle";
import { useT } from "@/lib/i18n";
import { Button } from "@/components/ui/button";

export default function Error({
  error,
  reset,
}: {
  error: Error & { digest?: string };
  reset: () => void;
}) {
  const t = useT();

  useEffect(() => {
    console.error("Dashboard route error:", error);
  }, [error]);

  return (
    <div className="relative flex min-h-screen flex-col items-center justify-center gap-4 px-4 text-center">
      <div className="absolute right-4 top-4">
        <LocaleToggle />
      </div>
      <p className="text-lg font-semibold text-slate-900">{t("error.title")}</p>
      <p className="max-w-md text-sm text-slate-600">
        {error.message || t("error.fallback")}
      </p>
      <div className="flex gap-3">
        <Button type="button" className="bg-teal-600 hover:bg-teal-500" onClick={() => reset()}>
          {t("error.tryAgain")}
        </Button>
        <Button type="button" variant="outline" onClick={() => window.location.reload()}>
          {t("error.reload")}
        </Button>
      </div>
    </div>
  );
}
