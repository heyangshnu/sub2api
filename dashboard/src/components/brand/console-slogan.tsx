"use client";

import {
  createContext,
  useCallback,
  useContext,
  useEffect,
  useLayoutEffect,
  useRef,
  useState,
} from "react";
import { createPortal } from "react-dom";
import { useAuth } from "@/lib/auth-context";
import {
  consumeSloganAfterLoginFlag,
  hasSloganPlayed,
  markSloganPlayed,
  SLOGAN_FLY_MS,
  SLOGAN_GROW_MS,
  SLOGAN_HERO_MS,
  SLOGAN_HOLD_MS,
} from "@/lib/brand";
import { ct } from "@/lib/console-typography";
import { useLocale } from "@/lib/i18n";
import { cn } from "@/lib/utils";

type Phase = "idle" | "grow" | "fly" | "pinned";

type ConsoleSloganContextValue = {
  isPlaying: boolean;
  isPinned: boolean;
};

const ConsoleSloganContext = createContext<ConsoleSloganContextValue>({
  isPlaying: false,
  isPinned: false,
});

export function useConsoleSlogan() {
  return useContext(ConsoleSloganContext);
}

function screenCenter() {
  return { x: window.innerWidth / 2, y: window.innerHeight / 2 };
}

const BACKDROP_CLASS =
  "absolute inset-0 bg-white/25 backdrop-blur-xl backdrop-saturate-150 transition-opacity duration-500";

export function ConsoleSloganLayout({
  pageTitle,
  headerLeft,
  headerRight,
  mainClassName,
  children,
}: {
  pageTitle: string;
  headerLeft?: React.ReactNode;
  headerRight: React.ReactNode;
  mainClassName?: string;
  children: React.ReactNode;
}) {
  const { messages } = useLocale();
  const { isAuthenticated, isGuest, sloganPlayId, setSloganPinned } = useAuth();
  const words = messages.slogan.words;
  const sub = messages.slogan.sub;
  const sloganLine = words.join(" · ");

  const [mounted, setMounted] = useState(false);
  const [phase, setPhase] = useState<Phase>("idle");
  const [heroRunId, setHeroRunId] = useState(0);
  const lastPlayedIdRef = useRef(0);
  const compactTargetRef = useRef<HTMLParagraphElement>(null);
  const flyBlockRef = useRef<HTMLDivElement>(null);
  const flyAnimRef = useRef<Animation | null>(null);

  const isPinned = phase === "pinned";
  const isPlaying = phase === "grow" || phase === "fly";

  const startHero = useCallback(() => {
    const reduced = window.matchMedia("(prefers-reduced-motion: reduce)").matches;
    if (reduced) {
      markSloganPlayed();
      setSloganPinned(true);
      setPhase("pinned");
      return;
    }
    setHeroRunId((id) => id + 1);
    setPhase("grow");
    const toFly = window.setTimeout(() => setPhase("fly"), SLOGAN_GROW_MS + SLOGAN_HOLD_MS);
    const fallbackPinned = window.setTimeout(() => {
      markSloganPlayed();
      setSloganPinned(true);
      setPhase("pinned");
      flyAnimRef.current?.cancel();
    }, SLOGAN_HERO_MS + 120);
    return () => {
      window.clearTimeout(toFly);
      window.clearTimeout(fallbackPinned);
      flyAnimRef.current?.cancel();
    };
  }, [setSloganPinned]);

  useEffect(() => {
    setMounted(true);
  }, []);

  useEffect(() => {
    if (!mounted || isGuest) {
      setPhase("idle");
      return;
    }
    if (hasSloganPlayed()) {
      setSloganPinned(true);
      setPhase("pinned");
    }
  }, [mounted, isGuest, isAuthenticated, setSloganPinned]);

  useEffect(() => {
    if (!mounted || !isAuthenticated || isGuest) return;
    if (hasSloganPlayed()) return;

    const pending = consumeSloganAfterLoginFlag();
    const shouldPlay = pending || sloganPlayId > lastPlayedIdRef.current;
    if (!shouldPlay) return;

    lastPlayedIdRef.current = Math.max(lastPlayedIdRef.current, sloganPlayId);
    return startHero();
  }, [mounted, isAuthenticated, isGuest, sloganPlayId, startHero]);

  useLayoutEffect(() => {
    if (phase !== "fly") return;

    const block = flyBlockRef.current;
    const target = compactTargetRef.current;
    if (!block || !target) return;

    const center = screenCenter();
    const targetRect = target.getBoundingClientRect();
    const blockRect = block.getBoundingClientRect();

    const endX = targetRect.left + targetRect.width / 2;
    const endY = targetRect.top + targetRect.height / 2;
    const scale = Math.min(0.42, Math.max(0.28, targetRect.width / Math.max(blockRect.width, 1)));

    const dx = endX - center.x;
    const dy = endY - center.y;
    const midX = dx * 0.42;
    const midY = dy * 0.28 - Math.min(72, Math.abs(dy) * 0.12 + 36);

    block.style.left = `${center.x}px`;
    block.style.top = `${center.y}px`;
    block.style.transform = "translate(-50%, -50%) scale(1)";
    block.style.opacity = "1";

    flyAnimRef.current?.cancel();
    const anim = block.animate(
      [
        { transform: "translate(-50%, -50%) scale(1)", opacity: 1, offset: 0 },
        {
          transform: `translate(calc(-50% + ${midX}px), calc(-50% + ${midY}px)) scale(0.72)`,
          opacity: 1,
          offset: 0.42,
        },
        {
          transform: `translate(calc(-50% + ${dx}px), calc(-50% + ${dy}px)) scale(${scale})`,
          opacity: 1,
          offset: 1,
        },
      ],
      {
        duration: SLOGAN_FLY_MS,
        easing: "cubic-bezier(0.33, 0.82, 0.38, 1)",
        fill: "forwards",
      }
    );
    flyAnimRef.current = anim;

    const onDone = () => {
      markSloganPlayed();
      setSloganPinned(true);
      setPhase("pinned");
      block.style.opacity = "0";
    };
    anim.addEventListener("finish", onDone);
    anim.addEventListener("cancel", () => block.style.opacity = "0");

    return () => {
      anim.removeEventListener("finish", onDone);
      anim.cancel();
    };
  }, [phase, heroRunId, setSloganPinned]);

  const showOverlay = mounted && isPlaying;
  const center = typeof window !== "undefined" ? screenCenter() : { x: 0, y: 0 };

  const overlayPortal =
    showOverlay && typeof document !== "undefined"
      ? createPortal(
          <div className="pointer-events-none fixed inset-0 z-[200]" role="presentation" aria-hidden>
            <div className={cn(BACKDROP_CLASS, phase === "fly" && "opacity-90")} />
            {phase === "grow" && (
              <div
                className="absolute z-10 flex flex-col items-center px-6 text-center"
                style={{
                  left: center.x,
                  top: center.y,
                  transform: "translate(-50%, -50%)",
                }}
              >
                <div key={`grow-${heroRunId}`} className="animate-slogan-hero-grow flex flex-col items-center">
                  <p className="text-3xl font-semibold tracking-tight text-slate-900 drop-shadow-sm md:text-5xl">
                    {sloganLine}
                  </p>
                  <p className="mt-4 max-w-md text-sm text-slate-600 md:text-base">{sub}</p>
                </div>
              </div>
            )}
            {phase === "fly" && (
              <div
                ref={flyBlockRef}
                className="slogan-fly-block absolute z-20 flex flex-col items-center whitespace-nowrap text-center will-change-transform"
                style={{ left: center.x, top: center.y }}
              >
                <p className="text-3xl font-semibold tracking-tight text-teal-900 drop-shadow-sm md:text-5xl">
                  {sloganLine}
                </p>
                <p className="mt-2 text-sm text-slate-500 md:text-base">{sub}</p>
              </div>
            )}
          </div>,
          document.body
        )
      : null;

  return (
    <ConsoleSloganContext.Provider value={{ isPlaying, isPinned }}>
      {overlayPortal}
      <div className="flex min-h-0 min-w-0 flex-1 flex-col">
        <header className="sticky top-0 z-30 border-b border-slate-200/80 bg-white/75 backdrop-blur-xl">
          <div className="flex h-14 items-center justify-between gap-3 px-4 md:px-6 lg:px-8">
            <div className="relative flex min-w-0 flex-1 items-center gap-3">
              {headerLeft}
              {(isAuthenticated || isPlaying) && (
                <p
                  ref={compactTargetRef}
                  className={cn(
                    ct.panelTitle,
                    "min-w-0 flex-1 truncate text-teal-800",
                    !isPinned && "invisible"
                  )}
                  aria-live="polite"
                >
                  <span className="font-semibold">{sloganLine}</span>
                  <span className="mx-2 hidden font-normal text-slate-400 sm:inline">·</span>
                  <span
                    className={cn(
                      "hidden font-normal sm:inline",
                      ct.panelDesc,
                      "mt-0 inline text-slate-500"
                    )}
                  >
                    {sub}
                  </span>
                </p>
              )}
              {!isPinned && !isPlaying ? (
                <h1 className="min-w-0 truncate text-base font-semibold text-slate-900">{pageTitle}</h1>
              ) : null}
            </div>
            {headerRight}
          </div>
        </header>

        <main
          className={cn(
            "console-main flex min-h-0 flex-1 flex-col px-4 py-6 md:px-6 lg:px-8 lg:py-8",
            isPlaying && "pointer-events-none",
            mainClassName
          )}
        >
          {children}
        </main>
      </div>
    </ConsoleSloganContext.Provider>
  );
}
