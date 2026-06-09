import type { UserProfile } from "@/lib/api";

export type PlatformModel = {
  id: string;
  label: string;
  provider_model?: string;
  kind?: "chat" | "image";
  provider?: string;
};

/** Legacy upstream ids in CHAT_ENABLED_MODELS / subscription → platform id */
const LEGACY_PLATFORM_ID: Record<string, string> = {
  "deepseek-chat": "deepseek",
  "deepseek-coder": "deepseek",
};

export const FALLBACK_PLATFORM_MODELS: PlatformModel[] = [
  { id: "deepseek", label: "DeepSeek", kind: "chat" },
  { id: "gpt", label: "GPT", kind: "chat" },
  { id: "gemini", label: "Gemini", kind: "chat" },
  { id: "claude", label: "Claude", kind: "chat" },
  { id: "image", label: "Image", kind: "image" },
];

function normalizePlatformId(id: string): string {
  return LEGACY_PLATFORM_ID[id] ?? id;
}

export function modelLabel(catalog: PlatformModel[], id: string): string {
  return catalog.find((m) => m.id === id)?.label ?? id;
}

export function resolveSelectableModels(
  catalog: PlatformModel[],
  chatEnabledModels: string[] | undefined,
  profile: UserProfile | null | undefined,
  subscriptionsEnabled?: boolean
): PlatformModel[] {
  const base = catalog.length > 0 ? catalog : FALLBACK_PLATFORM_MODELS;
  let allowedIds =
    chatEnabledModels && chatEnabledModels.length > 0
      ? chatEnabledModels.map(normalizePlatformId)
      : base.map((m) => m.id);

  if (subscriptionsEnabled && profile?.subscription?.allowed_models?.length) {
    const subSet = new Set(profile.subscription.allowed_models.map(normalizePlatformId));
    allowedIds = allowedIds.filter((id) => subSet.has(id));
  }

  const allowedSet = new Set(allowedIds);
  const filtered = base.filter((m) => allowedSet.has(m.id));
  return filtered.length > 0 ? filtered : base.slice(0, 1);
}
