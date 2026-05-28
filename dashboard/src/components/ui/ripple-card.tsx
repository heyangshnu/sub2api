"use client";

import { useEffect, useMemo, useState } from "react";
import { createBounceVariant } from "@/lib/bounce-variant";
import { cn } from "@/lib/utils";

const glassCard =
  "rounded-xl border border-slate-200/90 bg-white/90 text-sm text-slate-800 shadow-[0_4px_24px_rgba(15,23,42,0.05)] backdrop-blur-xl ring-1 ring-slate-200/80";

type RippleCardProps = React.ComponentProps<"div"> & {
  rippleDelay?: number;
};

export function RippleCard({
  className,
  children,
  rippleDelay = 0,
  ...props
}: RippleCardProps) {
  const variant = useMemo(() => createBounceVariant(), []);
  const [phase, setPhase] = useState<"idle" | "play" | "still">("idle");

  useEffect(() => {
    const reduced =
      typeof window !== "undefined" &&
      window.matchMedia("(prefers-reduced-motion: reduce)").matches;

    if (reduced) {
      setPhase("still");
      return;
    }

    if (rippleDelay > 0) {
      setPhase("still");
      const startTimer = setTimeout(() => {
        setPhase("idle");
        requestAnimationFrame(() => setPhase("play"));
      }, rippleDelay);
      return () => clearTimeout(startTimer);
    }

    setPhase("idle");
    const startTimer = setTimeout(() => setPhase("play"), 0);
    return () => clearTimeout(startTimer);
  }, [rippleDelay]);

  useEffect(() => {
    if (phase !== "play") return;
    const doneTimer = setTimeout(() => setPhase("still"), variant.durationMs);
    return () => clearTimeout(doneTimer);
  }, [phase, variant.durationMs]);

  const bounceStyle = {
    "--bounce-peak-y": `${variant.peakY}px`,
    "--bounce-dip-y": `${variant.dipY}px`,
    "--bounce-rebound-y": `${variant.reboundY}px`,
    "--bounce-peak-scale": variant.peakScale,
    "--bounce-start-scale": variant.startScale,
    "--bounce-duration": `${variant.durationMs}ms`,
    transformOrigin: variant.origin,
  } as React.CSSProperties;

  return (
    <div
      data-bounce={phase}
      className={cn("bounce-card", glassCard, className)}
      style={bounceStyle}
      {...props}
    >
      {children}
    </div>
  );
}
