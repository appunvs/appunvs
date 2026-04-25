import { useWindowDimensions, View } from 'react-native';

import { ChatPanel } from '@/screens/ChatPanel';
import { StagePanel } from '@/screens/StagePanel';
import { useTheme } from '@/theme';

// Chat tab.
//
//   < 720dp  ChatPanel takes the whole tab.
//   ≥ 720dp  ChatPanel + StagePanel side by side, divided by a hairline.
//            Stage panel hides its own header (the parent context tells
//            the user which Box they're on) and falls back to "no
//            bundle yet" empty state when nothing is published.
export default function ChatTab() {
  const { width } = useWindowDimensions();
  const theme = useTheme();
  const sideStage = width >= 720;

  if (!sideStage) {
    return <ChatPanel />;
  }

  return (
    <View style={{ flex: 1, flexDirection: 'row', backgroundColor: theme.colors.bgPage }}>
      <ChatPanel style={{ flex: 1, minWidth: 380 }} />
      <View
        style={{
          width: 1,
          backgroundColor: theme.colors.borderDefault,
        }}
      />
      <StagePanel
        showHeader={true}
        style={{ width: 480, maxWidth: '50%' }}
      />
    </View>
  );
}
