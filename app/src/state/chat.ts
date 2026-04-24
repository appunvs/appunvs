import { create } from 'zustand';

export interface ChatMessage {
  id: string;
  role: 'user' | 'assistant' | 'system';
  text: string;
  ts: number;
  pending?: boolean;
}

interface ChatState {
  messages: ChatMessage[];
  append: (m: ChatMessage) => void;
  updateLast: (mut: (m: ChatMessage) => ChatMessage) => void;
  clear: () => void;
}

export const useChatStore = create<ChatState>((set) => ({
  messages: [],
  append: (m) => set((s) => ({ messages: [...s.messages, m] })),
  updateLast: (mut) => set((s) => {
    if (s.messages.length === 0) return s;
    const next = s.messages.slice();
    next[next.length - 1] = mut(next[next.length - 1]);
    return { messages: next };
  }),
  clear: () => set({ messages: [] }),
}));
