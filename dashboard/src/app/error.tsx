"use client";

import { useEffect } from "react";
import { Button } from "@/components/ui/button";

export default function Error({
  error,
  reset,
}: {
  error: Error & { digest?: string };
  reset: () => void;
}) {
  useEffect(() => {
    console.error("Dashboard route error:", error);
  }, [error]);

  return (
    <div className="flex min-h-screen flex-col items-center justify-center gap-4 px-4 text-center">
      <p className="text-lg font-semibold text-slate-900">页面加载出错</p>
      <p className="max-w-md text-sm text-slate-600">
        {error.message ||
          "控制台组件异常。请打开浏览器开发者工具 (F12) 查看 Console 中的红色报错。"}
      </p>
      <div className="flex gap-3">
        <Button type="button" onClick={() => reset()}>
          重试
        </Button>
        <Button type="button" variant="outline" onClick={() => window.location.reload()}>
          刷新页面
        </Button>
      </div>
    </div>
  );
}
