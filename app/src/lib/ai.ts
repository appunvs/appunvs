// Streaming client for the AI agent endpoint.  The relay-side surface is
// still being designed (will land as POST /ai/turn returning an SSE / chunked
// stream of ChatTurnEvent frames); for now sendChatTurn yields a single
// stub frame so the Chat UI can be exercised in isolation.
//
// When the real endpoint lands, this file becomes a fetch + SSE reader and
// the rest of the app does not change.
import type { ChatTurnEvent } from '@/proto/chat';

export interface SendChatTurn {
  box_id?: string;
  text: string;
}

export async function* sendChatTurn(req: SendChatTurn): AsyncGenerator<ChatTurnEvent> {
  // Stub: emit a single faux assistant token + a finished frame so the
  // chat UI animates and the transcript renders end-to-end without the
  // relay AI route existing yet.
  await new Promise((r) => setTimeout(r, 80));
  yield {
    turn_id: 'stub',
    token: { text: `[stub] received "${req.text}" for box=${req.box_id ?? 'none'}` },
  };
  await new Promise((r) => setTimeout(r, 40));
  yield { turn_id: 'stub', finished: { stop_reason: 'end_turn' } };
}
