"use client";

import { useCallback, useEffect, useState } from "react";
import { useAuth } from "@/lib/auth-context";
import { apiClient, Transaction } from "@/lib/api";
import { Button } from "@/components/ui/button";
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card";
import { Badge } from "@/components/ui/badge";
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@/components/ui/table";
import { isConsumeType, isTopupType, transactionTypeLabel } from "@/lib/transaction-labels";
import { cn, formatUsd } from "@/lib/utils";

const glassCard =
  "border border-slate-200/90 bg-white/75 shadow-lg shadow-slate-200/40 backdrop-blur-xl ring-1 ring-slate-200/50";

type Tab = "topup" | "consume";

function formatDate(dateStr: string) {
  return new Date(dateStr).toLocaleString("en-US");
}

export function BillingPage() {
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

  const guestTable = (
    <Table>
      <TableHeader>
        <TableRow>
          <TableHead>Time</TableHead>
          <TableHead>Type</TableHead>
          <TableHead>Model</TableHead>
          <TableHead className="text-right">Amount</TableHead>
        </TableRow>
      </TableHeader>
      <TableBody>
        <TableRow>
          <TableCell colSpan={4} className="text-center text-slate-500">
            Sign in to view billing details
          </TableCell>
        </TableRow>
      </TableBody>
    </Table>
  );

  return (
    <div className="space-y-6">
      <div>
        <h1 className="text-lg font-medium text-slate-900">Billing</h1>
        <p className="mt-2 text-sm text-slate-600">Top-ups and API / chat usage</p>
      </div>

      <div className="flex gap-2">
        {(["consume", "topup"] as Tab[]).map((t) => (
          <Button
            key={t}
            type="button"
            variant={tab === t ? "default" : "outline"}
            size="sm"
            onClick={() => {
              setOffset(0);
              setTab(t);
              if (isGuest) return;
            }}
          >
            {t === "consume" ? "Usage" : "Top-ups"}
          </Button>
        ))}
      </div>

      <Card className={glassCard}>
        <CardHeader>
          <CardTitle className="text-sm">{tab === "consume" ? "Usage" : "Top-ups"}</CardTitle>
          <CardDescription>
            {isGuest ? "Preview" : `${total} entries`}
          </CardDescription>
        </CardHeader>
        <CardContent>
          {isGuest ? (
            guestTable
          ) : transactions.length === 0 ? (
            <p className="py-8 text-center text-sm text-slate-500">No records</p>
          ) : (
            <>
              <Table>
                <TableHeader>
                  <TableRow>
                    <TableHead>Time</TableHead>
                    <TableHead>Type</TableHead>
                    <TableHead>Model</TableHead>
                    <TableHead className="text-right">Amount</TableHead>
                    <TableHead className="text-right">Balance</TableHead>
                  </TableRow>
                </TableHeader>
                <TableBody>
                  {transactions.map((tx) => (
                    <TableRow key={tx.id}>
                      <TableCell className="text-sm">{formatDate(tx.created_at)}</TableCell>
                      <TableCell>
                        <Badge variant="outline">{transactionTypeLabel(tx.type)}</Badge>
                      </TableCell>
                      <TableCell className="text-sm">{tx.model || "—"}</TableCell>
                      <TableCell
                        className={cn(
                          "text-right text-sm",
                          isTopupType(tx.type) ? "text-emerald-600" : "text-rose-600"
                        )}
                      >
                        {isTopupType(tx.type) ? "+" : "-"}
                        {formatUsd(tx.amount, 2)}
                      </TableCell>
                      <TableCell className="text-right text-sm">
                        {formatUsd(tx.balance_after, 2)}
                      </TableCell>
                    </TableRow>
                  ))}
                </TableBody>
              </Table>
              {total > limit && (
                <div className="mt-4 flex justify-center gap-2">
                  <Button
                    variant="outline"
                    size="sm"
                    disabled={offset === 0}
                    onClick={() => requireAuth(() => setOffset(Math.max(0, offset - limit)))}
                  >
                    Previous
                  </Button>
                  <Button
                    variant="outline"
                    size="sm"
                    disabled={offset + limit >= total}
                    onClick={() => requireAuth(() => setOffset(offset + limit))}
                  >
                    Next
                  </Button>
                </div>
              )}
            </>
          )}
          {isGuest && (
            <Button
              type="button"
              className="mt-4 w-full"
              onClick={() => requireAuth(() => {})}
            >
              Sign in to view billing
            </Button>
          )}
        </CardContent>
      </Card>
    </div>
  );
}
