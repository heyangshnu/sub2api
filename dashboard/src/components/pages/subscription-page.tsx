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
          alert(res.message || "已开通");
        } else if (res.checkout_url) {
          window.location.href = res.checkout_url;
        }
      } catch (e) {
        alert(e instanceof Error ? e.message : "订阅失败");
      } finally {
        setLoading(false);
      }
    });
  };

  if (!enabled && plans.length === 0) {
    return (
      <div className="space-y-4">
        <h1 className="text-lg font-medium text-slate-900">订阅</h1>
        <p className="text-sm text-slate-600">
          服务端未启用订阅（<code className="text-xs">SUBSCRIPTIONS_ENABLED=false</code>
          ）。仅使用账户充值与全局模型配置即可。
        </p>
      </div>
    );
  }

  return (
    <div className="space-y-6">
      <div>
        <h1 className="text-lg font-medium text-slate-900">订阅档位</h1>
        <p className="mt-2 text-sm text-slate-600">
          选择档位以解锁可用模型与本周期消费上限；实际扣费仍从账户余额扣除。
        </p>
      </div>

      {isAuthenticated && sub?.active && (
        <Card className={glassCard}>
          <CardHeader>
            <CardTitle className="text-sm">当前订阅</CardTitle>
            <CardDescription>
              到期：{new Date(sub.period_end).toLocaleDateString("zh-CN")}
            </CardDescription>
          </CardHeader>
          <CardContent className="space-y-2 text-sm">
            <p>
              档位 <Badge>{sub.plan_id}</Badge> · 已用 {formatUsd(sub.spent_this_period, 2)} / 上限{" "}
              {formatUsd(sub.monthly_spend_cap_usd, 2)}
            </p>
            <p className="text-slate-600">
              可用模型：{sub.allowed_models.join(", ")}
            </p>
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
                  ? "免费"
                  : `${formatUsd(plan.monthly_price_usd, 2)} / 周期`}
              </CardDescription>
            </CardHeader>
            <CardContent className="space-y-3 text-sm">
              <p>消费上限：{formatUsd(plan.monthly_spend_cap_usd, 2)} / 周期</p>
              {plan.included_balance_usd > 0 && (
                <p>开通赠送余额：{formatUsd(plan.included_balance_usd, 2)}</p>
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
                  ? "登录后订阅"
                  : plan.monthly_price_usd <= 0
                    ? "免费开通"
                    : "订阅此档位"}
              </Button>
            </CardContent>
          </Card>
        ))}
      </div>
    </div>
  );
}
