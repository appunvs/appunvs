import { Tabs } from 'expo-router';
import { useWindowDimensions, Platform } from 'react-native';

// Single tabs layout with side-bar on browser/desktop and bottom-bar on
// mobile.  The breakpoint is 720dp — narrower than that we treat the
// surface as "phone" regardless of native platform, which keeps responsive
// browser windows behaving like the mobile shell.
//
// react-navigation's bottom-tab navigator (which expo-router defaults to)
// supports `tabBarPosition: 'left'` since v7.  We rely on that here so we
// don't have to hand-roll a second navigator.
export default function TabsLayout() {
  const { width } = useWindowDimensions();
  const sideTabs = (Platform.OS === 'web' || Platform.OS === 'macos' || Platform.OS === 'windows') && width >= 720;

  return (
    <Tabs
      screenOptions={{
        headerShown: false,
        // @ts-expect-error tabBarPosition exists in v7; types may lag in the SDK ship.
        tabBarPosition: sideTabs ? 'left' : 'bottom',
        tabBarStyle: sideTabs ? { width: 200 } : undefined,
      }}
    >
      <Tabs.Screen name="chat"    options={{ title: 'Chat' }} />
      <Tabs.Screen name="stage"   options={{ title: 'Stage' }} />
      <Tabs.Screen name="profile" options={{ title: 'Profile' }} />
    </Tabs>
  );
}
