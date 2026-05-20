"use client";

import { MessageSquarePlus, Trash2 } from "lucide-react";
import { cn } from "@/lib/utils";
import type { ChatSession } from "@/lib/chat-sessions";

type Props = {
  sessions: ChatSession[];
  activeId: string | null;
  onNewChat: () => void;
  onSelect: (id: string) => void;
  onDelete: (id: string) => void;
};

export function ChatSidebar({ sessions, activeId, onNewChat, onSelect, onDelete }: Props) {
  return (
    <aside className="flex w-[260px] shrink-0 flex-col border-r border-slate-200/80 bg-[#f5f5f7]">
      <div className="border-b border-slate-200/60 p-3">
        <button
          type="button"
          onClick={onNewChat}
          className="flex w-full items-center justify-center gap-2 rounded-2xl bg-[#E8F4FF] px-4 py-2.5 text-sm font-medium text-slate-800 ring-1 ring-[#d6e8ff] transition-colors hover:bg-[#dcecff]"
        >
          <MessageSquarePlus className="size-4" />
          新对话
        </button>
      </div>
      <nav className="flex-1 overflow-y-auto p-2">
        {sessions.length === 0 ? (
          <p className="px-3 py-6 text-center text-xs text-slate-500">暂无历史对话</p>
        ) : (
          <ul className="space-y-0.5">
            {sessions.map((s) => (
              <li key={s.id} className="group relative">
                <button
                  type="button"
                  onClick={() => onSelect(s.id)}
                  className={cn(
                    "w-full rounded-lg px-3 py-2.5 pr-9 text-left text-sm transition-colors",
                    activeId === s.id
                      ? "bg-white text-slate-900 shadow-sm ring-1 ring-slate-200/80"
                      : "text-slate-700 hover:bg-white/70"
                  )}
                >
                  <span className="line-clamp-2 leading-snug">{s.title}</span>
                </button>
                <button
                  type="button"
                  aria-label="删除对话"
                  onClick={(e) => {
                    e.stopPropagation();
                    onDelete(s.id);
                  }}
                  className="absolute right-1.5 top-1/2 -translate-y-1/2 rounded-md p-1.5 text-slate-400 opacity-0 transition-opacity hover:bg-slate-200 hover:text-red-600 group-hover:opacity-100"
                >
                  <Trash2 className="size-3.5" />
                </button>
              </li>
            ))}
          </ul>
        )}
      </nav>
    </aside>
  );
}
