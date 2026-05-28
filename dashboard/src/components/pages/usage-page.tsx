"use client";

import { useCallback, useEffect, useMemo, useState } from "react";
import Link from "next/link";
import { useAuth } from "@/lib/auth-context";
import { useCardBounceDelay } from "@/lib/use-card-bounce-delay";
import { useT } from "@/lib/i18n";
import {
  apiClient,
  DailyUsagePoint,
  ModelUsageRow,
  UsageSummary,
} from "@/lib/api";
import { Button, buttonVariants } from "@/components/ui/button";
import { PanelCard } from "@/components/ui/panel-card";
import { StatTile } from "@/components/ui/stat-tile";
import { Skeleton } from "@/components/ui/skeleton";
import { ct } from "@/lib/console-typography";
import { ConsoleTable, ConsoleTableHead, ConsoleTd, ConsoleTh } from "@/components/ui/console-table";
import { cn, formatUsd } from "@/lib/utils";

const DAY_OPTIONS = [7, 14, 30] as const;

function UsageBars({ points }: { points: DailyUsagePoint[] }) {
  const t = useT();
  const max = Math.max(...points.map((p) => p.total_consumed), 1e-9);
  return (
    <div className="space-y-4">
      <div className="flex h-44 items-end gap-1.5 border-b border-slate-200/90 pb-2">
        {points.map((p) => {
          const h = Math.round((p.total_consumed / max) * 100);
          return (
            <div
              key={p.date}
              className="flex min-w-0 flex-1 flex-col items-center justify-end gap-1.5"
              title={t("usage.chartTooltip", {
                date: p.date,
                amount: p.total_consumed.toFixed(4),
                count: p.request_count,
              })}
            >
              <div
                className="w-full max-w-4 rounded-t-md bg-gradient-to-t from-teal-600 to-teal-400 shadow-sm"
                style={{ height: `${Math.max(h, 4)}%` }}
              />
            </div>
          );
        })}
      </div>
      <div className={cn("flex gap-1.5", ct.tableCellMuted)}>
        {points.map((p) => (
          <div key={p.date} className="min-w-0 flex-1 text-center tabular-nums">
            {p.date.slice(5)}
          </div>
        ))}
      </div>
    </div>
  );
}

function DayToggle({
  chartDays,
  setChartDays,
}: {
  chartDays: number;
  setChartDays: (d: number) => void;
}) {
  const t = useT();
  const dayLabel = (d: number) =>
    d === 7 ? t("usage.days7") : d === 14 ? t("usage.days14") : t("usage.days30");

  return (
    <div className="inline-flex rounded-xl border border-slate-200/90 bg-slate-50/80 p-1">
      {DAY_OPTIONS.map((d) => (
        <button
          key={d}
          type="button"
          onClick={() => setChartDays(d)}
          className={cn(
            "rounded-lg px-3 py-1.5 text-sm font-medium transition-colors",
            chartDays === d
              ? "bg-teal-600 text-white shadow-sm"
              : "text-slate-600 hover:bg-white hover:text-slate-900"
          )}
        >
          {dayLabel(d)}
        </button>
      ))}
    </div>
  );
}

export function UsagePage() {
  const t = useT();
  const { isAuthenticated, isGuest, userProfile, apiKeys, apiKey, requireAuth, openAuthDialog, refreshProfile } =
    useAuth();
  const [summaryLoading, setSummaryLoading] = useState(false);
  const [summary, setSummary] = useState<UsageSummary | null>(null);
  const [chartDays, setChartDays] = useState<number>(14);
  const [accountPoints, setAccountPoints] = useState<DailyUsagePoint[]>([]);
  const [chartKeyId, setChartKeyId] = useState("");
  const [keyPoints, setKeyPoints] = useState<DailyUsagePoint[]>([]);
  const [modelRows, setModelRows] = useState<ModelUsageRow[]>([]);
  const [dailyLoading, setDailyLoading] = useState(false);
  const [modelLoading, setModelLoading] = useState(false);
  const [exporting, setExporting] = useState(false);

  const rippleBase = useCardBounceDelay();

  useEffect(() => {
    if (isAuthenticated) void refreshProfile();
  }, [isAuthenticated, refreshProfile]);

  const loadSummary = useCallback(async () => {
    if (!apiClient.getToken()) return;
    setSummaryLoading(true);
    try {
      const s = await apiClient.getUsageSummary();
      setSummary(s);
    } catch {
      setSummary(null);
    } finally {
      setSummaryLoading(false);
    }
  }, []);

  const loadAccountChart = useCallback(async () => {
    if (!apiClient.getToken()) return;
    setDailyLoading(true);
    try {
      const res = await apiClient.getAccountUsageDaily(chartDays);
      setAccountPoints(res.points || []);
    } catch {
      setAccountPoints([]);
    } finally {
      setDailyLoading(false);
    }
  }, [chartDays]);

  const loadByModel = useCallback(async () => {
    if (!apiClient.getToken()) return;
    setModelLoading(true);
    try {
      const res = await apiClient.getUsageByModel(30);
      setModelRows(res.rows || []);
    } catch {
      setModelRows([]);
    } finally {
      setModelLoading(false);
    }
  }, []);

  useEffect(() => {
    if (!isAuthenticated) return;
    void loadSummary();
    void loadByModel();
  }, [isAuthenticated, loadSummary, loadByModel]);

  useEffect(() => {
    if (!isAuthenticated) return;
    void loadAccountChart();
  }, [isAuthenticated, loadAccountChart]);

  useEffect(() => {
    const firstId = apiKeys.find((k) => k.id)?.id ?? "";
    if (!chartKeyId && firstId) setChartKeyId(firstId);
  }, [apiKeys, chartKeyId]);

  useEffect(() => {
    if (!isAuthenticated || !chartKeyId || !apiClient.getToken()) {
      setKeyPoints([]);
      return;
    }
    let cancelled = false;
    (async () => {
      try {
        const res = await apiClient.getUsageDaily(chartKeyId, chartDays);
        if (!cancelled) setKeyPoints(res.points || []);
      } catch {
        if (!cancelled) setKeyPoints([]);
      }
    })();
    return () => {
      cancelled = true;
    };
  }, [chartKeyId, chartDays, isAuthenticated, apiKeys]);

  const handleExport = async () => {
    if (!apiClient.getToken()) return;
    setExporting(true);
    try {
      const month = new Date().toISOString().slice(0, 7);
      const blob = await apiClient.downloadUsageExport(month);
      const url = URL.createObjectURL(blob);
      const a = document.createElement("a");
      a.href = url;
      a.download = `usage-${month}.csv`;
      a.click();
      URL.revokeObjectURL(url);
    } catch (e) {
      alert(e instanceof Error ? e.message : "Export failed");
    } finally {
      setExporting(false);
    }
  };

  const dash = "—";
  const stat = (loading: boolean, format: () => string) => (loading ? "…" : format());
  const todayTokenTotal =
    (summary?.today_input_tokens ?? 0) + (summary?.today_output_tokens ?? 0);
  const totalTokenTotal =
    (summary?.total_input_tokens ?? 0) + (summary?.total_output_tokens ?? 0);

  const pageActions = (
    <div className="flex flex-wrap items-center gap-2">
      <Button
        type="button"
        variant="outline"
        size="sm"
        className="border-slate-200"
        disabled={exporting}
        onClick={() => void handleExport()}
      >
        {exporting ? t("usage.exporting") : t("usage.exportCsv")}
      </Button>
      {chartKeyId ? (
        <Link
          href={`/account/logs?key_id=${encodeURIComponent(chartKeyId)}`}
          className={cn(buttonVariants({ variant: "outline", size: "sm" }), "border-slate-200")}
        >
          {t("usage.requestLogs")}
        </Link>
      ) : null}
    </div>
  );

  if (isGuest) {
    return (
      <div className="mx-auto max-w-5xl space-y-6">
        <div className="grid grid-cols-1 gap-4 sm:grid-cols-2 xl:grid-cols-4">
          <StatTile label={t("usage.balanceUsd")} value={dash} rippleDelay={rippleBase} />
          <StatTile label={t("usage.todaySpend")} value={dash} rippleDelay={rippleBase} />
          <StatTile label={t("usage.monthSpend")} value={dash} rippleDelay={rippleBase} />
          <StatTile label={t("usage.totalSpent")} value={dash} rippleDelay={rippleBase} />
        </div>
        <div className="grid grid-cols-1 gap-4 sm:grid-cols-2 xl:grid-cols-4">
          <StatTile label={t("usage.todayRequests")} value={dash} rippleDelay={rippleBase} />
          <StatTile label={t("usage.todayTokens")} value={dash} rippleDelay={rippleBase} />
          <StatTile label={t("usage.totalRequests")} value={dash} rippleDelay={rippleBase} />
          <StatTile label={t("usage.totalTokens")} value={dash} rippleDelay={rippleBase} />
        </div>
        <PanelCard
          title={t("usage.last14")}
          description={t("usage.guestChartDesc")}
          rippleDelay={rippleBase}
        >
          <div className={cn("flex h-40 items-center justify-center rounded-xl border border-dashed border-slate-200 bg-slate-50/80", ct.empty)}>
            {t("usage.sampleChart")}
          </div>
          <Button
            type="button"
            className="mt-5 bg-teal-600 hover:bg-teal-500"
            onClick={() => openAuthDialog("login")}
          >
            {t("usage.signInToView")}
          </Button>
        </PanelCard>
      </div>
    );
  }

  return (
    <div className="mx-auto max-w-5xl space-y-6">
      {pageActions ? <div className="flex justify-end">{pageActions}</div> : null}

      {!apiKey && (
        <div className={cn("rounded-xl border border-teal-200/90 bg-teal-50/80 px-4 py-3", ct.alertBrand)}>
          {t("usage.noApiKeyBefore")}{" "}
          <button
            type="button"
            className="font-medium underline"
            onClick={() => requireAuth(() => void (window.location.href = "/keys"))}
          >
            {t("usage.apiKeysLink")}
          </button>{" "}
          {t("usage.noApiKeyAfter")}
        </div>
      )}

      <div className="grid grid-cols-1 gap-4 sm:grid-cols-2 xl:grid-cols-4">
        <StatTile
          label={t("usage.balanceUsd")}
          value={formatUsd(userProfile?.balance, 2)}
          rippleDelay={rippleBase}
        />
        <StatTile
          label={t("usage.todaySpend")}
          value={stat(summaryLoading, () => formatUsd(summary?.today_spend_usd ?? 0, 2))}
          rippleDelay={rippleBase}
        />
        <StatTile
          label={t("usage.monthSpend")}
          value={stat(summaryLoading, () => formatUsd(summary?.month_spend_usd ?? 0, 2))}
          rippleDelay={rippleBase}
        />
        <StatTile
          label={t("usage.totalSpent")}
          value={stat(summaryLoading, () => formatUsd(summary?.total_spend_usd ?? 0, 2))}
          rippleDelay={rippleBase}
        />
      </div>

      <div className="grid grid-cols-1 gap-4 sm:grid-cols-2 xl:grid-cols-4">
        <StatTile
          label={t("usage.todayRequests")}
          value={stat(summaryLoading, () => String(summary?.today_request_count ?? 0))}
          rippleDelay={rippleBase}
        />
        <StatTile
          label={t("usage.todayTokens")}
          value={stat(summaryLoading, () => todayTokenTotal.toLocaleString())}
          rippleDelay={rippleBase}
        />
        <StatTile
          label={t("usage.totalRequests")}
          value={stat(summaryLoading, () => String(summary?.total_request_count ?? 0))}
          rippleDelay={rippleBase}
        />
        <StatTile
          label={t("usage.totalTokens")}
          value={stat(summaryLoading, () => totalTokenTotal.toLocaleString())}
          rippleDelay={rippleBase}
        />
      </div>

      {userProfile?.subscription?.active && (
        <PanelCard rippleDelay={rippleBase} contentClassName="!py-4">
          <p className={ct.alert}>
            {t("usage.planRemaining", {
              plan: userProfile.subscription.plan_id,
              amount: formatUsd(userProfile.subscription.remaining_cap_usd, 2),
            })}
          </p>
        </PanelCard>
      )}

      <PanelCard
        title={t("usage.accountTrend")}
        description={t("usage.ledgerNote")}
        action={<DayToggle chartDays={chartDays} setChartDays={setChartDays} />}
        rippleDelay={rippleBase}
      >
        {dailyLoading ? (
          <Skeleton className="h-44 w-full rounded-xl" />
        ) : accountPoints.length === 0 ? (
          <p className={cn("py-12 text-center", ct.empty)}>{t("usage.noUsage")}</p>
        ) : (
          <UsageBars points={accountPoints} />
        )}
      </PanelCard>

      {apiKeys.length > 0 && (
        <PanelCard
          title={t("usage.perKeyChart")}
          description={t("usage.ledgerNote")}
          action={
            <select
              className={cn("min-w-[200px] rounded-lg border border-slate-200 bg-white px-3 py-2 shadow-sm", ct.tableCell)}
              value={chartKeyId}
              onChange={(e) => setChartKeyId(e.target.value)}
              aria-label={t("usage.selectKey")}
            >
              {apiKeys.map((k) => (
                <option key={k.id} value={k.id}>
                  {k.name || k.key_prefix}
                </option>
              ))}
            </select>
          }
          rippleDelay={rippleBase}
        >
          {keyPoints.length === 0 ? (
            <p className={cn("py-12 text-center", ct.empty)}>{t("usage.noUsage")}</p>
          ) : (
            <UsageBars points={keyPoints} />
          )}
        </PanelCard>
      )}

      <PanelCard
        title={t("usage.byModel")}
        description={t("usage.byModelDays", { days: 30 })}
        rippleDelay={rippleBase}
      >
        {modelLoading ? (
          <Skeleton className="h-24 w-full rounded-xl" />
        ) : modelRows.length === 0 ? (
          <p className={cn("py-10 text-center", ct.empty)}>{t("usage.noUsage")}</p>
        ) : (
          <div className="overflow-x-auto rounded-xl border border-slate-200/80">
            <ConsoleTable className="min-w-[480px]">
              <ConsoleTableHead>
                <tr>
                  <ConsoleTh>{t("usage.colModel")}</ConsoleTh>
                  <ConsoleTh>{t("usage.colRequests")}</ConsoleTh>
                  <ConsoleTh>{t("usage.colInput")}</ConsoleTh>
                  <ConsoleTh>{t("usage.colOutput")}</ConsoleTh>
                  <ConsoleTh>{t("usage.colSpend")}</ConsoleTh>
                </tr>
              </ConsoleTableHead>
              <tbody>
                {modelRows.map((row) => (
                  <tr key={row.model} className="border-t border-slate-100">
                    <ConsoleTd variant="mono">{row.model}</ConsoleTd>
                    <ConsoleTd>{row.request_count}</ConsoleTd>
                    <ConsoleTd>{row.input_tokens.toLocaleString()}</ConsoleTd>
                    <ConsoleTd>{row.output_tokens.toLocaleString()}</ConsoleTd>
                    <ConsoleTd variant="strong">{formatUsd(row.total_consumed, 4)}</ConsoleTd>
                  </tr>
                ))}
              </tbody>
            </ConsoleTable>
          </div>
        )}
      </PanelCard>
    </div>
  );
}
