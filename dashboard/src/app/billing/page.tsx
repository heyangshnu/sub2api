"use client";

import { ConsoleShell } from "@/components/console-shell";
import { BillingPage } from "@/components/pages/billing-page";

export default function Page() {
  return (
    <ConsoleShell>
      <BillingPage />
    </ConsoleShell>
  );
}
