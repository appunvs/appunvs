// Package integration contains end-to-end tests that drive the relay's HTTP
// and WebSocket surface against a real Redis.  Redis is expected to be
// reachable at $APPUNVS_TEST_REDIS (default localhost:6379).  The test
// flushes the DB before running so run it against a scratch instance.
package integration_test

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"

	"github.com/appunvs/appunvs/relay/internal/auth"
	"github.com/appunvs/appunvs/relay/internal/handler"
	"github.com/appunvs/appunvs/relay/internal/hub"
	"github.com/appunvs/appunvs/relay/internal/pb"
	"github.com/appunvs/appunvs/relay/internal/sequencer"
	"github.com/appunvs/appunvs/relay/internal/stream"
)

type testServer struct {
	server *httptest.Server
	signer *auth.Signer
	rdb    *redis.Client
}

func newTestServer(t *testing.T) *testServer {
	t.Helper()

	addr := os.Getenv("APPUNVS_TEST_REDIS")
	if addr == "" {
		addr = "localhost:6379"
	}
	rdb := redis.NewClient(&redis.Options{Addr: addr})
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	if err := rdb.Ping(ctx).Err(); err != nil {
		t.Skipf("redis unavailable at %s: %v", addr, err)
	}
	if err := rdb.FlushDB(ctx).Err(); err != nil {
		t.Fatalf("flushdb: %v", err)
	}

	log := zap.NewNop()
	signer, err := auth.NewSigner("", "", "appunvs-test", "appunvs-test", time.Hour, time.Hour, log)
	if err != nil {
		t.Fatalf("signer: %v", err)
	}
	h := hub.New(log)
	seq := sequencer.New(rdb)
	store := stream.New(rdb, 1000)

	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.GET("/health", func(c *gin.Context) { c.String(http.StatusOK, "ok") })
	// These tests bypass /auth/signup + /auth/register and craft device tokens
	// directly via signer.Issue, so no account routes are wired here.
	r.GET("/ws", handler.WS(handler.Deps{
		Signer: signer,
		Hub:    h,
		Seq:    seq,
		Stream: store,
		Log:    log,
	}))

	ts := httptest.NewServer(r)
	t.Cleanup(func() {
		ts.Close()
		_ = rdb.Close()
	})
	return &testServer{server: ts, signer: signer, rdb: rdb}
}

// dial opens an authenticated WebSocket using a token crafted for the given
// user/device pair.  lastSeq=0 skips catch-up.
func (ts *testServer) dial(t *testing.T, userID, deviceID string, lastSeq int64) *websocket.Conn {
	t.Helper()
	tok, err := ts.signer.Issue(userID, deviceID, "browser")
	if err != nil {
		t.Fatalf("issue token: %v", err)
	}
	u, _ := url.Parse(ts.server.URL)
	u.Scheme = "ws"
	u.Path = "/ws"
	q := u.Query()
	q.Set("token", tok)
	if lastSeq > 0 {
		q.Set("last_seq", fmt.Sprintf("%d", lastSeq))
	}
	u.RawQuery = q.Encode()

	c, _, err := websocket.DefaultDialer.Dial(u.String(), nil)
	if err != nil {
		t.Fatalf("ws dial %s: %v", u.String(), err)
	}
	t.Cleanup(func() { _ = c.Close() })
	return c
}

func sendMsg(t *testing.T, c *websocket.Conn, m pb.Message) {
	t.Helper()
	_ = c.SetWriteDeadline(time.Now().Add(2 * time.Second))
	body, err := json.Marshal(&m)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	if err := c.WriteMessage(websocket.TextMessage, body); err != nil {
		t.Fatalf("write: %v", err)
	}
}

func readMsg(t *testing.T, c *websocket.Conn, within time.Duration) pb.Message {
	t.Helper()
	_ = c.SetReadDeadline(time.Now().Add(within))
	_, data, err := c.ReadMessage()
	if err != nil {
		t.Fatalf("read: %v", err)
	}
	var m pb.Message
	if err := json.Unmarshal(data, &m); err != nil {
		t.Fatalf("decode: %v: %s", err, data)
	}
	return m
}

// readOrTimeout returns (msg, true) on success or (zero, false) on timeout.
// Used to assert that a message DOES NOT arrive within a window.
func readOrTimeout(c *websocket.Conn, within time.Duration) (pb.Message, bool) {
	_ = c.SetReadDeadline(time.Now().Add(within))
	_, data, err := c.ReadMessage()
	if err != nil {
		var nerr net.Error
		if errors.As(err, &nerr) && nerr.Timeout() {
			return pb.Message{}, false
		}
		return pb.Message{}, false
	}
	var m pb.Message
	_ = json.Unmarshal(data, &m)
	return m, true
}

// TestProviderBroadcast: A and B are providers in the same namespace.
// A publishes two upserts.  Both A and B must receive each broadcast with
// monotonic seq 1,2 — the sender is included so it can confirm its own seq.
func TestProviderBroadcast(t *testing.T) {
	ts := newTestServer(t)

	userID := "user_alpha"
	a := ts.dial(t, userID, "deviceA", 0)
	b := ts.dial(t, userID, "deviceB", 0)

	payload := json.RawMessage(`{"id":"r1","data":"hello"}`)
	sendMsg(t, a, pb.Message{
		Namespace: userID,
		Role:      pb.RoleProvider,
		Op:        pb.OpUpsert,
		Table:     "records",
		Payload:   payload,
		TS:        time.Now().UnixMilli(),
	})

	for _, name := range []string{"B", "A"} {
		var c *websocket.Conn
		if name == "A" {
			c = a
		} else {
			c = b
		}
		got := readMsg(t, c, 2*time.Second)
		if got.Seq != 1 {
			t.Fatalf("%s: first broadcast seq = %d, want 1", name, got.Seq)
		}
		if got.DeviceID != "deviceA" {
			t.Fatalf("%s: device_id = %q, want deviceA", name, got.DeviceID)
		}
		if string(got.Payload) != string(payload) {
			t.Fatalf("%s: payload = %s, want %s", name, got.Payload, payload)
		}
	}

	sendMsg(t, a, pb.Message{
		Namespace: userID, Role: pb.RoleProvider, Op: pb.OpUpsert,
		Table: "records", Payload: json.RawMessage(`{"id":"r2","data":"world"}`),
		TS: time.Now().UnixMilli(),
	})
	for _, name := range []string{"B", "A"} {
		var c *websocket.Conn
		if name == "A" {
			c = a
		} else {
			c = b
		}
		got := readMsg(t, c, 2*time.Second)
		if got.Seq != 2 {
			t.Fatalf("%s: second broadcast seq = %d, want 2", name, got.Seq)
		}
	}
}

// TestNamespaceIsolation: two different user_ids must NOT see each other.
func TestNamespaceIsolation(t *testing.T) {
	ts := newTestServer(t)
	a := ts.dial(t, "user_one", "deviceA", 0)
	b := ts.dial(t, "user_two", "deviceB", 0)

	sendMsg(t, a, pb.Message{
		Namespace: "user_one", Role: pb.RoleProvider, Op: pb.OpUpsert,
		Table: "records", Payload: json.RawMessage(`{"id":"secret"}`),
		TS: time.Now().UnixMilli(),
	})
	if m, ok := readOrTimeout(b, 500*time.Millisecond); ok {
		t.Fatalf("cross-namespace leak: %+v", m)
	}
}

// TestCatchup: B receives seq 1,2, disconnects, A publishes seq 3.
// B reconnects with last_seq=2 and must replay seq 3 before any live traffic.
func TestCatchup(t *testing.T) {
	ts := newTestServer(t)
	userID := "user_catchup"
	a := ts.dial(t, userID, "deviceA", 0)
	b := ts.dial(t, userID, "deviceB", 0)

	for i := 0; i < 2; i++ {
		sendMsg(t, a, pb.Message{
			Namespace: userID, Role: pb.RoleProvider, Op: pb.OpUpsert, Table: "records",
			Payload: json.RawMessage(fmt.Sprintf(`{"id":"r%d"}`, i+1)),
			TS:      time.Now().UnixMilli(),
		})
		got := readMsg(t, b, 2*time.Second)
		if got.Seq != int64(i+1) {
			t.Fatalf("pre-catchup seq = %d, want %d", got.Seq, i+1)
		}
	}

	_ = b.Close()
	// give the relay a moment to unregister
	time.Sleep(50 * time.Millisecond)

	sendMsg(t, a, pb.Message{
		Namespace: userID, Role: pb.RoleProvider, Op: pb.OpUpsert, Table: "records",
		Payload: json.RawMessage(`{"id":"r3"}`),
		TS:      time.Now().UnixMilli(),
	})

	// Reconnect B with last_seq=2.
	b2 := ts.dial(t, userID, "deviceB", 2)
	replayed := readMsg(t, b2, 2*time.Second)
	if replayed.Seq != 3 {
		t.Fatalf("catchup seq = %d, want 3", replayed.Seq)
	}
	if !strings.Contains(string(replayed.Payload), `"r3"`) {
		t.Fatalf("catchup payload = %s, missing r3", replayed.Payload)
	}
}

// TestConnectorToProvider: a connector (C) sends op=upsert; it must reach the
// provider (P) but not other connectors, and the relay does NOT assign a seq.
func TestConnectorToProvider(t *testing.T) {
	ts := newTestServer(t)
	userID := "user_conn"
	p := ts.dial(t, userID, "deviceP", 0)
	c := ts.dial(t, userID, "deviceC", 0)

	// Arm P as a provider so the filter matches.  (Role is assigned per
	// message on the relay side.)  We do this by having P send one provider
	// message; both P (self-echo) and C must receive seq=1.
	sendMsg(t, p, pb.Message{
		Namespace: userID, Role: pb.RoleProvider, Op: pb.OpUpsert, Table: "records",
		Payload: json.RawMessage(`{"id":"p0"}`), TS: time.Now().UnixMilli(),
	})
	if got := readMsg(t, c, 2*time.Second); got.Seq != 1 {
		t.Fatalf("warmup (C) seq = %d, want 1", got.Seq)
	}
	if got := readMsg(t, p, 2*time.Second); got.Seq != 1 {
		t.Fatalf("warmup (P self-echo) seq = %d, want 1", got.Seq)
	}

	// Now C sends a connector op.  P should receive it; the message must not
	// carry a relay-assigned seq.
	sendMsg(t, c, pb.Message{
		Namespace: userID, Role: pb.RoleConnector, Op: pb.OpDelete, Table: "records",
		Payload: json.RawMessage(`{"id":"p0"}`), TS: time.Now().UnixMilli(),
	})
	got := readMsg(t, p, 2*time.Second)
	if got.Role != pb.RoleConnector {
		t.Fatalf("role = %q, want connector", got.Role.String())
	}
	if got.Op != pb.OpDelete {
		t.Fatalf("op = %q, want delete", got.Op.String())
	}
	if got.Seq != 0 {
		t.Fatalf("connector forward should not carry seq; got %d", got.Seq)
	}
}
