"use client";

import { useCallback, useEffect, useMemo, useState } from "react";
import { useAuth } from "@/lib/auth-context";
import { formatLocaleDateTime, useLocale, useT } from "@/lib/i18n";
import { apiClient, Transaction } from "@/lib/api";
import { ct } from "@/lib/console-typography";
import { Button } from "@/components/ui/button";
import { Badge } from "@/components/ui/badge";
import { PanelCard } from "@/components/ui/panel-card";
import {
  ConsoleTable,
  ConsoleTableHead,
  ConsoleTd,
  ConsoleTh,
} from "@/components/ui/console-table";
import { isConsumeType, isTopupType, transactionTypeLabel } from "@/lib/transaction-labels";
import { consolePageClass } from "@/lib/console-layout";
import { cn, formatUsd } from "@/lib/utils";

type Tab = "topup" | "consume";

export function BillingPage() {
  const t = useT();
  const { locale } = useLocale();
  const { isGuest, isAuthenticated, requireAuth } = useAuth();
  const [tab, setTab] = useState<Tab>("consume");
  const [transactions, setTransactions] = useState<Transaction[]>([]);
  const [total, setTotal] = useState(0);
  const [offset, setOffset] = useState(0);
  const limit = 10;
  const load = useCallback(async () => {
    if (!apiClient.getToken()) return;
    const txData = await apiClient.getAccountTransactions(100, 0);
    const all = txData.transactions || [];
    const filtered = all.filter((tx) =>
      tab === "topup" ? isTopupType(tx.type) : isConsumeType(tx.type)
    );
    setTotal(filtered.length);
    setTransactions(filtered.slice(offset, offset + limit));
  }, [tab, offset]);

  useEffect(() => {
    if (isAuthenticated) void load();
  }, [isAuthenticated, load]);

  return (
    <div className={cn(consolePageClass, "space-y-5")}>
      <p className={ct.pageDesc}>{t("billing.desc")}</p>

      <div className="flex gap-2">
        {(["consume", "topup"] as Tab[]).map((tabKey) => (
          <Button
            key={tabKey}
            type="button"
            variant={tab === tabKey ? "default" : "outline"}
            size="sm"
            className={tab === tabKey ? "bg-teal-600 hover:bg-teal-500" : ""}
            onClick={() => {
              setOffset(0);
              setTab(tabKey);
            }}
          >
            {tabKey === "consume" ? t("billing.tabUsage") : t("billing.tabTopup")}
          </Button>
        ))}
      </div>

      <PanelCard
        title={tab === "consume" ? t("billing.tabUsage") : t("billing.tabTopup")}
        description={isGuest ? t("common.preview") : t("common.entries", { count: total })}
      >
        {isGuest ? (
          <p className={cn("py-8 text-center", ct.empty)}>{t("billing.signInView")}</p>
        ) : transactions.length === 0 ? (
          <p className={cn("py-8 text-center", ct.empty)}>{t("billing.noRecords")}</p>
        ) : (
          <>
            <div className="overflow-x-auto rounded-lg border border-slate-200/80">
              <ConsoleTable>
                <ConsoleTableHead>
                  <tr>
                    <ConsoleTh>{t("billing.colTime")}</ConsoleTh>
                    <ConsoleTh>{t("billing.colType")}</ConsoleTh>
                    <ConsoleTh>{t("billing.colModel")}</ConsoleTh>
                    <ConsoleTh className="text-right">{t("billing.colAmount")}</ConsoleTh>
                    <ConsoleTh className="text-right">{t("billing.colBalance")}</ConsoleTh>
                  </tr>
                </ConsoleTableHead>
                <tbody>
                  {transactions.map((tx) => (
                    <tr key={tx.id} className="border-t border-slate-100">
                      <ConsoleTd variant="muted">
                        {formatLocaleDateTime(tx.created_at, locale)}
                      </ConsoleTd>
                      <ConsoleTd>
                        <Badge variant="outline" className="text-sm font-normal">
                          {transactionTypeLabel(tx.type, t)}
                        </Badge>
                      </ConsoleTd>
                      <ConsoleTd>{tx.model || "—"}</ConsoleTd>
                      <ConsoleTd
                        className={cn(
                          "text-right",
                          isTopupType(tx.type) ? "text-teal-600" : "text-rose-600"
                        )}
                        variant="strong"
                      >
                        {isTopupType(tx.type) ? "+" : "-"}
                        {formatUsd(tx.amount, 2)}
                      </ConsoleTd>
                      <ConsoleTd className="text-right" variant="strong">
                        {formatUsd(tx.balance_after, 2)}
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
                  disabled={offset === 0}
                  onClick={() => requireAuth(() => setOffset(Math.max(0, offset - limit)))}
                >
                  {t("common.previous")}
                </Button>
                <Button
                  variant="outline"
                  size="sm"
                  disabled={offset + limit >= total}
                  onClick={() => requireAuth(() => setOffset(offset + limit))}
                >
                  {t("common.next")}
                </Button>
              </div>
            )}
          </>
        )}
        {isGuest && (
          <Button
            type="button"
            className="mt-4 w-full bg-teal-600 text-sm hover:bg-teal-500"
            onClick={() => requireAuth(() => {})}
          >
            {t("billing.signInBilling")}
          </Button>
        )}
      </PanelCard>
    </div>
  );
}
