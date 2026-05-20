import type { ChatMessage } from "@/lib/chat-stream";

export type ChatSession = {
  id: string;
  title: string;
  messages: ChatMessage[];
  model: string;
  updatedAt: number;
};

const STORAGE_KEY = "sub2api_chat_sessions_v1";

const DEFAULT_GREETING: ChatMessage = {
  role: "assistant",
  content: "你好，我是 Sub2API 助手。有什么可以帮你？",
};

export function createSessionId(): string {
  return `s_${Date.now()}_${Math.random().toString(36).slice(2, 9)}`;
}

export function defaultMessages(): ChatMessage[] {
  return [{ ...DEFAULT_GREETING }];
}

export function titleFromMessages(messages: ChatMessage[]): string {
  const firstUser = messages.find((m) => m.role === "user" && m.content.trim());
  if (!firstUser) return "新对话";
  const t = firstUser.content.trim().replace(/\s+/g, " ");
  return t.length > 28 ? `${t.slice(0, 28)}…` : t;
}

export function loadSessions(): ChatSession[] {
  if (typeof window === "undefined") return [];
  try {
    const raw = localStorage.getItem(STORAGE_KEY);
    if (!raw) return [];
    const parsed = JSON.parse(raw) as ChatSession[];
    return Array.isArray(parsed)
      ? parsed.sort((a, b) => b.updatedAt - a.updatedAt)
      : [];
  } catch {
    return [];
  }
}

export function saveSessions(sessions: ChatSession[]): void {
  if (typeof window === "undefined") return;
  localStorage.setItem(STORAGE_KEY, JSON.stringify(sessions));
}

export function upsertSession(session: ChatSession, sessions: ChatSession[]): ChatSession[] {
  const idx = sessions.findIndex((s) => s.id === session.id);
  const next = idx >= 0 ? [...sessions] : [session, ...sessions];
  if (idx >= 0) next[idx] = session;
  else next[0] = session;
  return next.sort((a, b) => b.updatedAt - a.updatedAt);
}

export function removeSession(id: string, sessions: ChatSession[]): ChatSession[] {
  return sessions.filter((s) => s.id !== id);
}
