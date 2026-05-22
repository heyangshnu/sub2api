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
  return new Date(dateStr).toLocaleString("zh-CN");
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
          <TableHead>时间</TableHead>
          <TableHead>类型</TableHead>
          <TableHead>模型</TableHead>
          <TableHead className="text-right">金额</TableHead>
        </TableRow>
      </TableHeader>
      <TableBody>
        <TableRow>
          <TableCell colSpan={4} className="text-center text-slate-500">
            登录后查看账单明细
          </TableCell>
        </TableRow>
      </TableBody>
    </Table>
  );

  return (
    <div className="space-y-6">
      <div>
        <h1 className="text-lg font-medium text-slate-900">账单</h1>
        <p className="mt-2 text-sm text-slate-600">充值入账与 API/对话消费明细</p>
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
            {t === "consume" ? "消费账单" : "充值账单"}
          </Button>
        ))}
      </div>

      <Card className={glassCard}>
        <CardHeader>
          <CardTitle className="text-sm">{tab === "consume" ? "消费记录" : "充值记录"}</CardTitle>
          <CardDescription>
            {isGuest ? "示例结构" : `共 ${total} 条`}
          </CardDescription>
        </CardHeader>
        <CardContent>
          {isGuest ? (
            guestTable
          ) : transactions.length === 0 ? (
            <p className="py-8 text-center text-sm text-slate-500">暂无记录</p>
          ) : (
            <>
              <Table>
                <TableHeader>
                  <TableRow>
                    <TableHead>时间</TableHead>
                    <TableHead>类型</TableHead>
                    <TableHead>模型</TableHead>
                    <TableHead className="text-right">金额</TableHead>
                    <TableHead className="text-right">余额</TableHead>
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
                    上一页
                  </Button>
                  <Button
                    variant="outline"
                    size="sm"
                    disabled={offset + limit >= total}
                    onClick={() => requireAuth(() => setOffset(offset + limit))}
                  >
                    下一页
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
              登录查看账单
            </Button>
          )}
        </CardContent>
      </Card>
    </div>
  );
}
