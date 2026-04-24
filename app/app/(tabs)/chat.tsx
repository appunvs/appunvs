import { useState, useRef, useEffect } from 'react';
import {
  View, Text, TextInput, FlatList, Pressable, KeyboardAvoidingView, Platform, StyleSheet,
} from 'react-native';
import { useSafeAreaInsets } from 'react-native-safe-area-context';

import { useChatStore, type ChatMessage } from '@/state/chat';
import { useActiveBox } from '@/state/box';
import { sendChatTurn } from '@/lib/ai';

// Chat tab — input box + transcript.  Streams ChatTurnEvent frames in via
// the ai client; UI is intentionally minimal so the relay-side agent loop
// can be exercised before any styling work.
export default function ChatScreen() {
  const insets = useSafeAreaInsets();
  const messages = useChatStore((s) => s.messages);
  const append = useChatStore((s) => s.append);
  const updateLast = useChatStore((s) => s.updateLast);
  const activeBox = useActiveBox();
  const [draft, setDraft] = useState('');
  const listRef = useRef<FlatList<ChatMessage>>(null);

  useEffect(() => {
    listRef.current?.scrollToEnd({ animated: true });
  }, [messages.length]);

  const onSend = async () => {
    const text = draft.trim();
    if (!text) return;
    setDraft('');
    append({ id: crypto.randomUUID(), role: 'user', text, ts: Date.now() });
    const turnId = crypto.randomUUID();
    append({ id: turnId, role: 'assistant', text: '', ts: Date.now(), pending: true });

    try {
      for await (const frame of sendChatTurn({ box_id: activeBox?.box_id, text })) {
        if (frame.token?.text) {
          updateLast((m) => ({ ...m, text: m.text + frame.token!.text }));
        } else if (frame.finished) {
          updateLast((m) => ({ ...m, pending: false }));
        } else if (frame.error) {
          updateLast((m) => ({ ...m, text: m.text + `\n[error] ${frame.error!.error}`, pending: false }));
        }
      }
    } catch (err) {
      updateLast((m) => ({ ...m, text: m.text + `\n[transport error] ${String(err)}`, pending: false }));
    }
  };

  return (
    <KeyboardAvoidingView
      behavior={Platform.OS === 'ios' ? 'padding' : undefined}
      style={[styles.root, { paddingTop: insets.top }]}
    >
      <FlatList
        ref={listRef}
        contentContainerStyle={styles.list}
        data={messages}
        keyExtractor={(m) => m.id}
        renderItem={({ item }) => (
          <View style={[styles.bubble, item.role === 'user' ? styles.userBubble : styles.aiBubble]}>
            <Text style={styles.bubbleText}>{item.text || (item.pending ? '…' : '')}</Text>
          </View>
        )}
      />
      <View style={[styles.inputRow, { paddingBottom: insets.bottom + 8 }]}>
        <TextInput
          style={styles.input}
          value={draft}
          onChangeText={setDraft}
          onSubmitEditing={onSend}
          placeholder="describe a change…"
          placeholderTextColor="#888"
          multiline
        />
        <Pressable style={styles.send} onPress={onSend}>
          <Text style={styles.sendText}>Send</Text>
        </Pressable>
      </View>
    </KeyboardAvoidingView>
  );
}

const styles = StyleSheet.create({
  root:        { flex: 1, backgroundColor: '#0b0d10' },
  list:        { padding: 12, gap: 8 },
  bubble:      { padding: 10, borderRadius: 10, maxWidth: '85%' },
  userBubble:  { alignSelf: 'flex-end', backgroundColor: '#2e6cdf' },
  aiBubble:    { alignSelf: 'flex-start', backgroundColor: '#222831' },
  bubbleText:  { color: '#f4f5f7', fontSize: 15 },
  inputRow:    { flexDirection: 'row', padding: 8, borderTopWidth: 1, borderTopColor: '#1f2329', gap: 8 },
  input:       { flex: 1, color: '#f4f5f7', backgroundColor: '#161a20', padding: 10, borderRadius: 8, minHeight: 40, maxHeight: 120 },
  send:        { paddingHorizontal: 16, justifyContent: 'center', backgroundColor: '#2e6cdf', borderRadius: 8 },
  sendText:    { color: '#fff', fontWeight: '600' },
});
