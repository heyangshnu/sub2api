"use client";

import { useCallback, useEffect, useState } from "react";
import Link from "next/link";
import { useAuth } from "@/lib/auth-context";
import { apiClient, DailyUsagePoint } from "@/lib/api";
import { Button, buttonVariants } from "@/components/ui/button";
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card";
import { Skeleton } from "@/components/ui/skeleton";
import { cn, formatUsd } from "@/lib/utils";

const glassCard =
  "border border-slate-200/90 bg-white/75 text-slate-800 shadow-lg shadow-slate-200/40 backdrop-blur-xl ring-1 ring-slate-200/50";

function StatCard({ label, value }: { label: string; value: string }) {
  return (
    <Card className={glassCard}>
      <CardHeader className="pb-2">
        <CardDescription className="text-sm text-slate-800">{label}</CardDescription>
        <CardTitle className="mt-1 text-sm font-normal tabular-nums text-slate-900">{value}</CardTitle>
      </CardHeader>
    </Card>
  );
}

function UsageBars({ points }: { points: DailyUsagePoint[] }) {
  const max = Math.max(...points.map((p) => p.total_consumed), 1e-9);
  return (
    <div className="space-y-3">
      <div className="flex h-40 items-end gap-1 border-b border-slate-200/80 pb-1">
        {points.map((p) => {
          const h = Math.round((p.total_consumed / max) * 100);
          return (
            <div
              key={p.date}
              className="flex min-w-0 flex-1 flex-col items-center justify-end gap-1"
              title={`${p.date}: $${p.total_consumed.toFixed(4)} · ${p.request_count} 次`}
            >
              <div
                className="w-full max-w-[14px] rounded-t-md bg-gradient-to-t from-sky-600 to-sky-400"
                style={{ height: `${Math.max(h, 2)}%` }}
              />
            </div>
          );
        })}
      </div>
      <div className="flex gap-1 text-[10px] leading-tight text-slate-500">
        {points.map((p) => (
          <div key={p.date} className="min-w-0 flex-1 text-center">
            {p.date.slice(5)}
          </div>
        ))}
      </div>
    </div>
  );
}

export function UsagePage() {
  const { isAuthenticated, isGuest, userProfile, apiKeys, apiKey, requireAuth, openAuthDialog, refreshProfile } =
    useAuth();
  const [loading, setLoading] = useState(false);
  const [usageTotal, setUsageTotal] = useState<number | null>(null);
  const [requestCount, setRequestCount] = useState<number | null>(null);
  const [chartKeyId, setChartKeyId] = useState("");
  const [usagePoints, setUsagePoints] = useState<DailyUsagePoint[]>([]);
  const [dailyLoading, setDailyLoading] = useState(false);

  useEffect(() => {
    if (isAuthenticated) void refreshProfile();
  }, [isAuthenticated, refreshProfile]);

  const loadUsage = useCallback(async () => {
    if (!apiClient.getToken() || !apiKey) return;
    setLoading(true);
    try {
      const u = await apiClient.getUsage();
      setUsageTotal(u.total_used);
      setRequestCount(u.request_count);
    } catch {
      setUsageTotal(null);
      setRequestCount(null);
    } finally {
      setLoading(false);
    }
  }, [apiKey]);

  useEffect(() => {
    if (!isAuthenticated) return;
    void loadUsage();
  }, [isAuthenticated, loadUsage, apiKey]);

  useEffect(() => {
    const firstId = apiKeys.find((k) => k.id)?.id ?? "";
    if (!chartKeyId && firstId) setChartKeyId(firstId);
  }, [apiKeys, chartKeyId]);

  useEffect(() => {
    if (!isAuthenticated || !chartKeyId || !apiClient.getToken()) {
      setUsagePoints([]);
      return;
    }
    let cancelled = false;
    (async () => {
      setDailyLoading(true);
      try {
        const res = await apiClient.getUsageDaily(chartKeyId, 14);
        if (!cancelled) setUsagePoints(res.points || []);
      } catch {
        if (!cancelled) setUsagePoints([]);
      } finally {
        if (!cancelled) setDailyLoading(false);
      }
    })();
    return () => {
      cancelled = true;
    };
  }, [chartKeyId, isAuthenticated, apiKeys]);

  if (isGuest) {
    return (
      <div className="space-y-6">
        <div>
          <h1 className="text-lg font-medium text-slate-900">用量信息</h1>
          <p className="mt-2 text-sm text-slate-600">
            OpenAI 兼容 API 网关 · 按 Token 计费 · 支持多模型与 API Key 分账统计
          </p>
        </div>
        <div className="grid grid-cols-1 gap-4 md:grid-cols-3">
          <StatCard label="账户余额（USD）" value="—" />
          <StatCard label="累计消费（USD）" value="—" />
          <StatCard label="请求次数" value="—" />
        </div>
        <Card className={glassCard}>
          <CardHeader>
            <CardTitle className="text-sm font-medium">近 14 日消费趋势</CardTitle>
            <CardDescription className="text-sm text-slate-600">
              登录后查看您的真实用量与按 Key 统计
            </CardDescription>
          </CardHeader>
          <CardContent>
            <div className="flex h-36 items-center justify-center rounded-xl border border-dashed border-slate-200 bg-slate-50/80 text-sm text-slate-500">
              示例图表区域
            </div>
            <Button
              type="button"
              className="mt-4 bg-slate-900 hover:bg-slate-800"
              onClick={() => openAuthDialog("login")}
            >
              登录查看我的用量
            </Button>
          </CardContent>
        </Card>
      </div>
    );
  }

  return (
    <div className="space-y-6">
      <div className="flex flex-wrap items-center justify-between gap-2">
        <h1 className="text-lg font-medium text-slate-900">用量信息</h1>
        {chartKeyId && (
          <Link
            href={`/account/logs?key_id=${encodeURIComponent(chartKeyId)}`}
            className={cn(buttonVariants({ variant: "outline", size: "sm" }), "border-slate-200")}
          >
            请求日志
          </Link>
        )}
      </div>

      {!apiKey && (
        <div className="rounded-2xl border border-sky-200 bg-sky-50/90 px-4 py-3 text-sm text-sky-950">
          请先在{" "}
          <button
            type="button"
            className="font-medium underline"
            onClick={() => requireAuth(() => void (window.location.href = "/keys"))}
          >
            API Keys
          </button>{" "}
          创建 Key 以展示用量曲线（创建后会自动绑定）。
        </div>
      )}

      <div className="grid grid-cols-1 gap-4 md:grid-cols-3">
        <StatCard label="充值结余（USD）" value={formatUsd(userProfile?.balance, 2)} />
        <StatCard label="可消费余额（USD）" value={formatUsd(userProfile?.spendable_balance, 2)} />
        <StatCard
          label="累计消费（USD）"
          value={loading ? "…" : formatUsd(usageTotal ?? 0, 2)}
        />
      </div>

      {userProfile?.subscription?.active && (
        <Card className={glassCard}>
          <CardContent className="pt-4 text-sm text-slate-700">
            当前订阅：<strong>{userProfile.subscription.plan_id}</strong> · 本周期剩余额度{" "}
            <strong>{formatUsd(userProfile.subscription.remaining_cap_usd, 2)}</strong>
          </CardContent>
        </Card>
      )}

      {apiKeys.length > 0 && (
        <Card className={glassCard}>
          <CardHeader className="flex flex-col gap-4 sm:flex-row sm:items-end sm:justify-between">
            <div>
              <CardTitle className="text-sm font-medium">近 14 日消费（按天）</CardTitle>
              <CardDescription className="mt-1 text-sm text-slate-600">
                基于扣费流水汇总（UTC）
              </CardDescription>
            </div>
            <select
              className="min-w-[200px] rounded-lg border border-slate-200 bg-white/90 px-3 py-2 text-sm"
              value={chartKeyId}
              onChange={(e) => setChartKeyId(e.target.value)}
            >
              {apiKeys.map((k) => (
                <option key={k.id} value={k.id}>
                  {k.name || k.key_prefix}
                </option>
              ))}
            </select>
          </CardHeader>
          <CardContent>
            {dailyLoading ? (
              <Skeleton className="h-36 w-full rounded-xl" />
            ) : usagePoints.length === 0 ? (
              <p className="py-10 text-center text-sm text-slate-500">暂无消费记录</p>
            ) : (
              <UsageBars points={usagePoints} />
            )}
          </CardContent>
        </Card>
      )}

      <p className="text-xs text-slate-500">
        请求次数（本 Key）：{requestCount ?? "—"}
      </p>
    </div>
  );
}
