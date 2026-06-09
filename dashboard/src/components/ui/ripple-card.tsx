import { cn } from "@/lib/utils";

const glassCard =
  "rounded-xl border border-slate-200/90 bg-white/90 text-sm text-slate-800 shadow-[0_4px_24px_rgba(15,23,42,0.05)] backdrop-blur-xl ring-1 ring-slate-200/80";

type RippleCardProps = React.ComponentProps<"div">;

export function RippleCard({ className, children, ...props }: RippleCardProps) {
  return (
    <div className={cn(glassCard, className)} {...props}>
      {children}
    </div>
  );
}
