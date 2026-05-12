"use client";

import { useEffect, useState } from "react";
import { useAuth } from "@/lib/auth-context";
import { apiClient, UsageResponse, Transaction, Model } from "@/lib/api";
import { Button } from "@/components/ui/button";
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

function formatDate(dateStr: string) {
  return new Date(dateStr).toLocaleString("zh-CN");
}

function formatAmount(amount: number) {
  return `$${amount.toFixed(6)}`;
}

export function Dashboard() {
  const { logout, user, authMode, apiKey } = useAuth();
  const [usage, setUsage] = useState<UsageResponse | null>(null);
  const [transactions, setTransactions] = useState<Transaction[]>([]);
  const [models, setModels] = useState<Model[]>([]);
  const [loading, setLoading] = useState(true);
  const [txTotal, setTxTotal] = useState(0);
  const [txOffset, setTxOffset] = useState(0);
  const txLimit = 10;

  useEffect(() => {
    loadData();
  }, []);

  useEffect(() => {
    loadTransactions();
  }, [txOffset]);

  const loadData = async () => {
    try {
      const [usageData, modelsData] = await Promise.all([
        apiClient.getUsage(),
        apiClient.getModels(),
      ]);
      setUsage(usageData);
      setModels(modelsData.data);
    } catch (error) {
      console.error("Failed to load data:", error);
    } finally {
      setLoading(false);
    }
  };

  const loadTransactions = async () => {
    try {
      const txData = await apiClient.getTransactions(txLimit, txOffset);
      setTransactions(txData.transactions || []);
      setTxTotal(txData.total);
    } catch (error) {
      console.error("Failed to load transactions:", error);
    }
  };

  if (loading) {
    return (
      <div className="min-h-screen bg-background p-8">
        <div className="max-w-6xl mx-auto space-y-8">
          <Skeleton className="h-12 w-64" />
          <div className="grid grid-cols-1 md:grid-cols-3 gap-4">
            <Skeleton className="h-32" />
            <Skeleton className="h-32" />
            <Skeleton className="h-32" />
          </div>
          <Skeleton className="h-64" />
        </div>
      </div>
    );
  }

  return (
    <div className="min-h-screen bg-background">
      {/* Header */}
      <header className="border-b">
        <div className="max-w-6xl mx-auto px-8 py-4 flex justify-between items-center">
          <h1 className="text-2xl font-bold">Sub2API Dashboard</h1>
          <div className="flex items-center gap-3">
            {user && (
              <span className="text-sm text-muted-foreground">
                {user.email}
              </span>
            )}
            {authMode === "api_key" && apiKey && (
              <span className="text-xs text-muted-foreground bg-muted px-2 py-1 rounded">
                {apiKey.slice(0, 15)}...
              </span>
            )}
            <TopupDialog />
            <Button variant="outline" onClick={logout}>
              Logout
            </Button>
          </div>
        </div>
      </header>

      {/* Main Content */}
      <main className="max-w-6xl mx-auto px-8 py-8 space-y-8">
        {/* Stats Cards */}
        <div className="grid grid-cols-1 md:grid-cols-3 gap-4">
          <Card>
            <CardHeader className="pb-2">
              <CardDescription>Balance</CardDescription>
              <CardTitle className="text-3xl text-green-600">
                ${usage?.balance.toFixed(4) || "0.0000"}
              </CardTitle>
            </CardHeader>
          </Card>
          <Card>
            <CardHeader className="pb-2">
              <CardDescription>Total Used</CardDescription>
              <CardTitle className="text-3xl">
                ${usage?.total_used.toFixed(4) || "0.0000"}
              </CardTitle>
            </CardHeader>
          </Card>
          <Card>
            <CardHeader className="pb-2">
              <CardDescription>Total Requests</CardDescription>
              <CardTitle className="text-3xl">
                {usage?.request_count || 0}
              </CardTitle>
            </CardHeader>
          </Card>
        </div>

        {/* API Keys Card - show for JWT login */}
        {authMode === "jwt" && <ApiKeysCard />}

        {/* Models */}
        <Card>
          <CardHeader>
            <CardTitle>Available Models</CardTitle>
            <CardDescription>Supported models and their providers</CardDescription>
          </CardHeader>
          <CardContent>
            <div className="flex flex-wrap gap-2">
              {models.map((model) => (
                <Badge key={model.id} variant="secondary" className="text-sm py-1 px-3">
                  {model.id}
                  <span className="ml-2 text-xs text-muted-foreground">({model.owned_by})</span>
                </Badge>
              ))}
            </div>
          </CardContent>
        </Card>

        {/* Transactions */}
        <Card>
          <CardHeader>
            <CardTitle>Recent Transactions</CardTitle>
            <CardDescription>
              Showing {transactions.length} of {txTotal} transactions
            </CardDescription>
          </CardHeader>
          <CardContent>
            {transactions.length === 0 ? (
              <p className="text-muted-foreground text-center py-8">No transactions yet</p>
            ) : (
              <>
                <Table>
                  <TableHeader>
                    <TableRow>
                      <TableHead>Time</TableHead>
                      <TableHead>Type</TableHead>
                      <TableHead>Model</TableHead>
                      <TableHead>Tokens</TableHead>
                      <TableHead className="text-right">Amount</TableHead>
                      <TableHead className="text-right">Balance After</TableHead>
                    </TableRow>
                  </TableHeader>
                  <TableBody>
                    {transactions.map((tx) => (
                      <TableRow key={tx.id}>
                        <TableCell className="text-sm text-muted-foreground">
                          {formatDate(tx.created_at)}
                        </TableCell>
                        <TableCell>
                          <Badge
                            variant={tx.type === "topup" ? "default" : tx.type === "consume" ? "secondary" : "outline"}
                          >
                            {tx.type}
                          </Badge>
                        </TableCell>
                        <TableCell>{tx.model || "-"}</TableCell>
                        <TableCell>
                          {tx.input_tokens || tx.output_tokens
                            ? `${tx.input_tokens || 0} / ${tx.output_tokens || 0}`
                            : "-"}
                        </TableCell>
                        <TableCell className={`text-right ${tx.type === "topup" ? "text-green-600" : "text-red-600"}`}>
                          {tx.type === "topup" ? "+" : "-"}{formatAmount(tx.amount)}
                        </TableCell>
                        <TableCell className="text-right">
                          {formatAmount(tx.balance_after)}
                        </TableCell>
                      </TableRow>
                    ))}
                  </TableBody>
                </Table>

                {/* Pagination */}
                {txTotal > txLimit && (
                  <div className="flex justify-center gap-2 mt-4">
                    <Button
                      variant="outline"
                      size="sm"
                      onClick={() => setTxOffset(Math.max(0, txOffset - txLimit))}
                      disabled={txOffset === 0}
                    >
                      Previous
                    </Button>
                    <span className="flex items-center px-4 text-sm text-muted-foreground">
                      Page {Math.floor(txOffset / txLimit) + 1} of {Math.ceil(txTotal / txLimit)}
                    </span>
                    <Button
                      variant="outline"
                      size="sm"
                      onClick={() => setTxOffset(txOffset + txLimit)}
                      disabled={txOffset + txLimit >= txTotal}
                    >
                      Next
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
