"use client";

import { useState } from "react";
import { AppShell } from "@/components/app-shell";
import { apiClient } from "@/lib/api";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";

const fieldLabel = "text-sm text-slate-800";
const fieldInput =
  "h-10 w-full max-w-md rounded-lg border border-slate-200 bg-white text-sm text-slate-900";

export default function ProfilePage() {
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
      setMsg("密码已更新");
    } catch (e) {
      setErr(e instanceof Error ? e.message : "更新失败");
    }
  };

  return (
    <AppShell>
      <div className="mx-auto w-full max-w-2xl px-6 py-8 md:px-8">
        <h1 className="text-sm font-medium text-slate-900">个人中心</h1>
        <p className="mt-2 text-sm text-slate-800">修改登录密码</p>

        {(msg || err) && (
          <p className={`mt-4 text-sm ${err ? "text-red-700" : "text-slate-700"}`}>{err || msg}</p>
        )}

        <div className="mt-6 max-w-md space-y-4">
          <div className="space-y-2">
            <Label htmlFor="cur" className={fieldLabel}>
              当前密码
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
              新密码
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
            onClick={() => void savePassword()}
            className="h-10 rounded-lg bg-slate-900 px-6 text-sm hover:bg-slate-800"
          >
            更新密码
          </Button>
        </div>
      </div>
    </AppShell>
  );
}
