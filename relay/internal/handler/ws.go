// Package handler wires the /ws endpoint: it authenticates the token,
// replays the Redis Stream from last_seq, then enters the live loop.
package handler

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
	"go.uber.org/zap"

	"github.com/appunvs/appunvs/relay/internal/auth"
	"github.com/appunvs/appunvs/relay/internal/billing"
	"github.com/appunvs/appunvs/relay/internal/hub"
	"github.com/appunvs/appunvs/relay/internal/pb"
	"github.com/appunvs/appunvs/relay/internal/sequencer"
	"github.com/appunvs/appunvs/relay/internal/store"
	"github.com/appunvs/appunvs/relay/internal/stream"
)

// upgrader permits any origin; production should narrow this in the reverse
// proxy layer (TLS termination also happens there).
var upgrader = websocket.Upgrader{
	ReadBufferSize:  4096,
	WriteBufferSize: 4096,
	CheckOrigin:     func(*http.Request) bool { return true },
}

// Deps is the set of collaborators the /ws handler needs.
type Deps struct {
	Signer *auth.Signer
	Hub    *hub.Hub
	Seq    *sequencer.Seq
	Stream *stream.Store
	// Store is the persistent relay state. When non-nil the WS handler
	// validates provider op=upsert/delete against the user's declared
	// schema (app_tables + app_columns).  Nil (as in the early integration
	// rigs) disables validation for back-compat.
	Store *store.Store
	// Quota is the billing gate. When non-nil every provider broadcast runs
	// through CheckAndIncrement before we assign a seq or fan out; on
	// ErrQuotaExceeded we send the sender an op=quota_exceeded envelope and
	// drop the message. Nil leaves the path unchanged — useful for the
	// bring-your-own-store integration rigs that predate billing.
	Quota billing.QuotaGate
	Log   *zap.Logger
}

// WS builds a gin handler for GET /ws.
func WS(d Deps) gin.HandlerFunc {
	return func(c *gin.Context) {
		token := c.Query("token")
		if token == "" {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "missing token"})
			return
		}
		claims, err := d.Signer.Verify(token, auth.TokenDevice)
		if err != nil {
			d.Log.Info("ws auth failed", zap.Error(err))
			c.JSON(http.StatusUnauthorized, gin.H{"error": "bad token"})
			return
		}
		lastSeq := int64(0)
		if q := c.Query("last_seq"); q != "" {
			if n, err := strconv.ParseInt(q, 10, 64); err == nil && n > 0 {
				lastSeq = n
			}
		}

		ns := claims.UserID // namespace defaults to user_id
		ws, err := upgrader.Upgrade(c.Writer, c.Request, nil)
		if err != nil {
			d.Log.Info("ws upgrade failed", zap.Error(err))
			return
		}
		// Role is unspecified until the first message identifies it;
		// broadcasts targeting providers/connectors will simply skip the
		// socket until then.  Clients set role on every Message per the wire
		// protocol.
		conn := hub.NewConn(ws, claims.DeviceID, claims.UserID, ns, pb.RoleUnspecified, d.Log)
		d.Hub.Register(conn)

		// Catch-up replay before registering readers.
		if lastSeq > 0 {
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			msgs, err := d.Stream.Range(ctx, ns, lastSeq)
			cancel()
			if err != nil {
				d.Log.Warn("ws catchup failed", zap.String("namespace", ns), zap.Error(err))
			} else {
				for _, m := range msgs {
					if err := conn.SendBlocking(m); err != nil {
						break
					}
				}
				d.Log.Info("ws catchup",
					zap.String("device_id", claims.DeviceID),
					zap.String("namespace", ns),
					zap.Int64("from", lastSeq),
					zap.Int("count", len(msgs)))
			}
		}

		// WriteLoop runs in a goroutine; ReadLoop owns this request.
		go conn.WriteLoop()
		conn.ReadLoop(makeRead(d))
		// Read exited -> socket gone.
		d.Hub.Unregister(conn)
		_ = conn.Close()
	}
}

// makeRead returns the per-conn inbound handler.
func makeRead(d Deps) hub.ReadHandler {
	return func(c *hub.Conn, msg *pb.Message) {
		// Authoritative identity comes from the JWT; clients cannot spoof.
		msg.DeviceID = c.DeviceID
		msg.UserID = c.UserID
		if msg.Namespace == "" {
			msg.Namespace = c.Namespace
		}
		// Refresh cached role from each message; it's the authoritative signal
		// for this connection's current behavior.
		c.Role = msg.Role

		if msg.Namespace != c.Namespace {
			d.Log.Info("ws: dropping cross-namespace message",
				zap.String("device_id", c.DeviceID),
				zap.String("msg_ns", msg.Namespace),
				zap.String("conn_ns", c.Namespace))
			return
		}

		ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
		defer cancel()

		if msg.Role.IsProvider() {
			// Schema validation: only gate real data ops.  Schema-change
			// ops (table_create / column_add / ...) bypass validation
			// because they target the meta-table "_schema".  The HTTP
			// surface is the only legitimate source for those ops, but
			// we don't reject them here — we just don't consult the
			// schema for them.
			if d.Store != nil && (msg.Op == pb.OpUpsert || msg.Op == pb.OpDelete) {
				if err := validateAgainstSchema(ctx, d.Store.Schema(), msg); err != nil {
					d.Log.Warn("ws: dropping invalid provider message",
						zap.String("device_id", c.DeviceID),
						zap.String("user_id", c.UserID),
						zap.String("namespace", c.Namespace),
						zap.String("op", msg.Op.String()),
						zap.String("table", msg.Table),
						zap.Error(err))
					return
				}
			}
			// Quota gate: enforce messages_per_day before we spend a seq.
			// On over-limit we echo a quota_exceeded envelope back to the
			// sender only (role=provider) and drop the inbound message.
			// Non-quota errors from the gate are logged but not fatal —
			// billing state should never take down the wire path.
			if d.Quota != nil {
				if err := d.Quota.CheckAndIncrement(ctx, c.UserID, int64(len(msg.Payload))); err != nil {
					if errors.Is(err, billing.ErrQuotaExceeded) {
						d.Log.Info("ws: quota exceeded",
							zap.String("device_id", c.DeviceID),
							zap.String("user_id", c.UserID))
						notice := &pb.Message{
							DeviceID:  c.DeviceID,
							UserID:    c.UserID,
							Namespace: c.Namespace,
							Role:      pb.RoleProvider,
							Op:        pb.OpQuotaExceeded,
							Table:     msg.Table,
							TS:        time.Now().UnixMilli(),
						}
						if err := c.SendBlocking(notice); err != nil {
							d.Log.Info("ws: notify quota failed", zap.Error(err))
						}
						return
					}
					d.Log.Warn("ws: quota gate error", zap.Error(err))
					// Fall through: a billing hiccup shouldn't drop data.
				}
			}
			// Provider path: assign seq, persist, broadcast to everyone.
			seq, err := d.Seq.Next(ctx, msg.Namespace)
			if err != nil {
				d.Log.Error("ws: seq next failed", zap.Error(err))
				return
			}
			msg.Seq = seq
			if err := d.Stream.Append(ctx, msg.Namespace, msg); err != nil {
				d.Log.Error("ws: stream append failed", zap.Error(err))
				return
			}
			d.Log.Info("broadcast provider",
				zap.String("device_id", c.DeviceID),
				zap.String("user_id", c.UserID),
				zap.String("namespace", c.Namespace),
				zap.Int64("seq", seq),
				zap.String("table", msg.Table))
			// Include the sender so it learns its own seq — per protocol:
			// "广播给同 namespace 下所有在线设备（含发送者自己，用于确认 seq）".
			d.Hub.Broadcast(msg, hub.AllRoles, nil)
			return
		}

		// Connector path: forward to providers only; no seq, no persistence.
		d.Log.Info("broadcast connector",
			zap.String("device_id", c.DeviceID),
			zap.String("user_id", c.UserID),
			zap.String("namespace", c.Namespace),
			zap.String("op", msg.Op.String()),
			zap.String("table", msg.Table))
		d.Hub.Broadcast(msg, hub.ProvidersOnly, c)
	}
}

// validateAgainstSchema enforces the user's declared schema on an inbound
// provider message:
//
//   - op=delete:  payload.id must be a non-empty string.
//   - op=upsert:  the target table must exist for this user; payload must
//     decode as a JSON object; every required column must be present; every
//     present key that matches a declared column must conform to its type.
//     Unknown keys are allowed and silently accepted (logged at debug).
//
// The implicit primary key "id: text" is required on every upsert.  The
// reserved meta-table "_schema" is rejected here: clients MUST go through
// the HTTP surface for schema mutations.
func validateAgainstSchema(ctx context.Context, sc *store.Schema, msg *pb.Message) error {
	if msg.Table == "" {
		return errors.New("missing table")
	}
	if msg.Table == "_schema" {
		return errors.New("table \"_schema\" is reserved for schema broadcasts")
	}

	if msg.Op == pb.OpDelete {
		var payload map[string]interface{}
		if len(msg.Payload) == 0 {
			return errors.New("delete: missing payload")
		}
		if err := json.Unmarshal(msg.Payload, &payload); err != nil {
			return err
		}
		id, ok := payload["id"].(string)
		if !ok || id == "" {
			return errors.New("delete: payload.id must be a non-empty string")
		}
		// Still require the table to be declared for the user so a
		// stale client can't issue deletes against tables the user has
		// dropped.
		if _, err := sc.GetTable(ctx, msg.UserID, msg.Table); err != nil {
			return err
		}
		return nil
	}

	// OpUpsert.
	tbl, err := sc.GetTable(ctx, msg.UserID, msg.Table)
	if err != nil {
		return err
	}
	var payload map[string]json.RawMessage
	if len(msg.Payload) == 0 {
		return errors.New("upsert: missing payload")
	}
	if err := json.Unmarshal(msg.Payload, &payload); err != nil {
		return err
	}
	// Implicit primary key.
	idRaw, ok := payload["id"]
	if !ok {
		return errors.New("upsert: payload.id is required")
	}
	var idStr string
	if err := json.Unmarshal(idRaw, &idStr); err != nil || idStr == "" {
		return errors.New("upsert: payload.id must be a non-empty string")
	}

	// Required columns.
	for _, col := range tbl.Columns {
		if col.Required {
			if _, present := payload[col.Name]; !present {
				return fmt.Errorf("upsert: missing required column %q", col.Name)
			}
		}
	}
	// Type-check every known column that's present.
	colIndex := make(map[string]store.Column, len(tbl.Columns))
	for _, col := range tbl.Columns {
		colIndex[col.Name] = col
	}
	for k, raw := range payload {
		if k == "id" {
			continue
		}
		col, known := colIndex[k]
		if !known {
			// Extra keys are allowed in v1 (logged at debug by the
			// caller).  Skip type-checking unknown fields.
			continue
		}
		if err := checkColumnType(col.Type, raw); err != nil {
			return fmt.Errorf("upsert: column %q: %w", k, err)
		}
	}
	return nil
}

// checkColumnType validates a single JSON value against the declared type.
// null is accepted for any type (it means "unset"); required columns still
// have to be PRESENT, but may be null.
func checkColumnType(typ string, raw json.RawMessage) error {
	if len(raw) == 0 {
		return errors.New("empty value")
	}
	// Null is always acceptable (used as "unset").
	if string(raw) == "null" {
		return nil
	}
	var v interface{}
	if err := json.Unmarshal(raw, &v); err != nil {
		return err
	}
	switch typ {
	case store.ColumnTypeText:
		if _, ok := v.(string); !ok {
			return errors.New("expected string")
		}
	case store.ColumnTypeNumber:
		if _, ok := v.(float64); !ok {
			return errors.New("expected number")
		}
	case store.ColumnTypeBool:
		if _, ok := v.(bool); !ok {
			return errors.New("expected bool")
		}
	case store.ColumnTypeJSON:
		// Anything that decoded is acceptable for "json".
	default:
		return fmt.Errorf("unknown column type %q", typ)
	}
	return nil
}
