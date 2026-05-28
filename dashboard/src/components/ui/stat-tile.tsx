"use client";

import { ct } from "@/lib/console-typography";
import { RippleCard } from "@/components/ui/ripple-card";
import { cn } from "@/lib/utils";

type StatTileProps = {
  label: string;
  value: string;
  rippleDelay?: number;
  className?: string;
  hint?: string;
};

export function StatTile({ label, value, rippleDelay = 0, className, hint }: StatTileProps) {
  return (
    <RippleCard rippleDelay={rippleDelay} className={cn("h-full min-h-[5.5rem]", className)}>
      <div className="relative z-10 flex h-full flex-col justify-center px-5 py-4">
        <p className={ct.statLabel}>{label}</p>
        <p className={cn(ct.statValue, "mt-1")}>{value}</p>
        {hint ? <p className={ct.statHint}>{hint}</p> : null}
      </div>
    </RippleCard>
  );
}
