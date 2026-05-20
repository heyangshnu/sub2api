import { clsx, type ClassValue } from "clsx"
import { twMerge } from "tailwind-merge"

export function cn(...inputs: ClassValue[]) {
  return twMerge(clsx(inputs))
}

/** Safe USD display; avoids crash when API omits or nulls numeric fields. */
export function formatUsd(value: unknown, digits = 4): string {
  const n = typeof value === "number" && Number.isFinite(value) ? value : 0
  return `$${n.toFixed(digits)}`
}
