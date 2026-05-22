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
        <h1 className="text-lg font-medium text-slate-900">Chat</h1>
        <Card className="border border-slate-200/90 bg-white/75 shadow-lg backdrop-blur-xl">
          <CardHeader>
            <CardTitle className="text-sm">Sign in required</CardTitle>
          </CardHeader>
          <CardContent className="space-y-4">
            <p className="text-sm text-slate-600">
              Chat in the console; usage is billed from your account balance. Available models depend on your
              subscription or server configuration.
            </p>
            <Button type="button" onClick={() => openAuthDialog("login")}>
              Sign in to chat
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
