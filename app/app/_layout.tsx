import { Stack } from 'expo-router';
import { GestureHandlerRootView } from 'react-native-gesture-handler';
import { SafeAreaProvider } from 'react-native-safe-area-context';
import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import { useColorScheme, View } from 'react-native';
import { StatusBar } from 'expo-status-bar';

import { ThemeProvider, useTheme } from '@/theme';

const queryClient = new QueryClient({
  defaultOptions: {
    queries: {
      // Boxes don't change often during a session; keep the cache warm
      // so reopening the switcher feels instant. Mutations on /box
      // explicitly invalidate this.
      staleTime: 30_000,
      retry: 1,
    },
  },
});

export default function RootLayout() {
  return (
    <GestureHandlerRootView style={{ flex: 1 }}>
      <SafeAreaProvider>
        <QueryClientProvider client={queryClient}>
          <ThemeProvider>
            <ThemedShell />
          </ThemeProvider>
        </QueryClientProvider>
      </SafeAreaProvider>
    </GestureHandlerRootView>
  );
}

// ThemedShell sits inside ThemeProvider so the StatusBar style and the
// root background can react to scheme changes without a flicker.
function ThemedShell() {
  const theme = useTheme();
  const system = useColorScheme();
  // Status-bar style needs to invert from background; light scheme →
  // dark icons, dark scheme → light icons.
  const statusBarStyle = theme.scheme === 'light' ? 'dark' : 'light';
  // Keep useColorScheme() referenced so it triggers re-render on system
  // changes even when the user has not set an override yet.
  void system;

  return (
    <View style={{ flex: 1, backgroundColor: theme.colors.bgPage }}>
      <StatusBar style={statusBarStyle} />
      <Stack
        screenOptions={{
          headerShown: false,
          animation: 'fade',
          contentStyle: { backgroundColor: theme.colors.bgPage },
        }}
      />
    </View>
  );
}
