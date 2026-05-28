"use client";

import { useAuth } from "@/lib/auth-context";
import { ct } from "@/lib/console-typography";
import { useT } from "@/lib/i18n";
import { ApiKeysCard } from "@/components/api-keys-card";
import { Button } from "@/components/ui/button";
import { PanelCard } from "@/components/ui/panel-card";
import { cn } from "@/lib/utils";

export function KeysPage() {
  const t = useT();
  const { isGuest, openAuthDialog } = useAuth();

  if (isGuest) {
    return (
      <div className="mx-auto max-w-5xl space-y-5">
        <p className={ct.pageDesc}>{t("keys.guestDesc")}</p>
        <PanelCard
          title={t("keys.baseUrl")}
          description={`${process.env.NEXT_PUBLIC_API_URL || "https://api.cloudtoken.uk"}/v1`}
        >
          <p className={cn(ct.alert, "mb-4")}>{t("keys.guestBody")}</p>
          <Button type="button" className="bg-teal-600 text-sm hover:bg-teal-500" onClick={() => openAuthDialog("login")}>
            {t("keys.signInManage")}
          </Button>
        </PanelCard>
      </div>
    );
  }

  return (
    <div className="mx-auto max-w-5xl">
      <ApiKeysCard />
    </div>
  );
}
