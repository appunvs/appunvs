// Hand-mirrored ChatTurnEvent shape from shared/proto/appunvs.proto.  Keep
// in sync with relay/internal/ai/ai.go; both pull from the same proto.
//
// Exactly one of the optional fields is non-nil per frame.
export interface ChatTurnEvent {
  turn_id: string;
  token?:    { text: string };
  tool_call?: { call_id: string; name: string; args_json: string };
  tool_res?:  { call_id: string; result_json: string; is_error?: boolean };
  finished?:  { stop_reason: string; tokens_in?: number; tokens_out?: number };
  error?:     { error: string };
}
