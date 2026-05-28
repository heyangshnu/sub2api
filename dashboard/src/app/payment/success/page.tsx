"use client";

import { Suspense, useEffect, useState } from "react";
import { useSearchParams } from "next/navigation";
import Link from "next/link";
import { LocaleToggle } from "@/components/locale-toggle";
import { useT } from "@/lib/i18n";
import { Button } from "@/components/ui/button";
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card";

function PaymentSuccessContent() {
  const t = useT();
  const searchParams = useSearchParams();
  const sessionId = searchParams.get("session_id");
  const [status, setStatus] = useState<"loading" | "success" | "error">("loading");
  const [amount, setAmount] = useState<number | null>(null);

  useEffect(() => {
    if (sessionId) {
      void checkPaymentStatus();
    } else {
      setStatus("error");
    }
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [sessionId]);

  const checkPaymentStatus = async () => {
    try {
      const res = await fetch(
        `${process.env.NEXT_PUBLIC_API_URL}/v1/payment/status/${sessionId}`
      );

      if (res.ok) {
        const data = await res.json();
        const paid =
          data.payment_status === "paid" ||
          data.status === "complete" ||
          data.status === "paid";
        if (paid) {
          setStatus("success");
          setAmount(data.amount);
        } else {
          setStatus("error");
        }
      } else {
        setStatus("error");
      }
    } catch {
      setStatus("error");
    }
  };

  return (
    <Card className="w-full max-w-md border border-slate-200/90 bg-white/80 text-slate-900 shadow-xl shadow-slate-200/40 backdrop-blur-2xl ring-1 ring-teal-500/10">
      <CardHeader className="text-center">
        {status === "loading" && (
          <>
            <CardTitle>{t("payment.processing")}</CardTitle>
            <CardDescription>{t("payment.confirming")}</CardDescription>
          </>
        )}
        {status === "success" && (
          <>
            <div className="mx-auto mb-4 flex h-16 w-16 items-center justify-center rounded-full bg-teal-100">
              <svg
                className="h-8 w-8 text-teal-600"
                fill="none"
                stroke="currentColor"
                viewBox="0 0 24 24"
              >
                <path
                  strokeLinecap="round"
                  strokeLinejoin="round"
                  strokeWidth={2}
                  d="M5 13l4 4L19 7"
                />
              </svg>
            </div>
            <CardTitle className="text-teal-600">{t("payment.success")}</CardTitle>
            <CardDescription>
              {amount
                ? t("payment.added", { amount: String(amount) })
                : t("payment.balanceUpdated")}
            </CardDescription>
          </>
        )}
        {status === "error" && (
          <>
            <div className="mx-auto mb-4 flex h-16 w-16 items-center justify-center rounded-full bg-red-100">
              <svg
                className="h-8 w-8 text-red-600"
                fill="none"
                stroke="currentColor"
                viewBox="0 0 24 24"
              >
                <path
                  strokeLinecap="round"
                  strokeLinejoin="round"
                  strokeWidth={2}
                  d="M6 18L18 6M6 6l12 12"
                />
              </svg>
            </div>
            <CardTitle className="text-red-600">{t("payment.notConfirmed")}</CardTitle>
            <CardDescription>{t("payment.contactSupport")}</CardDescription>
          </>
        )}
      </CardHeader>
      <CardContent className="flex justify-center">
        <Link href="/">
          <Button className="bg-teal-600 hover:bg-teal-500">{t("payment.backDashboard")}</Button>
        </Link>
      </CardContent>
    </Card>
  );
}

export default function PaymentSuccess() {
  const t = useT();

  return (
    <div className="relative flex min-h-screen items-center justify-center p-4">
      <div className="absolute right-4 top-4">
        <LocaleToggle />
      </div>
      <Suspense
        fallback={
          <Card className="w-full max-w-md border border-slate-200/90 bg-white/80 text-slate-900 shadow-lg ring-1 ring-slate-200/50 backdrop-blur-xl">
            <CardHeader className="text-center">
              <CardTitle>{t("common.loading")}</CardTitle>
            </CardHeader>
          </Card>
        }
      >
        <PaymentSuccessContent />
      </Suspense>
    </div>
  );
}
