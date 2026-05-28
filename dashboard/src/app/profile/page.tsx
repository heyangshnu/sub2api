"use client";

import { useMemo, useState } from "react";
import { ConsoleShell } from "@/components/console-shell";
import { useAuth } from "@/lib/auth-context";
import { useT } from "@/lib/i18n";
import { apiClient } from "@/lib/api";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { PanelCard } from "@/components/ui/panel-card";
import { ct } from "@/lib/console-typography";
import { cn } from "@/lib/utils";

const fieldLabel = ct.tableCell;
const fieldInput =
  "h-10 w-full max-w-md rounded-lg border border-slate-200 bg-white text-sm text-slate-900";

export default function ProfilePage() {
  const t = useT();
  const { requireAuth, isGuest, openAuthDialog } = useAuth();
  const [currentPassword, setCurrentPassword] = useState("");
  const [newPassword, setNewPassword] = useState("");
  const [msg, setMsg] = useState("");
  const [err, setErr] = useState("");
  const savePassword = async () => {
    setErr("");
    setMsg("");
    try {
      await apiClient.changePassword(currentPassword, newPassword);
      setCurrentPassword("");
      setNewPassword("");
      setMsg(t("profile.passwordUpdated"));
    } catch (e) {
      setErr(e instanceof Error ? e.message : t("profile.updateFailed"));
    }
  };

  if (isGuest) {
    return (
      <ConsoleShell>
        <div className="mx-auto max-w-2xl space-y-4">
          <p className={ct.pageDesc}>{t("profile.guestDesc")}</p>
          <Button type="button" className="bg-teal-600 hover:bg-teal-500" onClick={() => openAuthDialog("login")}>
            {t("auth.signIn")}
          </Button>
        </div>
      </ConsoleShell>
    );
  }

  return (
    <ConsoleShell>
      <div className="mx-auto w-full max-w-2xl">
        <p className={ct.pageDesc}>{t("profile.changePassword")}</p>

        {(msg || err) && (
          <p className={cn("mt-4", ct.alert, err && "text-red-700")}>{err || msg}</p>
        )}

        <PanelCard className="mt-6 max-w-md" contentClassName="!p-4">
          <div className="space-y-4">
            <div className="space-y-2">
              <Label htmlFor="cur" className={fieldLabel}>
                {t("profile.currentPassword")}
              </Label>
              <Input
                id="cur"
                type="password"
                value={currentPassword}
                onChange={(e) => setCurrentPassword(e.target.value)}
                className={fieldInput}
                autoComplete="current-password"
              />
            </div>
            <div className="space-y-2">
              <Label htmlFor="new" className={fieldLabel}>
                {t("profile.newPassword")}
              </Label>
              <Input
                id="new"
                type="password"
                value={newPassword}
                onChange={(e) => setNewPassword(e.target.value)}
                className={fieldInput}
                autoComplete="new-password"
              />
            </div>
            <Button
              type="button"
              onClick={() => requireAuth(() => void savePassword())}
              className="h-10 rounded-lg bg-teal-600 px-6 text-sm hover:bg-teal-500"
            >
              {t("profile.updatePassword")}
            </Button>
          </div>
        </PanelCard>
      </div>
    </ConsoleShell>
  );
}
