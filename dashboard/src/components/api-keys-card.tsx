"use client";

import { useState } from "react";
import { useAuth } from "@/lib/auth-context";
import { useT } from "@/lib/i18n";
import { apiClient, APIKey } from "@/lib/api";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { ct } from "@/lib/console-typography";
import { CardContent, CardHeader } from "@/components/ui/card";
import { Badge } from "@/components/ui/badge";
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@/components/ui/table";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogHeader,
  DialogTitle,
  DialogFooter,
} from "@/components/ui/dialog";
import { RippleCard } from "@/components/ui/ripple-card";
import { cn, formatUsd } from "@/lib/utils";

function formatDate(dateStr: string) {
  return new Date(dateStr).toLocaleString("en-US", {
    year: "numeric",
    month: "2-digit",
    day: "2-digit",
  });
}

export function ApiKeysCard({ rippleDelay = 0 }: { rippleDelay?: number }) {
  const t = useT();
  const { apiKeys, apiKey: currentApiKey, userProfile, refreshKeys, bindUsageApiKey, requireAuth, authMode } =
    useAuth();
  const [showKey, setShowKey] = useState<string | null>(null);
  const [copied, setCopied] = useState(false);

  // Create Key Dialog
  const [createOpen, setCreateOpen] = useState(false);
  const [createName, setCreateName] = useState("");
  const [createPassword, setCreatePassword] = useState("");
  const [createRateLimit, setCreateRateLimit] = useState(60);
  const [createSpendLimit, setCreateSpendLimit] = useState("");
  const [createLoading, setCreateLoading] = useState(false);
  const [createError, setCreateError] = useState("");
  const [newKey, setNewKey] = useState<string | null>(null);

  const [testStatus, setTestStatus] = useState<"idle" | "testing" | "ok" | "fail">("idle");
  const [testMessage, setTestMessage] = useState("");

  const runConnectionTest = async (keyOverride?: string) => {
    const key = keyOverride ?? currentApiKey;
    if (!key) {
      setTestStatus("fail");
      setTestMessage(t("apiKeys.testNeedKey"));
      return;
    }
    setTestStatus("testing");
    setTestMessage("");
    apiClient.setApiKey(key);
    const result = await apiClient.testApiKeyConnection();
    if (result.ok) {
      setTestStatus("ok");
      setTestMessage(t("apiKeys.testOk", { count: result.modelCount }));
    } else {
      setTestStatus("fail");
      setTestMessage(result.message);
    }
  };

  // Settings Dialog
  const [settingsOpen, setSettingsOpen] = useState(false);
  const [editingKey, setEditingKey] = useState<APIKey | null>(null);
  const [ipWhitelistInput, setIpWhitelistInput] = useState("");
  const [settingsRateLimit, setSettingsRateLimit] = useState(60);
  const [settingsLoading, setSettingsLoading] = useState(false);
  const [settingsError, setSettingsError] = useState("");

  const copyToClipboard = async (text: string) => {
    await navigator.clipboard.writeText(text);
    setCopied(true);
    setTimeout(() => setCopied(false), 2000);
  };

  const handleCreateKey = async () => {
    requireAuth(async () => {
    setCreateLoading(true);
    setCreateError("");
    try {
      const spend =
        createSpendLimit.trim() === "" ? undefined : parseFloat(createSpendLimit);
      const result = await apiClient.createKey(
        createPassword,
        createName,
        createRateLimit,
        spend
      );
      setNewKey(result.key);
      setCreatePassword("");
      setCreateName("");
      bindUsageApiKey(result.key);
      await refreshKeys();
    } catch (err) {
      setCreateError(err instanceof Error ? err.message : t("apiKeys.createFailed"));
    }
    setCreateLoading(false);
    });
  };

  const handleCloseCreateDialog = () => {
    setCreateOpen(false);
    setNewKey(null);
    setCreateError("");
    setCreatePassword("");
    setCreateName("");
    setCreateRateLimit(60);
  };

  const openSettings = (key: APIKey) => {
    setEditingKey(key);
    setIpWhitelistInput((key.ip_whitelist || []).join("\n"));
    setSettingsRateLimit(key.rate_limit || 60);
    setSettingsError("");
    setSettingsOpen(true);
  };

  const handleSaveSettings = async () => {
    if (!editingKey) return;
    setSettingsLoading(true);
    setSettingsError("");
    try {
      const ipWhitelist = ipWhitelistInput
        .split("\n")
        .map((s) => s.trim())
        .filter((s) => s.length > 0);
      await apiClient.updateKeySettings(editingKey.id, ipWhitelist, settingsRateLimit);
      await refreshKeys();
      setSettingsOpen(false);
    } catch (err) {
      setSettingsError(err instanceof Error ? err.message : t("apiKeys.updateFailed"));
    }
    setSettingsLoading(false);
  };

  const handleDeleteKey = async (keyId: string) => {
    requireAuth(async () => {
    if (!confirm(t("common.confirmDeleteKey"))) {
      return;
    }
    try {
      await apiClient.deleteKey(keyId);
      await refreshKeys();
    } catch (err) {
      alert(err instanceof Error ? err.message : t("apiKeys.deleteFailed"));
    }
    });
  };

  // Login with API key only (no JWT): show limited view
  if (authMode !== "jwt" && currentApiKey && apiKeys.length === 0) {
    return (
      <RippleCard rippleDelay={rippleDelay} className="text-slate-800">
        <CardHeader>
          <h2 className={ct.panelTitle}>{t("apiKeys.currentKey")}</h2>
          <p className={ct.panelDesc}>{t("apiKeys.loggedInWithKey")}</p>
        </CardHeader>
        <CardContent>
          <div className="flex items-center gap-2 p-3 bg-muted rounded-md">
            <code className="text-sm flex-1 font-mono">
              {showKey === currentApiKey ? currentApiKey : `${currentApiKey.slice(0, 20)}...`}
            </code>
            <Button
              variant="ghost"
              size="sm"
              onClick={() => setShowKey(showKey === currentApiKey ? null : currentApiKey)}
            >
              {showKey === currentApiKey ? t("apiKeys.hide") : t("apiKeys.show")}
            </Button>
            <Button
              variant="ghost"
              size="sm"
              onClick={() => copyToClipboard(currentApiKey)}
            >
              {copied ? t("apiKeys.copied") : t("apiKeys.copy")}
            </Button>
          </div>
          <p className={cn(ct.panelDesc, "mt-2")}>{t("apiKeys.manageWithEmail")}</p>
        </CardContent>
      </RippleCard>
    );
  }

  return (
    <RippleCard rippleDelay={rippleDelay} className="text-slate-800">
      <CardHeader>
        <div className="flex justify-between items-center">
          <div>
            <h2 className={ct.panelTitle}>{t("apiKeys.title")}</h2>
            <p className={ct.panelDesc}>{t("apiKeys.desc")}</p>
            {testMessage ? (
              <p
                className={cn(
                  ct.panelDesc,
                  "mt-1",
                  testStatus === "ok"
                    ? "text-emerald-600"
                    : testStatus === "fail"
                      ? "text-red-600"
                      : undefined
                )}
              >
                {testMessage}
              </p>
            ) : null}
          </div>
          <div className="flex flex-wrap gap-2">
            <Button
              variant="outline"
              size="sm"
              disabled={!currentApiKey || testStatus === "testing"}
              onClick={() => runConnectionTest()}
            >
              {testStatus === "testing" ? t("apiKeys.testing") : t("apiKeys.testConnectivity")}
            </Button>
            <Button variant="outline" size="sm" onClick={() => refreshKeys()}>
              {t("common.refresh")}
            </Button>
            <Dialog open={createOpen} onOpenChange={(open) => (open ? setCreateOpen(true) : handleCloseCreateDialog())}>
                <Button
                  type="button"
                  size="sm"
                  disabled={userProfile ? !userProfile.can_create_key : false}
                  title={
                    userProfile && !userProfile.can_create_key
                      ? t("apiKeys.firstTopupTitle")
                      : undefined
                  }
                  onClick={() => requireAuth(() => setCreateOpen(true))}
                >
                  {t("apiKeys.createKey")}
                </Button>
              <DialogContent>
                <DialogHeader>
                  <DialogTitle>{newKey ? t("apiKeys.keyCreated") : t("apiKeys.createNew")}</DialogTitle>
                  <DialogDescription>
                    {newKey ? t("apiKeys.saveKeyWarn") : t("apiKeys.createVerify")}
                  </DialogDescription>
                </DialogHeader>

                {newKey ? (
                  <div className="space-y-4">
                    <div className="p-4 bg-yellow-500/10 border border-yellow-500/30 rounded-md">
                      <p className="text-sm font-semibold mb-2">{t("apiKeys.yourNewKey")}</p>
                      <code className={cn(ct.tableCellMono, "block rounded bg-slate-50 p-2 break-all")}>
                        {newKey}
                      </code>
                      <Button
                        className="w-full mt-3"
                        onClick={() => copyToClipboard(newKey)}
                      >
                        {copied ? t("apiKeys.copied") : t("apiKeys.copyClipboard")}
                      </Button>
                    </div>
                    <DialogFooter className="flex-col sm:flex-row gap-2">
                      <Button
                        type="button"
                        variant="outline"
                        disabled={testStatus === "testing"}
                        onClick={() => newKey && runConnectionTest(newKey)}
                      >
                        {testStatus === "testing" ? t("apiKeys.testing") : t("apiKeys.testKey")}
                      </Button>
                      <Button onClick={handleCloseCreateDialog}>{t("apiKeys.savedKey")}</Button>
                    </DialogFooter>
                    {testMessage && newKey ? (
                      <p
                        className={cn(
                          ct.panelDesc,
                          testStatus === "ok"
                            ? "text-emerald-600"
                            : testStatus === "fail"
                              ? "text-red-600"
                              : undefined
                        )}
                      >
                        {testMessage}
                      </p>
                    ) : null}
                  </div>
                ) : (
                  <div className="space-y-4">
                    <div className="space-y-2">
                      <Label htmlFor="key-name">{t("apiKeys.keyName")}</Label>
                      <Input
                        id="key-name"
                        placeholder={t("apiKeys.keyNamePh")}
                        value={createName}
                        onChange={(e) => setCreateName(e.target.value)}
                      />
                    </div>
                    <div className="space-y-2">
                      <Label htmlFor="key-rate-limit">{t("apiKeys.rateLimit")}</Label>
                      <Input
                        id="key-rate-limit"
                        type="number"
                        min={1}
                        max={3600}
                        value={createRateLimit}
                        onChange={(e) => setCreateRateLimit(parseInt(e.target.value) || 60)}
                      />
                    </div>
                    <div className="space-y-2">
                      <Label htmlFor="key-spend-limit">{t("apiKeys.spendLimit")}</Label>
                      <Input
                        id="key-spend-limit"
                        type="number"
                        min={0}
                        step="0.01"
                        placeholder={
                          userProfile
                            ? t("apiKeys.spendLimitPh", {
                                balance: userProfile.balance.toFixed(2),
                              })
                            : t("apiKeys.spendLimitPhShort")
                        }
                        value={createSpendLimit}
                        onChange={(e) => setCreateSpendLimit(e.target.value)}
                      />
                    </div>
                    <div className="space-y-2">
                      <Label htmlFor="key-password">{t("apiKeys.yourPassword")}</Label>
                      <Input
                        id="key-password"
                        type="password"
                        placeholder={t("apiKeys.passwordPh")}
                        value={createPassword}
                        onChange={(e) => setCreatePassword(e.target.value)}
                      />
                    </div>
                    {createError && (
                      <p className="text-sm text-red-500">{createError}</p>
                    )}
                    <DialogFooter>
                      <Button
                        onClick={handleCreateKey}
                        disabled={createLoading || !createPassword}
                      >
                        {createLoading ? t("apiKeys.creating") : t("apiKeys.createKeyBtn")}
                      </Button>
                    </DialogFooter>
                  </div>
                )}
              </DialogContent>
            </Dialog>
          </div>
        </div>
      </CardHeader>
      <CardContent>
        {apiKeys.length === 0 ? (
          <p className="text-muted-foreground text-center py-8">{t("apiKeys.noKeys")}</p>
        ) : (
          <Table>
            <TableHeader>
              <TableRow>
                <TableHead>{t("apiKeys.colName")}</TableHead>
                <TableHead>{t("apiKeys.colPrefix")}</TableHead>
                <TableHead>{t("apiKeys.colBalance")}</TableHead>
                <TableHead>{t("apiKeys.colRate")}</TableHead>
                <TableHead>{t("apiKeys.colIp")}</TableHead>
                <TableHead>{t("apiKeys.colStatus")}</TableHead>
                <TableHead>{t("apiKeys.colCreated")}</TableHead>
                <TableHead>{t("apiKeys.colActions")}</TableHead>
              </TableRow>
            </TableHeader>
            <TableBody>
              {apiKeys.map((key: APIKey) => (
                <TableRow key={key.id}>
                  <TableCell className="font-medium">{key.name || t("common.default")}</TableCell>
                  <TableCell>
                    <code className={cn(ct.tableCellMono, "rounded bg-slate-50 px-2 py-1")}>
                      {key.key_prefix}
                    </code>
                  </TableCell>
                  <TableCell className="text-green-600 font-medium">
                    {formatUsd(key.balance, 4)}
                  </TableCell>
                  <TableCell className="text-sm">
                    {key.rate_limit || 60}/min
                  </TableCell>
                  <TableCell className="text-sm">
                    {key.ip_whitelist && key.ip_whitelist.length > 0
                      ? t("common.ipCount", { count: key.ip_whitelist.length })
                      : t("common.anyIp")}
                  </TableCell>
                  <TableCell>
                    <Badge variant={key.status === "active" ? "default" : "secondary"}>
                      {key.status}
                    </Badge>
                  </TableCell>
                  <TableCell className="text-muted-foreground text-sm">
                    {formatDate(key.created_at)}
                  </TableCell>
                  <TableCell>
                    <div className="flex gap-2">
                      <Button variant="ghost" size="sm" onClick={() => openSettings(key)}>
                        {t("apiKeys.settings")}
                      </Button>
                      <Button
                        variant="ghost"
                        size="sm"
                        className="text-red-500 hover:text-red-600"
                        onClick={() => handleDeleteKey(key.id)}
                      >
                        {t("apiKeys.delete")}
                      </Button>
                    </div>
                  </TableCell>
                </TableRow>
              ))}
            </TableBody>
          </Table>
        )}

        {/* Settings Dialog */}
        <Dialog open={settingsOpen} onOpenChange={setSettingsOpen}>
          <DialogContent>
            <DialogHeader>
              <DialogTitle>{t("apiKeys.keySettings")}</DialogTitle>
              <DialogDescription>
                {t("apiKeys.keySettingsDesc", {
                  name: editingKey?.name || t("apiKeys.title"),
                })}
              </DialogDescription>
            </DialogHeader>
            <div className="space-y-4">
              <div className="space-y-2">
                <Label htmlFor="settings-rate-limit">{t("apiKeys.rateLimit")}</Label>
                <Input
                  id="settings-rate-limit"
                  type="number"
                  min={1}
                  max={3600}
                  value={settingsRateLimit}
                  onChange={(e) => setSettingsRateLimit(parseInt(e.target.value) || 60)}
                />
                <p className={ct.panelDesc}>{t("apiKeys.rateHint")}</p>
              </div>
              <div className="space-y-2">
                <Label htmlFor="settings-ip">{t("apiKeys.ipWhitelist")}</Label>
                <textarea
                  id="settings-ip"
                  className="w-full h-24 p-2 border rounded-md text-sm font-mono"
                  placeholder="192.168.1.1&#10;10.0.0.0/24"
                  value={ipWhitelistInput}
                  onChange={(e) => setIpWhitelistInput(e.target.value)}
                />
                <p className={ct.panelDesc}>{t("apiKeys.ipHint")}</p>
              </div>
              {settingsError && (
                <p className="text-sm text-red-500">{settingsError}</p>
              )}
              <DialogFooter>
                <Button variant="outline" onClick={() => setSettingsOpen(false)}>
                  {t("common.cancel")}
                </Button>
                <Button onClick={handleSaveSettings} disabled={settingsLoading}>
                  {settingsLoading ? t("apiKeys.saving") : t("common.save")}
                </Button>
              </DialogFooter>
            </div>
          </DialogContent>
        </Dialog>

        {/* Show current API key if logged in with one */}
        {currentApiKey && (
          <div className="mt-4 p-4 border rounded-md bg-muted/50">
            <p className="text-sm text-muted-foreground mb-2">{t("apiKeys.sessionKey")}</p>
            <div className="flex items-center gap-2">
              <code className="text-sm flex-1 font-mono bg-background p-2 rounded">
                {showKey === currentApiKey ? currentApiKey : `${currentApiKey.slice(0, 25)}...`}
              </code>
              <Button
                variant="outline"
                size="sm"
                onClick={() => setShowKey(showKey === currentApiKey ? null : currentApiKey)}
              >
                {showKey === currentApiKey ? t("apiKeys.hide") : t("apiKeys.show")}
              </Button>
              <Button
                variant="outline"
                size="sm"
                onClick={() => copyToClipboard(currentApiKey)}
              >
                {copied ? t("apiKeys.copied") : t("apiKeys.copy")}
              </Button>
            </div>
          </div>
        )}
      </CardContent>
    </RippleCard>
  );
}
