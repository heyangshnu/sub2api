import type { ChatMessage } from "@/lib/chat-stream";

export type ChatSession = {
  id: string;
  title: string;
  messages: ChatMessage[];
  model: string;
  updatedAt: number;
};

export function defaultMessages(): ChatMessage[] {
  return [
    {
      role: "assistant",
      content: "Hi, I'm the Sub2API assistant. How can I help you today?",
    },
  ];
}

export function titleFromMessages(messages: ChatMessage[]): string {
  const firstUser = messages.find((m) => m.role === "user");
  if (!firstUser) return "New chat";
  const text = typeof firstUser.content === "string" ? firstUser.content : "";
  const trimmed = text.trim().slice(0, 40);
  return trimmed || "New chat";
}

export function createSessionId(): string {
  return `chat_${Date.now()}_${Math.random().toString(36).slice(2, 9)}`;
}

const STORAGE_KEY = "sub2api_chat_sessions";

export function loadSessions(): ChatSession[] {
  if (typeof window === "undefined") return [];
  try {
    const raw = localStorage.getItem(STORAGE_KEY);
    if (!raw) return [];
    const parsed = JSON.parse(raw) as ChatSession[];
    return Array.isArray(parsed) ? parsed : [];
  } catch {
    return [];
  }
}

export function saveSessions(sessions: ChatSession[]) {
  localStorage.setItem(STORAGE_KEY, JSON.stringify(sessions));
}

export function upsertSession(session: ChatSession, sessions: ChatSession[]): ChatSession[] {
  const idx = sessions.findIndex((s) => s.id === session.id);
  if (idx >= 0) {
    const next = [...sessions];
    next[idx] = session;
    return next.sort((a, b) => b.updatedAt - a.updatedAt);
  }
  return [session, ...sessions].sort((a, b) => b.updatedAt - a.updatedAt);
}

export function removeSession(id: string, sessions: ChatSession[]): ChatSession[] {
  return sessions.filter((s) => s.id !== id);
}
