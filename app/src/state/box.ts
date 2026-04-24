import { create } from 'zustand';

import type { Box, BundleRef, BoxResponse } from '@/lib/box';

interface ActiveBoxSlice {
  box: Box | null;
  current: BundleRef | null;
}

interface ActiveBoxState extends ActiveBoxSlice {
  setActive: (next: BoxResponse | { box: Box; current?: BundleRef | null } | null) => void;
}

export const useActiveBoxStore = create<ActiveBoxState>((set) => ({
  box: null,
  current: null,
  setActive: (next) => {
    if (next === null) {
      set({ box: null, current: null });
      return;
    }
    set({ box: next.box, current: next.current ?? null });
  },
}));

// Convenience selector that gives Chat / Stage a single object to read.
export const useActiveBox = (): (Box & { current?: BundleRef | null }) | null => {
  const box = useActiveBoxStore((s) => s.box);
  const current = useActiveBoxStore((s) => s.current);
  if (!box) return null;
  return { ...box, current: current ?? undefined };
};
