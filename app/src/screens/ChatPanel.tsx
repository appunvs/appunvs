// ChatPanel — the chat conversation surface, used by:
//
//   - the Chat tab (always)
//   - the Stage tab on wide screens (never, currently — Stage tab is
//     full-bleed for focus mode)
//
// The Chat tab on wide screens renders this side-by-side with StagePanel.
// Keep this component layout-neutral: no own padding around the outer
// edges; lay out happens in the parent.
import { useEffect, useRef, useState } from 'react';
import {
  View,
  FlatList,
  KeyboardAvoidingView,
  Platform,
  type StyleProp,
  type ViewStyle,
} from 'react-native';
import { useSafeAreaInsets } from 'react-native-safe-area-context';
import { useRouter } from 'expo-router';

import { useChatStore, type ChatMessage } from '@/state/chat';
import { useActiveBox } from '@/state/box';
import { sendChatTurn } from '@/lib/ai';
import { useTheme } from '@/theme';
import { Bubble, BoxSwitcher, EmptyState } from '@/components';
import { Button, Input, Text } from '@/ui';

export interface ChatPanelProps {
  /** When true, the box switcher chip in the header is shown. The
   *  Stage tab on wide screens may host this panel without the chip
   *  in a future iteration; current screens always pass true. */
  showSwitcher?: boolean;
  style?: StyleProp<ViewStyle>;
}

export function ChatPanel({ showSwitcher = true, style }: ChatPanelProps) {
  const theme = useTheme();
  const insets = useSafeAreaInsets();
  const messages = useChatStore((s) => s.messages);
  const append = useChatStore((s) => s.append);
  const updateLast = useChatStore((s) => s.updateLast);
  const activeBox = useActiveBox();
  const [draft, setDraft] = useState('');
  const [sending, setSending] = useState(false);
  const listRef = useRef<FlatList<ChatMessage>>(null);

  useEffect(() => {
    listRef.current?.scrollToEnd({ animated: true });
  }, [messages.length]);

  const onSend = async () => {
    const text = draft.trim();
    if (!text || sending) return;
    setSending(true);
    setDraft('');

    append({ id: cryptoRandomID(), role: 'user', text, ts: Date.now() });
    const turnId = cryptoRandomID();
    append({ id: turnId, role: 'assistant', text: '', ts: Date.now(), pending: true });

    try {
      for await (const frame of sendChatTurn({ box_id: activeBox?.box_id, text })) {
        if (frame.token?.text) {
          updateLast((m) => ({ ...m, text: m.text + frame.token!.text }));
        } else if (frame.finished) {
          updateLast((m) => ({ ...m, pending: false }));
        } else if (frame.error) {
          updateLast((m) => ({
            ...m,
            text: m.text + `\n[error] ${frame.error!.error}`,
            pending: false,
          }));
        }
      }
    } catch (err) {
      updateLast((m) => ({
        ...m,
        text: m.text + `\n[transport] ${String(err)}`,
        pending: false,
      }));
    } finally {
      setSending(false);
    }
  };

  if (!activeBox) {
    return (
      <View style={[{ flex: 1, backgroundColor: theme.colors.bgPage }, style]}>
        <Header showSwitcher={showSwitcher} />
        <EmptyState
          title="选个 Box 开始"
          hint="每个 Box 是一个独立项目，对话历史与代码都和它绑定。从下方新建一个，或扫码加载别人的。"
          action={<NewBoxLink />}
        />
      </View>
    );
  }

  return (
    <KeyboardAvoidingView
      behavior={Platform.OS === 'ios' ? 'padding' : undefined}
      style={[{ flex: 1, backgroundColor: theme.colors.bgPage }, style]}
    >
      <Header showSwitcher={showSwitcher} />
      <FlatList
        ref={listRef}
        contentContainerStyle={{
          padding: theme.spacing.l,
          gap: theme.spacing.s,
        }}
        data={messages}
        keyExtractor={(m) => m.id}
        renderItem={({ item }) => renderMessage(item)}
        ListEmptyComponent={
          <View style={{ paddingTop: theme.spacing.huge, alignItems: 'center' }}>
            <Text color="textSecondary">和 AI 说点什么，比如"做一个计数器 app"。</Text>
          </View>
        }
      />
      <View
        style={{
          flexDirection: 'row',
          padding: theme.spacing.s,
          gap: theme.spacing.s,
          paddingBottom: insets.bottom + theme.spacing.s,
          borderTopWidth: 1,
          borderTopColor: theme.colors.borderDefault,
          backgroundColor: theme.colors.bgCard,
        }}
      >
        <Input
          value={draft}
          onChangeText={setDraft}
          onSubmitEditing={onSend}
          placeholder="描述一个改动…"
          multiline
          style={{ flex: 1 }}
        />
        <Button
          label="发送"
          onPress={onSend}
          disabled={!draft.trim()}
          loading={sending}
          size="md"
        />
      </View>
    </KeyboardAvoidingView>
  );
}

function Header({ showSwitcher }: { showSwitcher: boolean }) {
  const theme = useTheme();
  const insets = useSafeAreaInsets();
  return (
    <View
      style={{
        paddingTop: insets.top + theme.spacing.s,
        paddingHorizontal: theme.spacing.l,
        paddingBottom: theme.spacing.s,
        flexDirection: 'row',
        alignItems: 'center',
        gap: theme.spacing.s,
        borderBottomWidth: 1,
        borderBottomColor: theme.colors.borderDefault,
        backgroundColor: theme.colors.bgPage,
      }}
    >
      {showSwitcher
        ? <BoxSwitcher />
        : <Text variant="h3">Chat</Text>}
      <View style={{ flex: 1 }} />
      {/* Reserved slot — future: New chat icon, share button, etc. */}
    </View>
  );
}

function NewBoxLink() {
  const router = useRouter();
  return (
    <Button label="新建 Box" onPress={() => router.push('/box/new')} />
  );
}

function renderMessage(item: ChatMessage) {
  // V1 chat store only carries text bubbles. Tool-call timelines land
  // as a third role once /ai/turn SSE is wired; until then the engine
  // flattens tool events into assistant text.
  return <Bubble role={item.role} text={item.text} pending={item.pending} />;
}

// Use globalThis.crypto.randomUUID when available; fall back to a
// short pseudo-random hex.  Hermes 0.76+ exposes randomUUID natively.
function cryptoRandomID(): string {
  const c = (globalThis as { crypto?: Crypto }).crypto;
  if (c?.randomUUID) return c.randomUUID();
  return 'm_' + Math.random().toString(36).slice(2, 12);
}
