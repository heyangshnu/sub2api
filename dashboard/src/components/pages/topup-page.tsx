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
  { value: "5", label: "$5", description: "Starter" },
  { value: "10", label: "$10", description: "Personal" },
  { value: "20", label: "$20", description: "Regular" },
  { value: "50", label: "$50", description: "Power user" },
  { value: "100", label: "$100", description: "Team" },
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
        alert(err instanceof Error ? err.message : "Payment failed");
      } finally {
        setLoading(false);
      }
    });
  };

  return (
    <div className="space-y-6">
      <div>
        <h1 className="text-lg font-medium text-slate-900">Top up</h1>
        <p className="mt-2 text-sm text-slate-600">
          Add USD to your account for API and chat usage. Subscriptions (if enabled) control model access and
          period spend caps separately.
        </p>
      </div>

      {isAuthenticated && userProfile && (
        <Card className={glassCard}>
          <CardContent className="pt-4 text-sm">
            Top-up balance: <strong>{formatUsd(userProfile.balance, 2)}</strong> · Spendable{" "}
            <strong>{formatUsd(userProfile.spendable_balance, 2)}</strong>
          </CardContent>
        </Card>
      )}

      <Card className={glassCard}>
        <CardHeader>
          <CardTitle className="text-sm">Select amount</CardTitle>
          <CardDescription>Secure payment via Stripe</CardDescription>
        </CardHeader>
        <CardContent className="space-y-4">
          <Select value={amount} onValueChange={(v) => v && setAmount(v)}>
            <SelectTrigger>
              <SelectValue placeholder="Select amount" />
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
            {isGuest ? "Sign in to pay" : loading ? "Redirecting…" : `Pay $${amount}`}
          </Button>
        </CardContent>
      </Card>
    </div>
  );
}
