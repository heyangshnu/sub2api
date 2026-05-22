"use client";

import { ConsoleShell } from "@/components/console-shell";
import { ChatConsolePage } from "@/components/pages/chat-page";

export default function Page() {
  return (
    <ConsoleShell>
      <ChatConsolePage />
    </ConsoleShell>
  );
}
