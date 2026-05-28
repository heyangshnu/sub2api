"use client";

import { useEffect, useState } from "react";
import { useAuth } from "@/lib/auth-context";
import { formatLocaleDate, useLocale, useT } from "@/lib/i18n";
import { apiClient, SubscriptionPlan, UserSubscriptionView } from "@/lib/api";
import { Button } from "@/components/ui/button";
import { PanelCard } from "@/components/ui/panel-card";
import { Badge } from "@/components/ui/badge";
import { RippleCard } from "@/components/ui/ripple-card";
import { ct } from "@/lib/console-typography";
import { cn, formatUsd } from "@/lib/utils";

export function SubscriptionPage() {
  const t = useT();
  const { locale } = useLocale();
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
          alert(res.message || t("subscription.activated"));
        } else if (res.checkout_url) {
          window.location.href = res.checkout_url;
        }
      } catch (e) {
        alert(e instanceof Error ? e.message : t("subscription.failed"));
      } finally {
        setLoading(false);
      }
    });
  };

  if (!enabled && plans.length === 0) {
    return (
      <div className="space-y-4">
        <p className={ct.pageDesc}>{t("subscription.disabled")}</p>
      </div>
    );
  }

  return (
    <div className="space-y-6">
      <p className={ct.pageDesc}>{t("subscription.desc")}</p>

      {isAuthenticated && sub?.active && (
        <PanelCard
          title={t("subscription.currentPlan")}
          description={t("subscription.renews", { date: formatLocaleDate(sub.period_end, locale) })}
        >
          <div className={cn("space-y-2", ct.tableCell)}>
            <p>
              {t("subscription.planLine", {
                plan: sub.plan_id,
                spent: formatUsd(sub.spent_this_period, 2),
                cap: formatUsd(sub.monthly_spend_cap_usd, 2),
              })}
            </p>
            <p className={ct.panelDesc}>
              {t("subscription.models", { list: sub.allowed_models.join(", ") })}
            </p>
          </div>
        </PanelCard>
      )}

      <div className="grid gap-4 md:grid-cols-2 lg:grid-cols-3">
        {plans.map((plan) => (
          <RippleCard key={plan.id}>
            <div className="p-5">
              <h2 className={cn(ct.panelTitle, "capitalize")}>{plan.id}</h2>
              <p className={ct.panelDesc}>
                {plan.monthly_price_usd <= 0
                  ? t("subscription.free")
                  : t("subscription.perPeriod", {
                      price: formatUsd(plan.monthly_price_usd, 2),
                    })}
              </p>
              <div className={cn("mt-4 space-y-3", ct.tableCell)}>
                <p>
                  {t("subscription.spendCap", {
                    cap: formatUsd(plan.monthly_spend_cap_usd, 2),
                  })}
                </p>
                {plan.included_balance_usd > 0 && (
                  <p>
                    {t("subscription.includedCredit", {
                      amount: formatUsd(plan.included_balance_usd, 2),
                    })}
                  </p>
                )}
                <div className="flex flex-wrap gap-1">
                  {plan.allowed_models.map((m) => (
                    <Badge key={m} variant="secondary" className="border border-slate-200 font-normal">
                      {m}
                    </Badge>
                  ))}
                </div>
                <Button
                  type="button"
                  className="w-full bg-teal-600 hover:bg-teal-500"
                  disabled={loading}
                  onClick={() => subscribe(plan.id)}
                >
                  {isGuest
                    ? t("subscription.signInSubscribe")
                    : plan.monthly_price_usd <= 0
                      ? t("subscription.activateFree")
                      : t("subscription.subscribe")}
                </Button>
              </div>
            </div>
          </RippleCard>
        ))}
      </div>
    </div>
  );
}
