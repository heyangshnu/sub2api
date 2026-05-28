"use client";

import { Suspense, useCallback, useEffect, useState } from "react";
import Link from "next/link";
import { useSearchParams } from "next/navigation";
import { ConsoleShell } from "@/components/console-shell";
import { useAuth } from "@/lib/auth-context";
import { formatLocaleDateTime, useLocale, useT } from "@/lib/i18n";
import { apiClient, RequestLogEntry } from "@/lib/api";
import { Button, buttonVariants } from "@/components/ui/button";
import { PanelCard } from "@/components/ui/panel-card";
import { ct } from "@/lib/console-typography";
import { ConsoleTable, ConsoleTableHead, ConsoleTd, ConsoleTh } from "@/components/ui/console-table";
import { Badge } from "@/components/ui/badge";
import { Skeleton } from "@/components/ui/skeleton";
import { consolePageClass } from "@/lib/console-layout";
import { cn } from "@/lib/utils";

function LogsInner() {
  const t = useT();
  const { locale } = useLocale();
  const searchParams = useSearchParams();
  const { isAuthenticated, isLoading, isGuest, authMode, apiKeys, openAuthDialog } = useAuth();
  const keyIdFromUrl = searchParams.get("key_id")?.trim() || "";

  const [keyId, setKeyId] = useState(keyIdFromUrl);
  const [logs, setLogs] = useState<RequestLogEntry[]>([]);
  const [total, setTotal] = useState(0);
  const [offset, setOffset] = useState(0);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const limit = 20;

  useEffect(() => {
    if (keyIdFromUrl) setKeyId(keyIdFromUrl);
  }, [keyIdFromUrl]);

  useEffect(() => {
    if (!keyId && apiKeys.length > 0) setKeyId(apiKeys[0].id);
  }, [apiKeys, keyId]);

  const load = useCallback(async () => {
    if (!keyId || authMode !== "jwt" || !apiClient.getToken()) {
      setLogs([]);
      setTotal(0);
      setLoading(false);
      return;
    }
    setLoading(true);
    setError(null);
    try {
      const res = await apiClient.getRequestLogs(keyId, limit, offset);
      setLogs(res.logs || []);
      setTotal(res.total);
    } catch (e) {
      setError(e instanceof Error ? e.message : t("logs.loadFailed"));
      setLogs([]);
      setTotal(0);
    } finally {
      setLoading(false);
    }
  }, [keyId, offset, authMode, limit, t]);

  useEffect(() => {
    void load();
  }, [load]);

  if (isLoading) {
    return (
      <ConsoleShell>
        <div className={cn(consolePageClass, "space-y-6")}>
          <Skeleton className="h-10 w-48 rounded-xl bg-slate-200/80" />
          <Skeleton className="h-64 rounded-2xl bg-slate-200/70" />
        </div>
      </ConsoleShell>
    );
  }

  if (isGuest || authMode !== "jwt") {
    return (
      <ConsoleShell>
        <div className="mx-auto max-w-xl space-y-4 py-8">
          <p className={ct.pageDesc}>{t("logs.guestDesc")}</p>
          <Button type="button" className="bg-teal-600 hover:bg-teal-500" onClick={() => openAuthDialog("login")}>
            {t("auth.signIn")}
          </Button>
        </div>
      </ConsoleShell>
    );
  }

  if (apiKeys.length === 0) {
    return (
      <ConsoleShell>
        <div className="mx-auto max-w-xl space-y-4">
          <p className={ct.pageDesc}>{t("logs.noKeys")}</p>
          <Link href="/keys" className={cn(buttonVariants({ variant: "outline" }))}>
            {t("logs.goKeys")}
          </Link>
        </div>
      </ConsoleShell>
    );
  }

  return (
    <ConsoleShell>
      <div className={cn(consolePageClass, "space-y-6")}>
        <PanelCard
          title={t("logs.filter")}
          description={t("logs.filterDesc")}
          action={
            <div className="flex flex-wrap items-center gap-2">
              <label htmlFor="log-key" className={ct.tableHead}>
                {t("logs.keyLabel")}
              </label>
              <select
                id="log-key"
                className={cn(
                  "min-w-[200px] rounded-lg border border-slate-200 bg-white/90 px-3 py-2",
                  ct.tableCell,
                  "shadow-sm outline-none focus-visible:ring-2 focus-visible:ring-teal-300"
                )}
                value={keyId}
                onChange={(e) => {
                  setOffset(0);
                  setKeyId(e.target.value);
                }}
              >
                {apiKeys.map((k) => (
                  <option key={k.id} value={k.id}>
                    {k.name || k.key_prefix} · {k.id.slice(0, 8)}…
                  </option>
                ))}
              </select>
            </div>
          }
        >
          {null}
        </PanelCard>

        <PanelCard
          title={t("logs.entries")}
          description={`${total} · ${logs.length}`}
        >
            {error && (
              <p className="mb-4 rounded-lg border border-rose-200 bg-rose-50 px-3 py-2 text-sm text-rose-800">
                {error}
              </p>
            )}
            {loading ? (
              <div className="space-y-2 py-8">
                <Skeleton className="h-10 w-full rounded-lg bg-slate-200/70" />
                <Skeleton className="h-10 w-full rounded-lg bg-slate-200/70" />
              </div>
            ) : logs.length === 0 ? (
              <p className={cn("py-12 text-center", ct.empty)}>{t("logs.noLogs")}</p>
            ) : (
              <>
                <div className="overflow-x-auto rounded-lg border border-slate-200/90">
                  <ConsoleTable>
                    <ConsoleTableHead>
                      <tr className="border-b border-slate-200">
                        <ConsoleTh>{t("billing.colTime")}</ConsoleTh>
                        <ConsoleTh>{t("billing.colModel")}</ConsoleTh>
                        <ConsoleTh>{t("logs.colStream")}</ConsoleTh>
                        <ConsoleTh>{t("logs.colOutcome")}</ConsoleTh>
                        <ConsoleTh className="text-right">{t("logs.colLatency")}</ConsoleTh>
                        <ConsoleTh>{t("logs.colRequestId")}</ConsoleTh>
                      </tr>
                    </ConsoleTableHead>
                    <tbody>
                      {logs.map((row) => (
                        <tr key={row.id} className="border-b border-slate-100 last:border-0">
                          <ConsoleTd variant="muted" className="whitespace-nowrap">
                            {formatLocaleDateTime(row.created_at, locale)}
                          </ConsoleTd>
                          <ConsoleTd>{row.model || "—"}</ConsoleTd>
                          <ConsoleTd variant="muted">
                            {row.stream ? t("common.yes") : t("common.no")}
                          </ConsoleTd>
                          <ConsoleTd>
                            <Badge variant="secondary" className="border border-slate-200 font-normal">
                              {row.outcome}
                            </Badge>
                          </ConsoleTd>
                          <ConsoleTd variant="strong" className="text-right">
                            {row.latency_ms}
                          </ConsoleTd>
                          <ConsoleTd variant="mono" className="max-w-[140px] truncate text-slate-500">
                            {row.request_id || "—"}
                          </ConsoleTd>
                        </tr>
                      ))}
                    </tbody>
                  </ConsoleTable>
                </div>

                {total > limit && (
                  <div className="mt-4 flex justify-center gap-2">
                    <Button
                      variant="outline"
                      size="sm"
                      className="border-slate-200 bg-white text-slate-800 hover:bg-slate-50"
                      onClick={() => setOffset(Math.max(0, offset - limit))}
                      disabled={offset === 0}
                    >
                      {t("common.previous")}
                    </Button>
                    <span className={cn("flex items-center px-4", ct.tableCellMuted)}>
                      {t("logs.pageOf", {
                        page: Math.floor(offset / limit) + 1,
                        total: Math.ceil(total / limit) || 1,
                      })}
                    </span>
                    <Button
                      variant="outline"
                      size="sm"
                      className="border-slate-200 bg-white text-slate-800 hover:bg-slate-50"
                      onClick={() => setOffset(offset + limit)}
                      disabled={offset + limit >= total}
                    >
                      {t("common.next")}
                    </Button>
                  </div>
                )}
              </>
            )}
        </PanelCard>
      </div>
    </ConsoleShell>
  );
}

export default function LogsPage() {
  return (
    <Suspense
      fallback={
        <div className="mx-auto max-w-6xl px-6 py-10 md:px-8">
          <Skeleton className="h-64 rounded-2xl bg-slate-200/70" />
        </div>
      }
    >
      <LogsInner />
    </Suspense>
  );
}
