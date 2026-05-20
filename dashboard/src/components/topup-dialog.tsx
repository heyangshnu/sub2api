"use client";

import { useState } from "react";
import { Button } from "@/components/ui/button";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogHeader,
  DialogTitle,
  DialogTrigger,
} from "@/components/ui/dialog";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";
import { useAuth } from "@/lib/auth-context";

const TOPUP_OPTIONS = [
  { value: "5", label: "$5", description: "入门体验" },
  { value: "10", label: "$10", description: "个人使用" },
  { value: "20", label: "$20", description: "常规用量" },
  { value: "50", label: "$50", description: "重度用户" },
  { value: "100", label: "$100", description: "企业用户" },
];

export function TopupDialog() {
  const { apiKey } = useAuth();
  const [amount, setAmount] = useState("10");
  const [loading, setLoading] = useState(false);
  const [open, setOpen] = useState(false);

  const handleTopup = async () => {
    if (!apiKey) return;

    setLoading(true);
    try {
      const res = await fetch(
        `${process.env.NEXT_PUBLIC_API_URL}/v1/payment/checkout`,
        {
          method: "POST",
          headers: {
            Authorization: `Bearer ${apiKey}`,
            "Content-Type": "application/json",
          },
          body: JSON.stringify({ amount: parseFloat(amount) }),
        }
      );

      const data = await res.json();

      if (!res.ok) {
        throw new Error(data.error?.message || "创建支付失败");
      }

      // Redirect to Stripe Checkout
      if (data.checkout_url) {
        window.location.href = data.checkout_url;
      }
    } catch (err) {
      alert(err instanceof Error ? err.message : "支付失败");
    } finally {
      setLoading(false);
    }
  };

  return (
    <Dialog open={open} onOpenChange={setOpen}>
      <DialogTrigger>
        <Button
          type="button"
          variant="default"
          size="sm"
          disabled={!apiKey}
          className="border border-emerald-200 bg-emerald-50 text-emerald-900 shadow-sm hover:bg-emerald-100 disabled:opacity-50"
        >
          充值
        </Button>
      </DialogTrigger>
      <DialogContent className="border-slate-200/90 bg-white/95 text-slate-900 shadow-xl backdrop-blur-xl sm:max-w-md">
        <DialogHeader>
          <DialogTitle>账户充值</DialogTitle>
          <DialogDescription className="text-slate-600">
            {apiKey
              ? "选择充值金额，支付完成后余额将自动到账到当前 API Key"
              : "请先创建 API Key 后再充值"}
          </DialogDescription>
        </DialogHeader>
        <div className="space-y-4 py-4">
          <div className="space-y-2">
            <label className="text-sm font-medium">充值金额</label>
            <Select value={amount} onValueChange={(v) => v && setAmount(v)}>
              <SelectTrigger>
                <SelectValue placeholder="选择金额" />
              </SelectTrigger>
              <SelectContent>
                {TOPUP_OPTIONS.map((opt) => (
                  <SelectItem key={opt.value} value={opt.value}>
                    <span className="font-medium">{opt.label}</span>
                    <span className="text-muted-foreground ml-2">
                      - {opt.description}
                    </span>
                  </SelectItem>
                ))}
              </SelectContent>
            </Select>
          </div>

          <div className="rounded-lg bg-muted p-4">
            <div className="flex justify-between text-sm">
              <span>充值金额</span>
              <span>${amount}</span>
            </div>
            <div className="flex justify-between text-sm mt-1">
              <span>到账金额</span>
              <span className="font-medium text-green-600">${amount}</span>
            </div>
          </div>

          <Button
            onClick={handleTopup}
            disabled={loading || !apiKey}
            className="w-full"
          >
            {loading ? "跳转中..." : `支付 $${amount}`}
          </Button>

          <p className="text-xs text-muted-foreground text-center">
            支付由 Stripe 安全处理
          </p>
        </div>
      </DialogContent>
    </Dialog>
  );
}
