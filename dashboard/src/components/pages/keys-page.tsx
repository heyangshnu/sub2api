"use client";

import { useAuth } from "@/lib/auth-context";
import { ApiKeysCard } from "@/components/api-keys-card";
import { Button } from "@/components/ui/button";
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card";

const glassCard =
  "border border-slate-200/90 bg-white/75 shadow-lg shadow-slate-200/40 backdrop-blur-xl ring-1 ring-slate-200/50";

export function KeysPage() {
  const { isGuest, openAuthDialog } = useAuth();

  if (isGuest) {
    return (
      <div className="space-y-6">
        <div>
          <h1 className="text-lg font-medium text-slate-900">API Keys</h1>
          <p className="mt-2 text-sm text-slate-600">
            Create keys for OpenAI-compatible API access, per-project usage, and spend limits.
          </p>
        </div>
        <Card className={glassCard}>
          <CardHeader>
            <CardTitle className="text-sm">Base URL</CardTitle>
            <CardDescription className="font-mono text-xs">
              {process.env.NEXT_PUBLIC_API_URL || "https://api.cloudtoken.uk"}/v1
            </CardDescription>
          </CardHeader>
          <CardContent className="space-y-4">
            <p className="text-sm text-slate-600">
              Sign in to create keys. The full key is shown only once—save it securely. IP allowlist,
              spend limits, and connectivity checks are supported.
            </p>
            <Button type="button" onClick={() => openAuthDialog("login")}>
              Sign in to manage API keys
            </Button>
          </CardContent>
        </Card>
      </div>
    );
  }

  return (
    <div className="space-y-4">
      <h1 className="text-lg font-medium text-slate-900">API Keys</h1>
      <ApiKeysCard />
    </div>
  );
}
