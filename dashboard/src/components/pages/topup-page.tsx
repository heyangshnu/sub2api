"use client";

import { useCallback, useEffect, useMemo, useState } from "react";
import { useAuth } from "@/lib/auth-context";
import { ct } from "@/lib/console-typography";
import { formatLocaleDateTime, useLocale, useT } from "@/lib/i18n";
import { apiClient, PaymentRecord } from "@/lib/api";
import { Button } from "@/components/ui/button";
import { PanelCard } from "@/components/ui/panel-card";
import {
  ConsoleTable,
  ConsoleTableHead,
  ConsoleTd,
  ConsoleTh,
} from "@/components/ui/console-table";
import { Skeleton } from "@/components/ui/skeleton";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";
import { consolePageClass } from "@/lib/console-layout";
import { cn, formatUsd } from "@/lib/utils";

export function TopupPage() {
  const t = useT();
  const { locale } = useLocale();
  const { isGuest, isAuthenticated, userProfile, requireAuth, refreshProfile } = useAuth();
  const [amount, setAmount] = useState("10");
  const [loading, setLoading] = useState(false);
  const [payments, setPayments] = useState<PaymentRecord[]>([]);
  const [paymentsLoading, setPaymentsLoading] = useState(false);
  const topupOptions = useMemo(
    () => [
      { value: "5", label: "$5", description: t("topup.tierStarter") },
      { value: "10", label: "$10", description: t("topup.tierPersonal") },
      { value: "20", label: "$20", description: t("topup.tierRegular") },
      { value: "50", label: "$50", description: t("topup.tierPower") },
      { value: "100", label: "$100", description: t("topup.tierTeam") },
    ],
    [t]
  );

  const loadPayments = useCallback(async () => {
    if (!apiClient.getToken()) return;
    setPaymentsLoading(true);
    try {
      const res = await apiClient.listPayments(20, 0);
      setPayments(res.payments || []);
    } catch {
      setPayments([]);
    } finally {
      setPaymentsLoading(false);
    }
  }, []);

  useEffect(() => {
    if (isAuthenticated) void loadPayments();
  }, [isAuthenticated, loadPayments]);

  const handleTopup = () => {
    requireAuth(async () => {
      setLoading(true);
      try {
        const data = await apiClient.createAccountCheckout(parseFloat(amount));
        if (data.checkout_url) window.location.href = data.checkout_url;
        await refreshProfile();
      } catch (err) {
        alert(err instanceof Error ? err.message : t("topup.paymentFailed"));
      } finally {
        setLoading(false);
      }
    });
  };

  return (
    <div className={cn(consolePageClass, "space-y-5")}>
      <p className={ct.pageDesc}>{t("topup.desc")}</p>

      {isAuthenticated && userProfile && (
        <PanelCard>
          <p className={ct.alert}>
            {t("topup.balanceLine", {
              topup: formatUsd(userProfile.balance, 2),
              spendable: formatUsd(userProfile.spendable_balance, 2),
            })}
          </p>
        </PanelCard>
      )}

      <PanelCard
        title={t("topup.selectAmount")}
        description={t("topup.stripeNote")}
      >
        <div className="space-y-4">
          <Select value={amount} onValueChange={(v) => v && setAmount(v)}>
            <SelectTrigger className={ct.tableCell}>
              <SelectValue placeholder={t("topup.selectPlaceholder")} />
            </SelectTrigger>
            <SelectContent>
              {topupOptions.map((opt) => (
                <SelectItem key={opt.value} value={opt.value} className="text-sm">
                  {opt.label} — {opt.description}
                </SelectItem>
              ))}
            </SelectContent>
          </Select>
          <Button
            type="button"
            className="w-full bg-teal-600 text-sm hover:bg-teal-500"
            disabled={loading}
            onClick={handleTopup}
          >
            {isGuest
              ? t("topup.signInPay")
              : loading
                ? t("topup.redirecting")
                : t("topup.pay", { amount })}
          </Button>
        </div>
      </PanelCard>

      {isAuthenticated && (
        <PanelCard title={t("topup.paymentHistory")}>
          {paymentsLoading ? (
            <Skeleton className="h-20 w-full rounded-xl" />
          ) : payments.length === 0 ? (
            <p className={cn("py-6 text-center", ct.empty)}>{t("topup.noPayments")}</p>
          ) : (
            <div className="overflow-x-auto rounded-lg border border-slate-200/80">
              <ConsoleTable className="min-w-[320px]">
                <ConsoleTableHead>
                  <tr>
                    <ConsoleTh>{t("topup.colDate")}</ConsoleTh>
                    <ConsoleTh>{t("topup.colAmount")}</ConsoleTh>
                    <ConsoleTh>{t("topup.colStatus")}</ConsoleTh>
                  </tr>
                </ConsoleTableHead>
                <tbody>
                  {payments.map((p) => (
                    <tr key={p.id} className="border-t border-slate-100">
                      <ConsoleTd variant="muted">
                        {formatLocaleDateTime(p.created_at, locale)}
                      </ConsoleTd>
                      <ConsoleTd variant="strong" className="text-teal-700">
                        {formatUsd(p.amount, 2)}
                      </ConsoleTd>
                      <ConsoleTd variant="muted">
                        {p.status === "completed" ? t("topup.statusCompleted") : p.status}
                      </ConsoleTd>
                    </tr>
                  ))}
                </tbody>
              </ConsoleTable>
            </div>
          )}
        </PanelCard>
      )}
    </div>
  );
}
