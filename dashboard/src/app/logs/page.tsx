"use client";

import { Suspense, useCallback, useEffect, useState } from "react";
import Link from "next/link";
import { useSearchParams } from "next/navigation";
import { ConsoleShell } from "@/components/console-shell";
import { useAuth } from "@/lib/auth-context";
import { apiClient, RequestLogEntry } from "@/lib/api";
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
import { cn } from "@/lib/utils";

const glassCard =
  "border border-slate-200/90 bg-white/75 text-slate-800 shadow-lg shadow-slate-200/40 backdrop-blur-xl ring-1 ring-slate-200/50";

function formatDate(dateStr: string) {
  return new Date(dateStr).toLocaleString("en-US");
}

function LogsInner() {
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
      setError(e instanceof Error ? e.message : "Failed to load");
      setLogs([]);
      setTotal(0);
    } finally {
      setLoading(false);
    }
  }, [keyId, offset, authMode, limit]);

  useEffect(() => {
    void load();
  }, [load]);

  if (isLoading) {
    return (
      <ConsoleShell>
        <div className="mx-auto max-w-6xl space-y-6">
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
          <h1 className="text-lg font-medium text-slate-900">Request logs</h1>
          <p className="text-sm text-slate-600">Sign in to view recent API calls per key.</p>
          <Button type="button" onClick={() => openAuthDialog("login")}>
            Sign in
          </Button>
        </div>
      </ConsoleShell>
    );
  }

  if (apiKeys.length === 0) {
    return (
      <ConsoleShell>
        <div className="mx-auto max-w-xl space-y-4">
          <h1 className="text-lg font-medium text-slate-900">Request logs</h1>
          <p className="text-sm text-slate-600">Create an API key first, then logs appear after calls.</p>
          <Link href="/keys" className={cn(buttonVariants({ variant: "outline" }))}>
            Go to API Keys
          </Link>
        </div>
      </ConsoleShell>
    );
  }

  return (
    <ConsoleShell>
      <div className="mx-auto max-w-6xl space-y-6">
        <h1 className="text-lg font-semibold text-slate-900 md:text-xl">Request logs</h1>
        <Card className={glassCard}>
          <CardHeader className="flex flex-col gap-4 sm:flex-row sm:items-end sm:justify-between">
            <div>
              <CardTitle className="text-slate-900">Filter</CardTitle>
              <CardDescription className="text-slate-600">
                Recent calls by API key (request body not stored)
              </CardDescription>
            </div>
            <div className="flex flex-wrap items-center gap-2">
              <label htmlFor="log-key" className="text-sm text-slate-600">
                Key
              </label>
              <select
                id="log-key"
                className={cn(
                  "min-w-[200px] rounded-lg border border-slate-200 bg-white/90 px-3 py-2 text-sm text-slate-800",
                  "shadow-sm outline-none focus-visible:ring-2 focus-visible:ring-sky-300"
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
          </CardHeader>
        </Card>

        <Card className={glassCard}>
          <CardHeader>
            <CardTitle className="text-slate-900">Entries</CardTitle>
            <CardDescription className="text-slate-600">
              {total} total · {logs.length} on this page
            </CardDescription>
          </CardHeader>
          <CardContent>
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
              <p className="py-12 text-center text-slate-500">No logs yet</p>
            ) : (
              <>
                <Table>
                  <TableHeader>
                    <TableRow className="border-slate-200 hover:bg-transparent">
                      <TableHead className="text-slate-600">Time</TableHead>
                      <TableHead className="text-slate-600">Model</TableHead>
                      <TableHead className="text-slate-600">Stream</TableHead>
                      <TableHead className="text-slate-600">Outcome</TableHead>
                      <TableHead className="text-right text-slate-600">Latency ms</TableHead>
                      <TableHead className="font-mono text-xs text-slate-600">request_id</TableHead>
                    </TableRow>
                  </TableHeader>
                  <TableBody>
                    {logs.map((row) => (
                      <TableRow key={row.id} className="border-slate-200">
                        <TableCell className="whitespace-nowrap text-sm text-slate-600">
                          {formatDate(row.created_at)}
                        </TableCell>
                        <TableCell className="text-slate-800">{row.model || "—"}</TableCell>
                        <TableCell className="text-slate-600">{row.stream ? "Yes" : "No"}</TableCell>
                        <TableCell>
                          <Badge variant="secondary" className="border border-slate-200 font-normal">
                            {row.outcome}
                          </Badge>
                        </TableCell>
                        <TableCell className="text-right tabular-nums text-slate-800">
                          {row.latency_ms}
                        </TableCell>
                        <TableCell className="max-w-[140px] truncate font-mono text-xs text-slate-500">
                          {row.request_id || "—"}
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
                      className="border-slate-200 bg-white text-slate-800 hover:bg-slate-50"
                      onClick={() => setOffset(Math.max(0, offset - limit))}
                      disabled={offset === 0}
                    >
                      Previous
                    </Button>
                    <span className="flex items-center px-4 text-sm text-slate-600">
                      {Math.floor(offset / limit) + 1} / {Math.ceil(total / limit) || 1}
                    </span>
                    <Button
                      variant="outline"
                      size="sm"
                      className="border-slate-200 bg-white text-slate-800 hover:bg-slate-50"
                      onClick={() => setOffset(offset + limit)}
                      disabled={offset + limit >= total}
                    >
                      Next
                    </Button>
                  </div>
                )}
              </>
            )}
          </CardContent>
        </Card>
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
