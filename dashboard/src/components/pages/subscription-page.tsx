"use client";

import { useEffect, useState } from "react";
import { useAuth } from "@/lib/auth-context";
import { apiClient, SubscriptionPlan, UserSubscriptionView } from "@/lib/api";
import { Button } from "@/components/ui/button";
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card";
import { Badge } from "@/components/ui/badge";
import { formatUsd } from "@/lib/utils";

const glassCard =
  "border border-slate-200/90 bg-white/75 shadow-lg shadow-slate-200/40 backdrop-blur-xl ring-1 ring-slate-200/50";

export function SubscriptionPage() {
  const { isGuest, isAuthenticated, requireAuth, refreshProfile, userProfile } = useAuth();
  const [plans, setPlans] = useState<SubscriptionPlan[]>([]);
  const [enabled, setEnabled] = useState(false);
  const [sub, setSub] = useState<UserSubscriptionView | null>(null);
  const [loading, setLoading] = useState(false);

  useEffect(() => {
    apiClient
      .getAuthConfig()
      .then((cfg) => {
        setEnabled(!!cfg.subscriptions_enabled);
        setPlans(cfg.subscription_plans || []);
      })
      .catch(() => {});
  }, []);

  useEffect(() => {
    if (!isAuthenticated) return;
    apiClient
      .getSubscription()
      .then((r) => setSub(r.subscription))
      .catch(() => setSub(null));
  }, [isAuthenticated, userProfile]);

  const subscribe = (planId: string) => {
    requireAuth(async () => {
      setLoading(true);
      try {
        const res = await apiClient.createSubscriptionCheckout(planId);
        if (res.activated) {
          await refreshProfile();
          const r = await apiClient.getSubscription();
          setSub(r.subscription);
          alert(res.message || "Plan activated");
        } else if (res.checkout_url) {
          window.location.href = res.checkout_url;
        }
      } catch (e) {
        alert(e instanceof Error ? e.message : "Subscription failed");
      } finally {
        setLoading(false);
      }
    });
  };

  if (!enabled && plans.length === 0) {
    return (
      <div className="space-y-4">
        <h1 className="text-lg font-medium text-slate-900">Subscription</h1>
        <p className="text-sm text-slate-600">
          Subscriptions are disabled on this server (
          <code className="text-xs">SUBSCRIPTIONS_ENABLED=false</code>). Use account top-up and global
          model settings.
        </p>
      </div>
    );
  }

  return (
    <div className="space-y-6">
      <div>
        <h1 className="text-lg font-medium text-slate-900">Plans</h1>
        <p className="mt-2 text-sm text-slate-600">
          Choose a plan for allowed models and period spend cap. Usage is still charged from your account balance.
        </p>
      </div>

      {isAuthenticated && sub?.active && (
        <Card className={glassCard}>
          <CardHeader>
            <CardTitle className="text-sm">Current plan</CardTitle>
            <CardDescription>
              Renews: {new Date(sub.period_end).toLocaleDateString("en-US")}
            </CardDescription>
          </CardHeader>
          <CardContent className="space-y-2 text-sm">
            <p>
              Plan <Badge>{sub.plan_id}</Badge> · Spent {formatUsd(sub.spent_this_period, 2)} / cap{" "}
              {formatUsd(sub.monthly_spend_cap_usd, 2)}
            </p>
            <p className="text-slate-600">Models: {sub.allowed_models.join(", ")}</p>
          </CardContent>
        </Card>
      )}

      <div className="grid gap-4 md:grid-cols-2 lg:grid-cols-3">
        {plans.map((plan) => (
          <Card key={plan.id} className={glassCard}>
            <CardHeader>
              <CardTitle className="text-base capitalize">{plan.id}</CardTitle>
              <CardDescription>
                {plan.monthly_price_usd <= 0
                  ? "Free"
                  : `${formatUsd(plan.monthly_price_usd, 2)} / period`}
              </CardDescription>
            </CardHeader>
            <CardContent className="space-y-3 text-sm">
              <p>Spend cap: {formatUsd(plan.monthly_spend_cap_usd, 2)} / period</p>
              {plan.included_balance_usd > 0 && (
                <p>Included credit: {formatUsd(plan.included_balance_usd, 2)}</p>
              )}
              <div className="flex flex-wrap gap-1">
                {plan.allowed_models.map((m) => (
                  <Badge key={m} variant="secondary" className="text-xs">
                    {m}
                  </Badge>
                ))}
              </div>
              <Button
                type="button"
                className="w-full"
                disabled={loading}
                onClick={() => subscribe(plan.id)}
              >
                {isGuest
                  ? "Sign in to subscribe"
                  : plan.monthly_price_usd <= 0
                    ? "Activate free plan"
                    : "Subscribe"}
              </Button>
            </CardContent>
          </Card>
        ))}
      </div>
    </div>
  );
}
