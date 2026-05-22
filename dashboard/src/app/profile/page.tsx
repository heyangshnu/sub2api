"use client";

import { useState } from "react";
import { ConsoleShell } from "@/components/console-shell";
import { useAuth } from "@/lib/auth-context";
import { apiClient } from "@/lib/api";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";

const fieldLabel = "text-sm text-slate-800";
const fieldInput =
  "h-10 w-full max-w-md rounded-lg border border-slate-200 bg-white text-sm text-slate-900";

export default function ProfilePage() {
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
      setMsg("Password updated");
    } catch (e) {
      setErr(e instanceof Error ? e.message : "Update failed");
    }
  };

  if (isGuest) {
    return (
      <ConsoleShell>
        <div className="mx-auto max-w-2xl space-y-4">
          <h1 className="text-lg font-medium text-slate-900">Profile</h1>
          <p className="text-sm text-slate-600">Sign in to change your password.</p>
          <Button type="button" onClick={() => openAuthDialog("login")}>
            Sign in
          </Button>
        </div>
      </ConsoleShell>
    );
  }

  return (
    <ConsoleShell>
      <div className="mx-auto w-full max-w-2xl">
        <h1 className="text-sm font-medium text-slate-900">Profile</h1>
        <p className="mt-2 text-sm text-slate-800">Change password</p>

        {(msg || err) && (
          <p className={`mt-4 text-sm ${err ? "text-red-700" : "text-slate-700"}`}>{err || msg}</p>
        )}

        <div className="mt-6 max-w-md space-y-4">
          <div className="space-y-2">
            <Label htmlFor="cur" className={fieldLabel}>
              Current password
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
              New password
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
            className="h-10 rounded-lg bg-slate-900 px-6 text-sm hover:bg-slate-800"
          >
            Update password
          </Button>
        </div>
      </div>
    </ConsoleShell>
  );
}
