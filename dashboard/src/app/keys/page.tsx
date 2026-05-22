"use client";

import { ConsoleShell } from "@/components/console-shell";
import { KeysPage } from "@/components/pages/keys-page";

export default function Page() {
  return (
    <ConsoleShell>
      <KeysPage />
    </ConsoleShell>
  );
}
