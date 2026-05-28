/**
 * Console card typography (codesome-style): one scale inside tiles.
 * - Labels & body: text-sm
 * - Metric values: text-xl semibold
 * - Section titles: text-sm medium (not text-base/lg mix)
 */
export const ct = {
  statLabel: "text-sm font-normal leading-5 text-slate-500",
  statValue: "text-xl font-semibold leading-7 tracking-tight text-slate-900 tabular-nums",
  statHint: "mt-1.5 text-sm font-normal leading-5 text-slate-500",

  panelTitle: "text-sm font-medium leading-5 text-slate-900",
  panelDesc: "mt-1 text-sm font-normal leading-5 text-slate-500",

  pageDesc: "text-sm font-normal leading-relaxed text-slate-500",

  tableWrap: "text-sm",
  tableHead: "text-sm font-medium text-slate-600",
  tableCell: "text-sm font-normal text-slate-800",
  tableCellMono: "font-mono text-sm font-normal text-slate-800",
  tableCellStrong: "text-sm font-medium text-slate-900 tabular-nums",
  tableCellMuted: "text-sm font-normal text-slate-500",

  alert: "text-sm font-normal leading-relaxed text-slate-700",
  alertBrand: "text-sm font-normal leading-relaxed text-teal-900",

  empty: "text-sm font-normal text-slate-500",
} as const;
