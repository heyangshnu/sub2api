export type BounceVariant = {
  peakY: number;
  dipY: number;
  reboundY: number;
  peakScale: number;
  startScale: number;
  durationMs: number;
  origin: string;
};

function rand(min: number, max: number): number {
  return min + Math.random() * (max - min);
}

/** Per-card random bounce so tiles do not move identically. */
export function createBounceVariant(): BounceVariant {
  const origins = ["center bottom", "50% 85%", "45% 90%", "55% 88%"] as const;
  return {
    peakY: -rand(8, 17),
    dipY: rand(3, 9),
    reboundY: -rand(2, 7),
    peakScale: rand(1.03, 1.09),
    startScale: rand(0.84, 0.93),
    durationMs: Math.round(rand(560, 780)),
    origin: origins[Math.floor(Math.random() * origins.length)],
  };
}
