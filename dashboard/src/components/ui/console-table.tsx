import { ct } from "@/lib/console-typography";
import { cn } from "@/lib/utils";

export function ConsoleTable({
  className,
  children,
  ...props
}: React.ComponentProps<"table">) {
  return (
    <table className={cn("w-full text-left", ct.tableWrap, className)} {...props}>
      {children}
    </table>
  );
}

export function ConsoleTableHead({ className, ...props }: React.ComponentProps<"thead">) {
  return <thead className={cn("bg-slate-50/90", className)} {...props} />;
}

export function ConsoleTh({ className, ...props }: React.ComponentProps<"th">) {
  return <th className={cn("px-4 py-2.5", ct.tableHead, className)} {...props} />;
}

export function ConsoleTd({
  className,
  variant = "default",
  ...props
}: React.ComponentProps<"td"> & { variant?: "default" | "mono" | "strong" | "muted" }) {
  const variantClass =
    variant === "mono"
      ? ct.tableCellMono
      : variant === "strong"
        ? ct.tableCellStrong
        : variant === "muted"
          ? ct.tableCellMuted
          : ct.tableCell;
  return <td className={cn("px-4 py-2.5", variantClass, className)} {...props} />;
}
