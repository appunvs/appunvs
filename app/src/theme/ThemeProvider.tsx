// ThemeProvider — wraps every screen.  Resolves the active scheme as:
//
//   1. user override from `useThemeOverrideStore` (Profile setting), or
//   2. `useColorScheme()` (system / browser dark-mode), or
//   3. `'dark'` as the v1 default for unset cases.
//
// Hydrates the override from AsyncStorage on first mount so a returning
// user sees their preferred scheme without a flash of the wrong one.

import { createContext, useContext, useEffect, useMemo, type ReactNode } from 'react';
import { useColorScheme } from 'react-native';

import { useThemeOverrideStore } from './store';
import { darkTheme, lightTheme, type Theme } from './tokens';

const ThemeContext = createContext<Theme>(darkTheme);

export function ThemeProvider({ children }: { children: ReactNode }) {
  const system = useColorScheme();
  const override = useThemeOverrideStore((s) => s.override);
  const hydrate = useThemeOverrideStore((s) => s.hydrate);

  useEffect(() => {
    void hydrate();
  }, [hydrate]);

  const theme = useMemo<Theme>(() => {
    const scheme: 'light' | 'dark' = override ?? (system ?? 'dark');
    return scheme === 'light' ? lightTheme : darkTheme;
  }, [override, system]);

  return <ThemeContext.Provider value={theme}>{children}</ThemeContext.Provider>;
}

export function useTheme(): Theme {
  return useContext(ThemeContext);
}
