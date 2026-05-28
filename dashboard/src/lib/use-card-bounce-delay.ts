"use client";

import { useConsoleSlogan } from "@/components/brand/console-slogan";
import { SLOGAN_HERO_MS } from "@/lib/brand";

/** Hold card bounce only while the login slogan hero is playing. */
export function useCardBounceDelay(): number {
  const { isPlaying } = useConsoleSlogan();
  return isPlaying ? SLOGAN_HERO_MS : 0;
}
