"use client";

import { ct } from "@/lib/console-typography";
import { RippleCard } from "@/components/ui/ripple-card";
import { cn } from "@/lib/utils";

type PanelCardProps = {
  title?: string;
  description?: string;
  action?: React.ReactNode;
  children: React.ReactNode;
  className?: string;
  contentClassName?: string;
};

export function PanelCard({
  title,
  description,
  action,
  children,
  className,
  contentClassName,
}: PanelCardProps) {
  return (
    <RippleCard className={className}>
      <div className={cn("relative z-10 p-5 md:p-5", contentClassName)}>
        {(title || description || action) && (
          <div className="mb-4 flex flex-col gap-2 sm:flex-row sm:items-center sm:justify-between">
            <div className="min-w-0">
              {title ? <h2 className={ct.panelTitle}>{title}</h2> : null}
              {description ? <p className={ct.panelDesc}>{description}</p> : null}
            </div>
            {action ? <div className="shrink-0">{action}</div> : null}
          </div>
        )}
        {children}
      </div>
    </RippleCard>
  );
}
