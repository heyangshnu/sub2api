/** Center grow → brief hold → shrink fly to top compact bar */
export const SLOGAN_GROW_MS = 1000;
export const SLOGAN_HOLD_MS = 350;
export const SLOGAN_FLY_MS = 980;
export const SLOGAN_HERO_MS = SLOGAN_GROW_MS + SLOGAN_HOLD_MS + SLOGAN_FLY_MS;
/** Whole-card bounce animation length (ms) — Hammer-style icon pop */
export const CARD_BOUNCE_MS = 680;

/** Set on login; consumed when usage page plays hero once */
export const SLOGAN_AFTER_LOGIN_KEY = "sub2api_slogan_after_login";
/** Set after hero finishes (or skipped) — prevents replay when re-entering usage */
export const SLOGAN_PLAYED_KEY = "sub2api_slogan_played";

export function flagSloganAfterLogin(): void {
  if (typeof window === "undefined") return;
  sessionStorage.removeItem(SLOGAN_PLAYED_KEY);
  sessionStorage.setItem(SLOGAN_AFTER_LOGIN_KEY, "1");
}

export function clearSloganAfterLoginFlag(): void {
  if (typeof window === "undefined") return;
  sessionStorage.removeItem(SLOGAN_AFTER_LOGIN_KEY);
}

export function consumeSloganAfterLoginFlag(): boolean {
  if (typeof window === "undefined") return false;
  if (sessionStorage.getItem(SLOGAN_AFTER_LOGIN_KEY) !== "1") return false;
  sessionStorage.removeItem(SLOGAN_AFTER_LOGIN_KEY);
  return true;
}

export function hasSloganPlayed(): boolean {
  if (typeof window === "undefined") return false;
  return sessionStorage.getItem(SLOGAN_PLAYED_KEY) === "1";
}

export function markSloganPlayed(): void {
  if (typeof window === "undefined") return;
  sessionStorage.setItem(SLOGAN_PLAYED_KEY, "1");
  clearSloganAfterLoginFlag();
}

export function clearSloganSession(): void {
  if (typeof window === "undefined") return;
  sessionStorage.removeItem(SLOGAN_AFTER_LOGIN_KEY);
  sessionStorage.removeItem(SLOGAN_PLAYED_KEY);
}
