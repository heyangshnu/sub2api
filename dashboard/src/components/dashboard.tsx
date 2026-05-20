"use client";

import { useCallback, useEffect, useState } from "react";
import Link from "next/link";
import { useAuth } from "@/lib/auth-context";
import {
  apiClient,
  UsageResponse,
  Transaction,
  Model,
  DailyUsagePoint,
} from "@/lib/api";
import { Button, buttonVariants } from "@/components/ui/button";
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card";
import { Badge } from "@/components/ui/badge";
import { Skeleton } from "@/components/ui/skeleton";
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@/components/ui/table";
import { TopupDialog } from "./topup-dialog";
import { ApiKeysCard } from "./api-keys-card";
import { cn, formatUsd } from "@/lib/utils";

const glassCard =
  "border border-slate-200/90 bg-white/75 text-slate-800 shadow-lg shadow-slate-200/40 backdrop-blur-xl ring-1 ring-slate-200/50";

function formatDate(dateStr: string) {
  return new Date(dateStr).toLocaleString("zh-CN");
}

function formatAmount(amount: unknown) {
  return formatUsd(amount, 6);
}

export function Dashboard() {
  const { logout, user, authMode, apiKey, apiKeys } = useAuth();
  const [usage, setUsage] = useState<UsageResponse | null>(null);
  const [transactions, setTransactions] = useState<Transaction[]>([]);
  const [models, setModels] = useState<Model[]>([]);
  const [loading, setLoading] = useState(true);
  const [txTotal, setTxTotal] = useState(0);
  const [txOffset, setTxOffset] = useState(0);
  const txLimit = 10;
  const [chartKeyId, setChartKeyId] = useState("");
  const [usagePoints, setUsagePoints] = useState<DailyUsagePoint[]>([]);
  const [dailyLoading, setDailyLoading] = useState(false);

  const loadData = useCallback(async () => {
    setLoading(true);
    try {
      const raw = apiClient.getApiKey();
      if (!raw) {
        setUsage(null);
        setModels([]);
        return;
      }
      const [usageData, modelsData] = await Promise.all([
        apiClient.getUsage(),
        apiClient.getModels(),
      ]);
      setUsage(usageData);
      setModels(Array.isArray(modelsData.data) ? modelsData.data : []);
    } catch (error) {
      console.error("Failed to load data:", error);
      setUsage(null);
      setModels([]);
    } finally {
      setLoading(false);
    }
  }, []);

  const loadTransactions = useCallback(async () => {
    if (!apiClient.getApiKey()) {
      setTransactions([]);
      setTxTotal(0);
      return;
    }
    try {
      const txData = await apiClient.getTransactions(txLimit, txOffset);
      setTransactions(txData.transactions || []);
      setTxTotal(txData.total);
    } catch (error) {
      console.error("Failed to load transactions:", error);
    }
  }, [txLimit, txOffset]);

  useEffect(() => {
    void loadData();
  }, [loadData, apiKey]);

  useEffect(() => {
    void loadTransactions();
  }, [loadTransactions, apiKey]);

  useEffect(() => {
    if (authMode !== "jwt") {
      setChartKeyId("");
      return;
    }
    const firstId = apiKeys.find((k) => k.id)?.id ?? "";
    if (!chartKeyId && firstId) {
      setChartKeyId(firstId);
      return;
    }
    if (chartKeyId && apiKeys.length > 0 && !apiKeys.some((k) => k.id === chartKeyId)) {
      setChartKeyId(firstId);
    }
  }, [authMode, apiKeys, chartKeyId]);

  useEffect(() => {
    if (authMode !== "jwt" || !chartKeyId || !apiClient.getToken()) {
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
  }, [chartKeyId, authMode, apiKeys]);

  if (loading) {
    return (
      <div className="min-h-screen p-8">
        <div className="mx-auto max-w-6xl space-y-8">
          <Skeleton className="h-12 w-64 rounded-xl bg-slate-200/80" />
          <div className="grid grid-cols-1 gap-4 md:grid-cols-3">
            <Skeleton className="h-32 rounded-2xl bg-slate-200/70" />
            <Skeleton className="h-32 rounded-2xl bg-slate-200/70" />
            <Skeleton className="h-32 rounded-2xl bg-slate-200/70" />
          </div>
          <Skeleton className="h-64 rounded-2xl bg-slate-200/70" />
        </div>
      </div>
    );
  }

  const needsApiKeyForUsage = authMode === "jwt" && !apiKey;

  return (
    <div className="min-h-screen">
      <header className="border-b border-slate-200/80 bg-white/60 backdrop-blur-xl">
        <div className="mx-auto flex max-w-6xl items-center justify-between px-6 py-4 md:px-8">
          <h1 className="text-xl font-semibold tracking-tight text-slate-900 md:text-2xl">
            Sub2API
          </h1>
          <div className="flex items-center gap-3">
            {user && (
              <span className="hidden text-sm text-slate-600 sm:inline">{user.email}</span>
            )}
            {authMode === "api_key" && apiKey && (
              <span className="rounded-lg border border-slate-200 bg-white/80 px-2 py-1 font-mono text-xs text-slate-700">
                {apiKey.slice(0, 15)}…
              </span>
            )}
            <TopupDialog />
            {authMode === "jwt" && chartKeyId && (
              <Link
                href={`/logs?key_id=${encodeURIComponent(chartKeyId)}`}
                className={cn(
                  buttonVariants({ variant: "outline" }),
                  "border-slate-200 bg-white/80 text-slate-800 hover:bg-slate-50"
                )}
              >
                请求日志
              </Link>
            )}
            <Button
              variant="outline"
              className="border-slate-200 bg-white/80 text-slate-800 hover:bg-slate-50"
              onClick={logout}
            >
              退出
            </Button>
          </div>
        </div>
      </header>

      <main className="mx-auto max-w-6xl space-y-8 px-6 py-8 md:px-8">
        {needsApiKeyForUsage && (
          <div
            className={cn(
              "rounded-2xl border border-sky-200 bg-sky-50/90 px-4 py-3 text-sm text-sky-950 backdrop-blur-sm",
              "ring-1 ring-sky-100"
            )}
          >
            用量、模型与交易依赖 API Key。请先创建 Key，再使用右上角<strong className="mx-1">充值</strong>
            ；创建成功后会自动绑定用于本页数据展示。
          </div>
        )}

        <div className="grid grid-cols-1 gap-4 md:grid-cols-3">
          <Card className={glassCard}>
            <CardHeader className="pb-2">
              <CardDescription className="text-slate-600">余额</CardDescription>
              <CardTitle className="text-3xl text-emerald-600">
                {formatUsd(usage?.balance, 4)}
              </CardTitle>
            </CardHeader>
          </Card>
          <Card className={glassCard}>
            <CardHeader className="pb-2">
              <CardDescription className="text-slate-600">累计消费</CardDescription>
              <CardTitle className="text-3xl text-slate-900">
                {formatUsd(usage?.total_used, 4)}
              </CardTitle>
            </CardHeader>
          </Card>
          <Card className={glassCard}>
            <CardHeader className="pb-2">
              <CardDescription className="text-slate-600">请求次数</CardDescription>
              <CardTitle className="text-3xl text-slate-900">
                {usage?.request_count ?? 0}
              </CardTitle>
            </CardHeader>
          </Card>
        </div>

        {authMode === "jwt" && apiKeys.length > 0 && chartKeyId && (
          <Card className={glassCard}>
            <CardHeader className="flex flex-col gap-4 sm:flex-row sm:items-end sm:justify-between">
              <div>
                <CardTitle className="text-slate-900">近 14 日消费（按天）</CardTitle>
                <CardDescription className="text-slate-600">
                  基于扣费流水汇总（UTC 日），不含未扣费请求
                </CardDescription>
              </div>
              <div className="flex flex-wrap items-center gap-2">
                <label htmlFor="usage-key" className="text-sm text-slate-600">
                  Key
                </label>
                <select
                  id="usage-key"
                  className={cn(
                    "min-w-[200px] rounded-lg border border-slate-200 bg-white/90 px-3 py-2 text-sm text-slate-800",
                    "shadow-sm outline-none focus-visible:ring-2 focus-visible:ring-sky-300"
                  )}
                  value={chartKeyId}
                  onChange={(e) => setChartKeyId(e.target.value)}
                >
                  {apiKeys.map((k) => (
                    <option key={k.id} value={k.id}>
                      {k.name || k.key_prefix} · {k.id.slice(0, 8)}…
                    </option>
                  ))}
                </select>
              </div>
            </CardHeader>
            <CardContent>
              {dailyLoading ? (
                <Skeleton className="h-36 w-full rounded-xl bg-slate-200/70" />
              ) : usagePoints.length === 0 ? (
                <p className="py-10 text-center text-sm text-slate-500">该 Key 在所选窗口内暂无消费记录</p>
              ) : (
                <UsageBars points={usagePoints} />
              )}
            </CardContent>
          </Card>
        )}

        {authMode === "jwt" && <ApiKeysCard />}

        <Card className={glassCard}>
          <CardHeader>
            <CardTitle className="text-slate-900">可用模型</CardTitle>
            <CardDescription className="text-slate-600">
              {models.length > 0
                ? "当前 Key 可访问的模型与提供方"
                : "绑定 API Key 后将显示模型列表"}
            </CardDescription>
          </CardHeader>
          <CardContent>
            {models.length === 0 ? (
              <p className="py-8 text-center text-slate-500">暂无数据</p>
            ) : (
              <div className="flex flex-wrap gap-2">
                {models.map((model) => (
                  <Badge
                    key={model.id}
                    variant="secondary"
                    className="border border-slate-200 bg-slate-100/90 px-3 py-1 text-sm text-slate-800"
                  >
                    {model.id}
                    <span className="ml-2 text-xs text-slate-500">({model.owned_by})</span>
                  </Badge>
                ))}
              </div>
            )}
          </CardContent>
        </Card>

        <Card className={glassCard}>
          <CardHeader>
            <CardTitle className="text-slate-900">最近交易</CardTitle>
            <CardDescription className="text-slate-600">
              共 {txTotal} 条 · 本页 {transactions.length} 条
            </CardDescription>
          </CardHeader>
          <CardContent>
            {transactions.length === 0 ? (
              <p className="py-8 text-center text-slate-500">暂无交易记录</p>
            ) : (
              <>
                <Table>
                  <TableHeader>
                    <TableRow className="border-slate-200 hover:bg-transparent">
                      <TableHead className="text-slate-600">时间</TableHead>
                      <TableHead className="text-slate-600">类型</TableHead>
                      <TableHead className="text-slate-600">模型</TableHead>
                      <TableHead className="text-slate-600">Token</TableHead>
                      <TableHead className="text-right text-slate-600">金额</TableHead>
                      <TableHead className="text-right text-slate-600">余额</TableHead>
                    </TableRow>
                  </TableHeader>
                  <TableBody>
                    {transactions.map((tx) => (
                      <TableRow key={tx.id} className="border-slate-200">
                        <TableCell className="text-sm text-slate-600">
                          {formatDate(tx.created_at)}
                        </TableCell>
                        <TableCell>
                          <Badge
                            variant={
                              tx.type === "topup"
                                ? "default"
                                : tx.type === "consume"
                                  ? "secondary"
                                  : "outline"
                            }
                            className="border-slate-200"
                          >
                            {tx.type}
                          </Badge>
                        </TableCell>
                        <TableCell className="text-slate-800">{tx.model || "—"}</TableCell>
                        <TableCell className="text-slate-600">
                          {tx.input_tokens || tx.output_tokens
                            ? `${tx.input_tokens || 0} / ${tx.output_tokens || 0}`
                            : "—"}
                        </TableCell>
                        <TableCell
                          className={`text-right ${tx.type === "topup" ? "text-emerald-600" : "text-rose-600"}`}
                        >
                          {tx.type === "topup" ? "+" : "-"}
                          {formatAmount(tx.amount)}
                        </TableCell>
                        <TableCell className="text-right text-slate-800">
                          {formatAmount(tx.balance_after)}
                        </TableCell>
                      </TableRow>
                    ))}
                  </TableBody>
                </Table>

                {txTotal > txLimit && (
                  <div className="mt-4 flex justify-center gap-2">
                    <Button
                      variant="outline"
                      size="sm"
                      className="border-slate-200 bg-white text-slate-800 hover:bg-slate-50"
                      onClick={() => setTxOffset(Math.max(0, txOffset - txLimit))}
                      disabled={txOffset === 0}
                    >
                      上一页
                    </Button>
                    <span className="flex items-center px-4 text-sm text-slate-600">
                      {Math.floor(txOffset / txLimit) + 1} / {Math.ceil(txTotal / txLimit) || 1}
                    </span>
                    <Button
                      variant="outline"
                      size="sm"
                      className="border-slate-200 bg-white text-slate-800 hover:bg-slate-50"
                      onClick={() => setTxOffset(txOffset + txLimit)}
                      disabled={txOffset + txLimit >= txTotal}
                    >
                      下一页
                    </Button>
                  </div>
                )}
              </>
            )}
          </CardContent>
        </Card>
      </main>
    </div>
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
