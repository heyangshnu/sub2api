"use client";

import { ConsoleShell } from "@/components/console-shell";
import { UsagePage } from "@/components/pages/usage-page";

export default function Home() {
  return (
    <ConsoleShell>
      <UsagePage />
    </ConsoleShell>
  );
}
