package pb

import (
	"encoding/json"
	"fmt"
)

// messageWire is the snake_case on-the-wire shape of Message used strictly
// for JSON (de)serialization.  Enums are emitted as short lowercase strings,
// matching canonical protojson for this package.
type messageWire struct {
	Seq       int64           `json:"seq,omitempty"`
	DeviceID  string          `json:"device_id,omitempty"`
	UserID    string          `json:"user_id,omitempty"`
	Namespace string          `json:"namespace,omitempty"`
	Role      string          `json:"role,omitempty"`
	Op        string          `json:"op,omitempty"`
	Table     string          `json:"table,omitempty"`
	Payload   json.RawMessage `json:"payload,omitempty"`
	TS        int64           `json:"ts,omitempty"`
}

// MarshalJSON emits canonical protojson for Message.
func (m Message) MarshalJSON() ([]byte, error) {
	w := messageWire{
		Seq:       m.Seq,
		DeviceID:  m.DeviceID,
		UserID:    m.UserID,
		Namespace: m.Namespace,
		Table:     m.Table,
		TS:        m.TS,
	}
	if m.Role != RoleUnspecified {
		w.Role = m.Role.String()
	}
	if m.Op != OpUnspecified {
		w.Op = m.Op.String()
	}
	if len(m.Payload) > 0 {
		w.Payload = json.RawMessage(m.Payload)
	}
	return json.Marshal(w)
}

// UnmarshalJSON parses canonical protojson for Message.
func (m *Message) UnmarshalJSON(b []byte) error {
	var w messageWire
	if err := json.Unmarshal(b, &w); err != nil {
		return fmt.Errorf("pb: decode Message: %w", err)
	}
	m.Seq = w.Seq
	m.DeviceID = w.DeviceID
	m.UserID = w.UserID
	m.Namespace = w.Namespace
	m.Role = ParseRole(w.Role)
	m.Op = ParseOp(w.Op)
	m.Table = w.Table
	m.TS = w.TS
	if len(w.Payload) > 0 {
		m.Payload = append(m.Payload[:0], w.Payload...)
	} else {
		m.Payload = nil
	}
	return nil
}
