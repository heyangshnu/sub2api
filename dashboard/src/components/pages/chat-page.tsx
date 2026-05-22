"use client";

import { useAuth } from "@/lib/auth-context";
import { ChatPage } from "@/components/chat/chat-page";
import { Button } from "@/components/ui/button";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";

export function ChatConsolePage() {
  const { isGuest, openAuthDialog } = useAuth();

  if (isGuest) {
    return (
      <div className="space-y-6">
        <h1 className="text-lg font-medium text-slate-900">AI 对话</h1>
        <Card className="border border-slate-200/90 bg-white/75 shadow-lg backdrop-blur-xl">
          <CardHeader>
            <CardTitle className="text-sm">登录后使用</CardTitle>
          </CardHeader>
          <CardContent className="space-y-4">
            <p className="text-sm text-slate-600">
              在控制台直接对话，按 Token 从账户余额扣费；可用模型受订阅档位或系统配置限制。
            </p>
            <Button type="button" onClick={() => openAuthDialog("login")}>
              登录开始对话
            </Button>
          </CardContent>
        </Card>
      </div>
    );
  }

  return (
    <div className="-mx-4 flex min-h-[calc(100vh-8rem)] flex-col md:-mx-6">
      <ChatPage />
    </div>
  );
}
