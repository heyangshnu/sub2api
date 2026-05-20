"use client";

import { useCallback, useEffect, useRef, useState } from "react";
import { useAuth } from "@/lib/auth-context";
import { streamDashboardChat, type ChatMessage } from "@/lib/chat-stream";
import { Button } from "@/components/ui/button";
import { apiClient } from "@/lib/api";
import { ChatSidebar } from "@/components/chat/chat-sidebar";
import {
  createSessionId,
  defaultMessages,
  loadSessions,
  removeSession,
  saveSessions,
  titleFromMessages,
  upsertSession,
  type ChatSession,
} from "@/lib/chat-sessions";
import { cn } from "@/lib/utils";

const FALLBACK_MODELS = ["deepseek-chat"];

export function ChatPage() {
  const { userProfile, refreshProfile } = useAuth();
  const [models, setModels] = useState<string[]>(FALLBACK_MODELS);
  const [model, setModel] = useState(FALLBACK_MODELS[0]);
  const [sessions, setSessions] = useState<ChatSession[]>([]);
  const [activeId, setActiveId] = useState<string | null>(null);
  const [messages, setMessages] = useState<ChatMessage[]>(defaultMessages());
  const [input, setInput] = useState("");
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const bottomRef = useRef<HTMLDivElement>(null);
  const hydrated = useRef(false);

  useEffect(() => {
    void refreshProfile();
  }, [refreshProfile]);

  useEffect(() => {
    apiClient
      .getAuthConfig()
      .then((cfg) => {
        const list =
          cfg.chat_enabled_models && cfg.chat_enabled_models.length > 0
            ? cfg.chat_enabled_models
            : FALLBACK_MODELS;
        setModels(list);
        setModel((m) => (list.includes(m) ? m : list[0]));
      })
      .catch(() => {});
  }, []);

  useEffect(() => {
    const loaded = loadSessions();
    setSessions(loaded);
    if (loaded.length > 0) {
      setActiveId(loaded[0].id);
      setMessages(loaded[0].messages);
      setModel(loaded[0].model);
    } else {
      const id = createSessionId();
      const fresh: ChatSession = {
        id,
        title: "新对话",
        messages: defaultMessages(),
        model: FALLBACK_MODELS[0],
        updatedAt: Date.now(),
      };
      setActiveId(id);
      setSessions([fresh]);
      saveSessions([fresh]);
    }
    hydrated.current = true;
  }, []);

  const persist = useCallback(
    (msgs: ChatMessage[], currentModel: string, id: string | null) => {
      if (!id || !hydrated.current) return;
      setSessions((prev) => {
        const session: ChatSession = {
          id,
          title: titleFromMessages(msgs),
          messages: msgs,
          model: currentModel,
          updatedAt: Date.now(),
        };
        const next = upsertSession(session, prev);
        saveSessions(next);
        return next;
      });
    },
    []
  );

  const scrollDown = useCallback(() => {
    bottomRef.current?.scrollIntoView({ behavior: "smooth" });
  }, []);

  const selectSession = (id: string) => {
    const s = sessions.find((x) => x.id === id);
    if (!s) return;
    setActiveId(id);
    setMessages(s.messages);
    setModel(s.model);
    setError(null);
  };

  const newChat = () => {
    const id = createSessionId();
    const fresh: ChatSession = {
      id,
      title: "新对话",
      messages: defaultMessages(),
      model,
      updatedAt: Date.now(),
    };
    setActiveId(id);
    setMessages(fresh.messages);
    setError(null);
    setInput("");
    setSessions((prev) => {
      const next = upsertSession(fresh, prev);
      saveSessions(next);
      return next;
    });
  };

  const deleteChat = (id: string) => {
    const next = removeSession(id, sessions);
    if (next.length === 0) {
      newChat();
      return;
    }
    saveSessions(next);
    setSessions(next);
    if (activeId === id) {
      setActiveId(next[0].id);
      setMessages(next[0].messages);
      setModel(next[0].model);
    }
  };

  const send = async () => {
    const text = input.trim();
    if (!text || loading || !activeId) return;
    const token = apiClient.getToken();
    if (!token) {
      setError("未登录");
      return;
    }
    setError(null);
    setInput("");
    const nextMessages: ChatMessage[] = [...messages, { role: "user", content: text }];
    const withAssistant: ChatMessage[] = [...nextMessages, { role: "assistant", content: "" }];
    setMessages(withAssistant);
    setLoading(true);
    scrollDown();

    try {
      await streamDashboardChat(token, nextMessages, model, (delta) => {
        setMessages((prev) => {
          const copy = [...prev];
          const last = copy[copy.length - 1];
          if (last?.role === "assistant") {
            copy[copy.length - 1] = { ...last, content: last.content + delta };
          }
          return copy;
        });
        scrollDown();
      });
      setMessages((prev) => {
        persist(prev, model, activeId);
        return prev;
      });
      await refreshProfile();
    } catch (e) {
      setError(e instanceof Error ? e.message : "发送失败");
      setMessages((prev) => {
        const trimmed = prev.slice(0, -1);
        persist(trimmed, model, activeId);
        return trimmed;
      });
    } finally {
      setLoading(false);
    }
  };

  return (
    <div className="flex h-[calc(100vh-57px)] min-h-0 w-full">
      <ChatSidebar
        sessions={sessions}
        activeId={activeId}
        onNewChat={newChat}
        onSelect={selectSession}
        onDelete={deleteChat}
      />

      <div className="flex min-w-0 flex-1 flex-col bg-white">
        <div className="flex-1 space-y-5 overflow-y-auto px-6 py-6 md:px-10 lg:px-14">
          {messages.map((m, i) => (
            <div
              key={i}
              className={cn(
                "w-full",
                m.role === "user" ? "flex justify-end" : "flex justify-start"
              )}
            >
              <div
                className={cn(
                  "max-w-[min(100%,48rem)] rounded-2xl px-4 py-3 text-[15px] leading-relaxed",
                  m.role === "user"
                    ? "bg-[#E8F4FF] text-slate-800 ring-1 ring-[#d6e8ff]"
                    : "bg-[#f5f5f7] text-slate-800"
                )}
              >
                <p className="whitespace-pre-wrap">
                  {m.content || (loading && i === messages.length - 1 ? "…" : "")}
                </p>
              </div>
            </div>
          ))}
          <div ref={bottomRef} />
        </div>

        <div className="border-t border-slate-200/80 bg-white px-6 py-5 md:px-10 lg:px-14">
          {error && (
            <p className="mb-3 rounded-lg bg-red-50 px-3 py-2 text-sm text-red-800">{error}</p>
          )}
          <div className="w-full">
            <div className="flex w-full gap-3 rounded-2xl border border-slate-200/90 bg-[#f5f5f7] p-2.5 shadow-sm ring-1 ring-slate-100">
              <textarea
                className="min-h-[48px] flex-1 resize-none bg-transparent px-3 py-2.5 text-[15px] text-slate-900 outline-none placeholder:text-slate-400"
                placeholder="输入消息，Enter 发送，Shift+Enter 换行"
                rows={1}
                value={input}
                onChange={(e) => setInput(e.target.value)}
                onKeyDown={(e) => {
                  if (e.key === "Enter" && !e.shiftKey) {
                    e.preventDefault();
                    void send();
                  }
                }}
                disabled={loading}
              />
              <Button
                type="button"
                className="h-11 shrink-0 rounded-xl bg-[#4d6bfe] px-6 hover:bg-[#3d5ce8]"
                onClick={() => void send()}
                disabled={loading || !input.trim()}
              >
                {loading ? "…" : "发送"}
              </Button>
            </div>
            <div className="mt-3 flex items-center gap-2 text-sm text-slate-600">
              <label htmlFor="chat-model" className="shrink-0 text-slate-500">
                模型
              </label>
              <select
                id="chat-model"
                value={model}
                onChange={(e) => {
                  const next = e.target.value;
                  setModel(next);
                  if (activeId) {
                    setSessions((prev) => {
                      const idx = prev.findIndex((s) => s.id === activeId);
                      if (idx < 0) return prev;
                      const updated = { ...prev[idx], model: next, updatedAt: Date.now() };
                      const list = upsertSession(updated, prev);
                      saveSessions(list);
                      return list;
                    });
                  }
                }}
                className="rounded-lg border border-slate-200 bg-white px-3 py-1.5 text-sm text-slate-800 outline-none focus-visible:ring-2 focus-visible:ring-slate-300"
              >
                {models.map((m) => (
                  <option key={m} value={m}>
                    {m}
                  </option>
                ))}
              </select>
            </div>
          </div>
        </div>
      </div>
    </div>
  );
}
