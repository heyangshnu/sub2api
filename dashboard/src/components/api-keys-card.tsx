"use client";

import { useState } from "react";
import { useAuth } from "@/lib/auth-context";
import { apiClient, APIKey } from "@/lib/api";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
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
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogHeader,
  DialogTitle,
  DialogTrigger,
  DialogFooter,
} from "@/components/ui/dialog";
import { formatUsd } from "@/lib/utils";

function formatDate(dateStr: string) {
  return new Date(dateStr).toLocaleString("zh-CN", {
    year: "numeric",
    month: "2-digit",
    day: "2-digit",
  });
}

export function ApiKeysCard() {
  const { apiKeys, apiKey: currentApiKey, userProfile, refreshKeys, bindUsageApiKey } = useAuth();
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
      setCreateError(err instanceof Error ? err.message : "Failed to create key");
    }
    setCreateLoading(false);
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
      setSettingsError(err instanceof Error ? err.message : "Failed to update settings");
    }
    setSettingsLoading(false);
  };

  const handleDeleteKey = async (keyId: string) => {
    if (!confirm("Are you sure you want to delete this key? This action cannot be undone.")) {
      return;
    }
    try {
      await apiClient.deleteKey(keyId);
      await refreshKeys();
    } catch (err) {
      alert(err instanceof Error ? err.message : "Failed to delete key");
    }
  };

  // Login with API key only (no JWT): show limited view
  if (currentApiKey && apiKeys.length === 0) {
    return (
      <Card className="border border-slate-200/90 bg-white/75 text-slate-800 shadow-lg shadow-slate-200/40 backdrop-blur-xl ring-1 ring-slate-200/50">
        <CardHeader>
          <CardTitle className="text-slate-900">Current API Key</CardTitle>
          <CardDescription className="text-slate-600">You are logged in with this API Key</CardDescription>
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
              {showKey === currentApiKey ? "Hide" : "Show"}
            </Button>
            <Button
              variant="ghost"
              size="sm"
              onClick={() => copyToClipboard(currentApiKey)}
            >
              {copied ? "Copied!" : "Copy"}
            </Button>
          </div>
          <p className="text-xs text-muted-foreground mt-2">
            To manage keys, please login with email/password.
          </p>
        </CardContent>
      </Card>
    );
  }

  return (
    <Card className="border border-slate-200/90 bg-white/75 text-slate-800 shadow-lg shadow-slate-200/40 backdrop-blur-xl ring-1 ring-slate-200/50">
      <CardHeader>
        <div className="flex justify-between items-center">
          <div>
            <CardTitle className="text-slate-900">API Keys</CardTitle>
            <CardDescription className="text-slate-600">
              创建 Key 后可在右上角充值，并用于 OpenAI 兼容接口；创建成功后将自动绑定到本控制台以展示用量。
            </CardDescription>
          </div>
          <div className="flex gap-2">
            <Button variant="outline" size="sm" onClick={() => refreshKeys()}>
              Refresh
            </Button>
            <Dialog open={createOpen} onOpenChange={(open) => (open ? setCreateOpen(true) : handleCloseCreateDialog())}>
              <DialogTrigger>
                <Button
                  type="button"
                  size="sm"
                  disabled={userProfile ? !userProfile.can_create_key : false}
                  title={
                    userProfile && !userProfile.can_create_key
                      ? "请先完成首次账户充值"
                      : undefined
                  }
                >
                  + Create Key
                </Button>
              </DialogTrigger>
              <DialogContent>
                <DialogHeader>
                  <DialogTitle>{newKey ? "Key Created" : "Create New API Key"}</DialogTitle>
                  <DialogDescription>
                    {newKey
                      ? "⚠️ Save this key NOW. It will never be shown again."
                      : "Creating a new key requires password verification."}
                  </DialogDescription>
                </DialogHeader>

                {newKey ? (
                  <div className="space-y-4">
                    <div className="p-4 bg-yellow-500/10 border border-yellow-500/30 rounded-md">
                      <p className="text-sm font-semibold mb-2">Your new API Key:</p>
                      <code className="text-xs block bg-background p-2 rounded break-all font-mono">
                        {newKey}
                      </code>
                      <Button
                        className="w-full mt-3"
                        onClick={() => copyToClipboard(newKey)}
                      >
                        {copied ? "✓ Copied!" : "Copy to Clipboard"}
                      </Button>
                    </div>
                    <DialogFooter>
                      <Button onClick={handleCloseCreateDialog}>
                        I&apos;ve saved the key
                      </Button>
                    </DialogFooter>
                  </div>
                ) : (
                  <div className="space-y-4">
                    <div className="space-y-2">
                      <Label htmlFor="key-name">Key Name (optional)</Label>
                      <Input
                        id="key-name"
                        placeholder="e.g. Production, Testing"
                        value={createName}
                        onChange={(e) => setCreateName(e.target.value)}
                      />
                    </div>
                    <div className="space-y-2">
                      <Label htmlFor="key-rate-limit">Rate Limit (requests/minute)</Label>
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
                      <Label htmlFor="key-spend-limit">消费上限 USD（可选，≤ 账户余额）</Label>
                      <Input
                        id="key-spend-limit"
                        type="number"
                        min={0}
                        step="0.01"
                        placeholder={
                          userProfile
                            ? `留空不限，充值余额 $${userProfile.balance.toFixed(2)}`
                            : "留空表示不限"
                        }
                        value={createSpendLimit}
                        onChange={(e) => setCreateSpendLimit(e.target.value)}
                      />
                    </div>
                    <div className="space-y-2">
                      <Label htmlFor="key-password">Your Password (required)</Label>
                      <Input
                        id="key-password"
                        type="password"
                        placeholder="Enter your account password"
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
                        {createLoading ? "Creating..." : "Create Key"}
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
          <p className="text-muted-foreground text-center py-8">
            No API Keys yet. Click &quot;Create Key&quot; to get started.
          </p>
        ) : (
          <Table>
            <TableHeader>
              <TableRow>
                <TableHead>Name</TableHead>
                <TableHead>Key Prefix</TableHead>
                <TableHead>Balance</TableHead>
                <TableHead>Rate Limit</TableHead>
                <TableHead>IP Whitelist</TableHead>
                <TableHead>Status</TableHead>
                <TableHead>Created</TableHead>
                <TableHead>Actions</TableHead>
              </TableRow>
            </TableHeader>
            <TableBody>
              {apiKeys.map((key: APIKey) => (
                <TableRow key={key.id}>
                  <TableCell className="font-medium">{key.name || "Default"}</TableCell>
                  <TableCell>
                    <code className="text-xs bg-muted px-2 py-1 rounded">
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
                      ? `${key.ip_whitelist.length} IPs`
                      : "Any IP"}
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
                        Settings
                      </Button>
                      <Button
                        variant="ghost"
                        size="sm"
                        className="text-red-500 hover:text-red-600"
                        onClick={() => handleDeleteKey(key.id)}
                      >
                        Delete
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
              <DialogTitle>Key Settings</DialogTitle>
              <DialogDescription>
                Configure security settings for <strong>{editingKey?.name || "this key"}</strong>
              </DialogDescription>
            </DialogHeader>
            <div className="space-y-4">
              <div className="space-y-2">
                <Label htmlFor="settings-rate-limit">Rate Limit (requests/minute)</Label>
                <Input
                  id="settings-rate-limit"
                  type="number"
                  min={1}
                  max={3600}
                  value={settingsRateLimit}
                  onChange={(e) => setSettingsRateLimit(parseInt(e.target.value) || 60)}
                />
                <p className="text-xs text-muted-foreground">
                  Range: 1–3600. Default: 60.
                </p>
              </div>
              <div className="space-y-2">
                <Label htmlFor="settings-ip">IP Whitelist (one per line, leave empty to allow all)</Label>
                <textarea
                  id="settings-ip"
                  className="w-full h-24 p-2 border rounded-md text-sm font-mono"
                  placeholder="192.168.1.1&#10;10.0.0.0/24"
                  value={ipWhitelistInput}
                  onChange={(e) => setIpWhitelistInput(e.target.value)}
                />
                <p className="text-xs text-muted-foreground">
                  Supports single IPs and CIDR (e.g. 10.0.0.0/24). Empty = allow any IP.
                </p>
              </div>
              {settingsError && (
                <p className="text-sm text-red-500">{settingsError}</p>
              )}
              <DialogFooter>
                <Button variant="outline" onClick={() => setSettingsOpen(false)}>
                  Cancel
                </Button>
                <Button onClick={handleSaveSettings} disabled={settingsLoading}>
                  {settingsLoading ? "Saving..." : "Save"}
                </Button>
              </DialogFooter>
            </div>
          </DialogContent>
        </Dialog>

        {/* Show current API key if logged in with one */}
        {currentApiKey && (
          <div className="mt-4 p-4 border rounded-md bg-muted/50">
            <p className="text-sm text-muted-foreground mb-2">Current Session API Key:</p>
            <div className="flex items-center gap-2">
              <code className="text-sm flex-1 font-mono bg-background p-2 rounded">
                {showKey === currentApiKey ? currentApiKey : `${currentApiKey.slice(0, 25)}...`}
              </code>
              <Button
                variant="outline"
                size="sm"
                onClick={() => setShowKey(showKey === currentApiKey ? null : currentApiKey)}
              >
                {showKey === currentApiKey ? "Hide" : "Show"}
              </Button>
              <Button
                variant="outline"
                size="sm"
                onClick={() => copyToClipboard(currentApiKey)}
              >
                {copied ? "Copied!" : "Copy"}
              </Button>
            </div>
          </div>
        )}
      </CardContent>
    </Card>
  );
}
