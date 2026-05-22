"use client";

import { useAuth } from "@/lib/auth-context";
import { ApiKeysCard } from "@/components/api-keys-card";
import { Button } from "@/components/ui/button";
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card";

const glassCard =
  "border border-slate-200/90 bg-white/75 shadow-lg shadow-slate-200/40 backdrop-blur-xl ring-1 ring-slate-200/50";

export function KeysPage() {
  const { isGuest, openAuthDialog, requireAuth } = useAuth();

  if (isGuest) {
    return (
      <div className="space-y-6">
        <div>
          <h1 className="text-lg font-medium text-slate-900">API Keys</h1>
          <p className="mt-2 text-sm text-slate-600">
            创建独立 Key 调用 OpenAI 兼容接口，便于分项目统计用量与设置消费上限。
          </p>
        </div>
        <Card className={glassCard}>
          <CardHeader>
            <CardTitle className="text-sm">接口地址</CardTitle>
            <CardDescription className="font-mono text-xs">
              {process.env.NEXT_PUBLIC_API_URL || "https://api.cloudtoken.uk"}/v1
            </CardDescription>
          </CardHeader>
          <CardContent className="space-y-4">
            <p className="text-sm text-slate-600">
              登录后可创建 Key；完整 Key 仅显示一次，请妥善保存。支持 IP 白名单、消费上限与连通性检测。
            </p>
            <Button type="button" onClick={() => openAuthDialog("login")}>
              登录管理 API Keys
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
