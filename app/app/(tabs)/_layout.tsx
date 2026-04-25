import { Tabs } from 'expo-router';
import { useWindowDimensions, Platform } from 'react-native';

import { useTheme } from '@/theme';

// Single tabs layout.  Three tabs stay constant across breakpoints so a
// resize never makes a tab disappear.  Layout style switches:
//
//   < 720dp  bottom tabbar, every tab full-screen
//   ≥ 720dp  side tabbar; Chat tab internally renders a Stage panel
//            on the right, Stage tab stays full-bleed (focus mode)
//
// react-navigation's bottom-tab navigator (which expo-router defaults to)
// supports `tabBarPosition: 'left'` since v7.
export default function TabsLayout() {
  const { width } = useWindowDimensions();
  const theme = useTheme();
  const sideTabs =
    (Platform.OS === 'web' || Platform.OS === 'macos' || Platform.OS === 'windows') &&
    width >= 720;

  return (
    <Tabs
      screenOptions={{
        headerShown: false,
        tabBarStyle: {
          backgroundColor: theme.colors.bgCard,
          borderTopColor: theme.colors.borderDefault,
          borderRightColor: theme.colors.borderDefault,
          ...(sideTabs ? { width: 220 } : { height: 60 }),
        },
        tabBarActiveTintColor: theme.colors.brandDark,
        tabBarInactiveTintColor: theme.colors.textSecondary,
        tabBarLabelStyle: {
          fontSize: 13,
          fontWeight: '600',
        },
        // @ts-expect-error tabBarPosition exists in react-navigation v7;
        // types may lag depending on SDK pin.
        tabBarPosition: sideTabs ? 'left' : 'bottom',
      }}
    >
      <Tabs.Screen name="chat"    options={{ title: 'Chat' }} />
      <Tabs.Screen name="stage"   options={{ title: 'Stage' }} />
      <Tabs.Screen name="profile" options={{ title: 'Profile' }} />
    </Tabs>
  );
}
