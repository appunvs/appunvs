// Theme override store — `null` means "follow system color scheme",
// `'light' | 'dark'` is an explicit user choice in Profile.  Persisted to
// AsyncStorage so the override survives app restarts; falls back to
// in-memory when AsyncStorage isn't installed (e.g. earliest-boot before
// hydration).
import { create } from 'zustand';
import AsyncStorage from '@react-native-async-storage/async-storage';

const KEY = 'appunvs.theme.override';

export type ThemeOverride = 'light' | 'dark' | null;

interface ThemeOverrideState {
  override: ThemeOverride;
  hydrated: boolean;
  set: (next: ThemeOverride) => Promise<void>;
  hydrate: () => Promise<void>;
}

export const useThemeOverrideStore = create<ThemeOverrideState>((set) => ({
  override: null,
  hydrated: false,
  set: async (next) => {
    set({ override: next });
    if (next === null) {
      await AsyncStorage.removeItem(KEY);
    } else {
      await AsyncStorage.setItem(KEY, next);
    }
  },
  hydrate: async () => {
    try {
      const raw = await AsyncStorage.getItem(KEY);
      const override: ThemeOverride =
        raw === 'light' || raw === 'dark' ? raw : null;
      set({ override, hydrated: true });
    } catch {
      set({ hydrated: true });
    }
  },
}));
