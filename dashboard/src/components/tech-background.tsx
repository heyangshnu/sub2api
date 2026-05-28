/**
 * 全站浅色科技风背景：白灰基底 + 淡色光晕 + 细网格（与登录 / 仪表盘一致）。
 */
export function TechBackground() {
  return (
    <>
      <div
        className="pointer-events-none fixed inset-0 -z-10 bg-[#f4f6fb]"
        aria-hidden
      />
      <div
        className="pointer-events-none fixed inset-0 -z-10 bg-[radial-gradient(ellipse_100%_75%_at_50%_-18%,rgba(20,184,166,0.1),transparent_52%)]"
        aria-hidden
      />
      <div
        className="pointer-events-none fixed inset-0 -z-10 bg-[radial-gradient(ellipse_55%_40%_at_100%_0%,rgba(45,212,191,0.08),transparent_48%)]"
        aria-hidden
      />
      <div
        className="pointer-events-none fixed inset-0 -z-10 bg-[radial-gradient(ellipse_50%_38%_at_0%_100%,rgba(16,185,129,0.06),transparent_48%)]"
        aria-hidden
      />
      <div
        className="pointer-events-none fixed inset-0 -z-10 bg-gradient-to-b from-white/40 via-transparent to-slate-100/80"
        aria-hidden
      />
    </>
  );
}
