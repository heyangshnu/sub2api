"use client";

import { useAuth } from "@/lib/auth-context";
import { useT } from "@/lib/i18n";
import { ChatPage } from "@/components/chat/chat-page";
import { Button } from "@/components/ui/button";
import { PanelCard } from "@/components/ui/panel-card";
import { consolePageClass } from "@/lib/console-layout";
import { ct } from "@/lib/console-typography";
import { cn } from "@/lib/utils";

export function ChatConsolePage() {
  const t = useT();
  const { isGuest, openAuthDialog } = useAuth();

  if (isGuest) {
    return (
      <div className={cn(consolePageClass, "space-y-6")}>
        <PanelCard title={t("chat.signInRequired")}>
          <div className="space-y-4">
            <p className={ct.pageDesc}>{t("chat.guestDesc")}</p>
            <Button type="button" className="bg-teal-600 hover:bg-teal-500" onClick={() => openAuthDialog("login")}>
              {t("chat.signInChat")}
            </Button>
          </div>
        </PanelCard>
      </div>
    );
  }

  return (
    <div className="flex min-h-0 flex-1 flex-col">
      <ChatPage />
    </div>
  );
}
