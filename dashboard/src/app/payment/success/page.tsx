"use client";

import { Suspense, useEffect, useState } from "react";
import { useSearchParams } from "next/navigation";
import { Button } from "@/components/ui/button";
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card";
import Link from "next/link";

function PaymentSuccessContent() {
  const searchParams = useSearchParams();
  const sessionId = searchParams.get("session_id");
  const [status, setStatus] = useState<"loading" | "success" | "error">("loading");
  const [amount, setAmount] = useState<number | null>(null);

  useEffect(() => {
    if (sessionId) {
      checkPaymentStatus();
    } else {
      setStatus("error");
    }
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
    <Card className="w-full max-w-md border border-slate-200/90 bg-white/80 text-slate-900 shadow-xl shadow-slate-200/40 backdrop-blur-2xl ring-1 ring-slate-200/50">
      <CardHeader className="text-center">
        {status === "loading" && (
          <>
            <CardTitle>Processing…</CardTitle>
            <CardDescription>Confirming payment status</CardDescription>
          </>
        )}
        {status === "success" && (
          <>
            <div className="mx-auto mb-4 h-16 w-16 rounded-full bg-green-100 flex items-center justify-center">
              <svg
                className="h-8 w-8 text-green-600"
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
            <CardTitle className="text-green-600">Payment successful</CardTitle>
            <CardDescription>
              {amount ? `$${amount} added to your account` : "Balance updated"}
            </CardDescription>
          </>
        )}
        {status === "error" && (
          <>
            <div className="mx-auto mb-4 h-16 w-16 rounded-full bg-red-100 flex items-center justify-center">
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
            <CardTitle className="text-red-600">Payment not confirmed</CardTitle>
            <CardDescription>Contact support or try again later</CardDescription>
          </>
        )}
      </CardHeader>
      <CardContent className="flex justify-center">
        <Link href="/">
          <Button>Back to dashboard</Button>
        </Link>
      </CardContent>
    </Card>
  );
}

export default function PaymentSuccess() {
  return (
    <div className="flex min-h-screen items-center justify-center p-4">
      <Suspense fallback={
        <Card className="w-full max-w-md border border-slate-200/90 bg-white/80 text-slate-900 backdrop-blur-xl ring-1 ring-slate-200/50 shadow-lg">
          <CardHeader className="text-center">
            <CardTitle>Loading…</CardTitle>
          </CardHeader>
        </Card>
      }>
        <PaymentSuccessContent />
      </Suspense>
    </div>
  );
}
