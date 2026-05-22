"use client";

import { useState } from "react";
import { useAuth } from "@/lib/auth-context";
import { apiClient } from "@/lib/api";
import { Button } from "@/components/ui/button";
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";
import { formatUsd } from "@/lib/utils";

const TOPUP_OPTIONS = [
  { value: "5", label: "$5", description: "入门体验" },
  { value: "10", label: "$10", description: "个人使用" },
  { value: "20", label: "$20", description: "常规用量" },
  { value: "50", label: "$50", description: "重度用户" },
  { value: "100", label: "$100", description: "企业用户" },
];

const glassCard =
  "border border-slate-200/90 bg-white/75 shadow-lg shadow-slate-200/40 backdrop-blur-xl ring-1 ring-slate-200/50";

export function TopupPage() {
  const { isGuest, isAuthenticated, userProfile, requireAuth, refreshProfile } = useAuth();
  const [amount, setAmount] = useState("10");
  const [loading, setLoading] = useState(false);

  const handleTopup = () => {
    requireAuth(async () => {
      setLoading(true);
      try {
        const data = await apiClient.createAccountCheckout(parseFloat(amount));
        if (data.checkout_url) window.location.href = data.checkout_url;
        await refreshProfile();
      } catch (err) {
        alert(err instanceof Error ? err.message : "支付失败");
      } finally {
        setLoading(false);
      }
    });
  };

  return (
    <div className="space-y-6">
      <div>
        <h1 className="text-lg font-medium text-slate-900">账户充值</h1>
        <p className="mt-2 text-sm text-slate-600">
          为账户余额充值（USD），用于 API 与首页对话按量扣费。与「订阅」不同：订阅决定可用模型与周期消费上限。
        </p>
      </div>

      {isAuthenticated && userProfile && (
        <Card className={glassCard}>
          <CardContent className="pt-4 text-sm">
            当前充值结余：<strong>{formatUsd(userProfile.balance, 2)}</strong> · 可消费{" "}
            <strong>{formatUsd(userProfile.spendable_balance, 2)}</strong>
          </CardContent>
        </Card>
      )}

      <Card className={glassCard}>
        <CardHeader>
          <CardTitle className="text-sm">选择充值金额</CardTitle>
          <CardDescription>Stripe 安全支付，到账后可用于消费</CardDescription>
        </CardHeader>
        <CardContent className="space-y-4">
          <Select value={amount} onValueChange={(v) => v && setAmount(v)}>
            <SelectTrigger>
              <SelectValue placeholder="选择金额" />
            </SelectTrigger>
            <SelectContent>
              {TOPUP_OPTIONS.map((opt) => (
                <SelectItem key={opt.value} value={opt.value}>
                  {opt.label} — {opt.description}
                </SelectItem>
              ))}
            </SelectContent>
          </Select>
          <Button
            type="button"
            className="w-full bg-emerald-600 hover:bg-emerald-700"
            disabled={loading}
            onClick={handleTopup}
          >
            {isGuest ? "登录后支付" : loading ? "跳转中…" : `支付 $${amount}`}
          </Button>
        </CardContent>
      </Card>
    </div>
  );
}
