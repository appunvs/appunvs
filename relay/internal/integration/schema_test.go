package integration_test

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"testing"
	"time"

	"github.com/gorilla/websocket"

	"github.com/appunvs/appunvs/relay/internal/pb"
)

// schemaSignupOut is the signup response shape; duplicated here so this test
// file stays self-contained and doesn't collide with other integration files.
type schemaSignupOut struct {
	UserID       string `json:"user_id"`
	SessionToken string `json:"session_token"`
}

// schemaSignup signs up a fresh user and returns {user_id, session_token}.
func schemaSignup(t *testing.T, rig *accountTestRig, email string) schemaSignupOut {
	t.Helper()
	resp := postJSON(t, rig.server.URL+"/auth/signup",
		map[string]string{"email": email, "password": "hunter22"}, "")
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != 200 {
		t.Fatalf("signup %s: status %d", email, resp.StatusCode)
	}
	var out schemaSignupOut
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		t.Fatalf("decode signup: %v", err)
	}
	return out
}

// schemaRegisterDevice POSTs /auth/register and returns the device token.
func schemaRegisterDevice(t *testing.T, rig *accountTestRig, session, deviceID string) string {
	t.Helper()
	r := postJSON(t, rig.server.URL+"/auth/register",
		map[string]string{"device_id": deviceID, "platform": "browser"}, session)
	defer func() { _ = r.Body.Close() }()
	if r.StatusCode != 200 {
		t.Fatalf("register: %d", r.StatusCode)
	}
	var reg pb.RegisterResponse
	if err := json.NewDecoder(r.Body).Decode(&reg); err != nil {
		t.Fatalf("decode register: %v", err)
	}
	return reg.Token
}

// schemaDial upgrades a /ws connection with the given device token.
func schemaDial(t *testing.T, rig *accountTestRig, tok string) *websocket.Conn {
	t.Helper()
	u, _ := url.Parse(rig.server.URL)
	u.Scheme = "ws"
	u.Path = "/ws"
	q := u.Query()
	q.Set("token", tok)
	u.RawQuery = q.Encode()
	c, _, err := websocket.DefaultDialer.Dial(u.String(), nil)
	if err != nil {
		t.Fatalf("ws dial: %v", err)
	}
	return c
}

// schemaReadOne reads a single message or returns ok=false on timeout.
func schemaReadOne(c *websocket.Conn, within time.Duration) (pb.Message, bool) {
	_ = c.SetReadDeadline(time.Now().Add(within))
	_, data, err := c.ReadMessage()
	if err != nil {
		return pb.Message{}, false
	}
	var m pb.Message
	if err := json.Unmarshal(data, &m); err != nil {
		return pb.Message{}, false
	}
	return m, true
}

// TestSchemaHTTPSurface exercises the full /schema/... REST tree: create,
// list, add column, delete column, delete table. Happy-path only.
func TestSchemaHTTPSurface(t *testing.T) {
	rig := newAccountRig(t)
	u := schemaSignup(t, rig, "schema@example.com")

	// Create table.
	resp := postJSON(t, rig.server.URL+"/schema/tables",
		map[string]string{"name": "todos"}, u.SessionToken)
	if resp.StatusCode != 200 {
		t.Fatalf("create table: %d", resp.StatusCode)
	}
	_ = resp.Body.Close()

	// Duplicate → 409.
	resp = postJSON(t, rig.server.URL+"/schema/tables",
		map[string]string{"name": "todos"}, u.SessionToken)
	if resp.StatusCode != 409 {
		t.Fatalf("duplicate table: %d, want 409", resp.StatusCode)
	}
	_ = resp.Body.Close()

	// Reserved name → 400.
	resp = postJSON(t, rig.server.URL+"/schema/tables",
		map[string]string{"name": "_meta"}, u.SessionToken)
	if resp.StatusCode != 400 {
		t.Fatalf("reserved table: %d, want 400", resp.StatusCode)
	}
	_ = resp.Body.Close()

	// Add a required column.
	resp = postJSON(t, rig.server.URL+"/schema/tables/todos/columns",
		map[string]interface{}{"name": "title", "type": "text", "required": true}, u.SessionToken)
	if resp.StatusCode != 200 {
		t.Fatalf("add column: %d", resp.StatusCode)
	}
	_ = resp.Body.Close()

	// List tables and assert the column came back.
	req, _ := http.NewRequest("GET", rig.server.URL+"/schema/tables", nil)
	req.Header.Set("Authorization", "Bearer "+u.SessionToken)
	lresp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	defer func() { _ = lresp.Body.Close() }()
	if lresp.StatusCode != 200 {
		t.Fatalf("list: %d", lresp.StatusCode)
	}
	var tables []struct {
		Name    string `json:"name"`
		Columns []struct {
			Name     string `json:"name"`
			Type     string `json:"type"`
			Required bool   `json:"required"`
		} `json:"columns"`
	}
	if err := json.NewDecoder(lresp.Body).Decode(&tables); err != nil {
		t.Fatalf("decode list: %v", err)
	}
	if len(tables) != 1 || tables[0].Name != "todos" {
		t.Fatalf("tables = %+v", tables)
	}
	if len(tables[0].Columns) != 1 || tables[0].Columns[0].Name != "title" ||
		tables[0].Columns[0].Type != "text" || !tables[0].Columns[0].Required {
		t.Fatalf("columns = %+v", tables[0].Columns)
	}

	// DELETE the column.
	dreq, _ := http.NewRequest("DELETE", rig.server.URL+"/schema/tables/todos/columns/title", nil)
	dreq.Header.Set("Authorization", "Bearer "+u.SessionToken)
	dresp, err := http.DefaultClient.Do(dreq)
	if err != nil {
		t.Fatalf("delete column: %v", err)
	}
	if dresp.StatusCode != 204 {
		t.Fatalf("delete column: %d", dresp.StatusCode)
	}
	_ = dresp.Body.Close()

	// DELETE the table.
	dreq, _ = http.NewRequest("DELETE", rig.server.URL+"/schema/tables/todos", nil)
	dreq.Header.Set("Authorization", "Bearer "+u.SessionToken)
	dresp, err = http.DefaultClient.Do(dreq)
	if err != nil {
		t.Fatalf("delete table: %v", err)
	}
	if dresp.StatusCode != 204 {
		t.Fatalf("delete table: %d", dresp.StatusCode)
	}
	_ = dresp.Body.Close()
}

// TestSchemaRequiresSession: device tokens must NOT work on /schema/*.
func TestSchemaRequiresSession(t *testing.T) {
	rig := newAccountRig(t)
	u := schemaSignup(t, rig, "dev@example.com")
	deviceTok := schemaRegisterDevice(t, rig, u.SessionToken, "dX")

	resp := postJSON(t, rig.server.URL+"/schema/tables",
		map[string]string{"name": "stuff"}, deviceTok)
	if resp.StatusCode != 401 {
		t.Fatalf("device token on /schema/tables: status=%d, want 401", resp.StatusCode)
	}
	_ = resp.Body.Close()
}

// TestSchemaUpsertValidation: provider upsert is dropped unless the table is
// declared AND every required column is present.  Approach: seed the table
// via the store directly (so no broadcasts consume seq), then verify the
// behavior from the WS side.
func TestSchemaUpsertValidation(t *testing.T) {
	rig := newAccountRig(t)
	u := schemaSignup(t, rig, "valid@example.com")
	devTok := schemaRegisterDevice(t, rig, u.SessionToken, "d1")

	// Seed schema without broadcasts.
	sc := rig.st.Schema()
	if _, err := sc.CreateTable(context.Background(), u.UserID, "todos"); err != nil {
		t.Fatalf("seed table: %v", err)
	}
	if _, err := sc.AddColumn(context.Background(), u.UserID, "todos", "title", "text", true); err != nil {
		t.Fatalf("seed col: %v", err)
	}

	c := schemaDial(t, rig, devTok)
	defer func() { _ = c.Close() }()

	// 1) Missing required "title" → dropped. Use a second short-lived
	// connection as a control: whatever seq the server assigns to the
	// next valid message is what the sender actually sees, so we don't
	// have to measure timeouts of drops.
	drop := fmt.Sprintf(
		`{"namespace":%q,"role":"provider","op":"upsert","table":"todos","payload":{"id":"t1"},"ts":%d}`,
		u.UserID, time.Now().UnixMilli())
	if err := c.WriteMessage(websocket.TextMessage, []byte(drop)); err != nil {
		t.Fatalf("write drop1: %v", err)
	}
	// 2) Unknown table → dropped.
	drop = fmt.Sprintf(
		`{"namespace":%q,"role":"provider","op":"upsert","table":"ghost","payload":{"id":"x"},"ts":%d}`,
		u.UserID, time.Now().UnixMilli())
	if err := c.WriteMessage(websocket.TextMessage, []byte(drop)); err != nil {
		t.Fatalf("write drop2: %v", err)
	}
	// 3) Type-mismatch: title expected text, got number.
	drop = fmt.Sprintf(
		`{"namespace":%q,"role":"provider","op":"upsert","table":"todos","payload":{"id":"t2","title":42},"ts":%d}`,
		u.UserID, time.Now().UnixMilli())
	if err := c.WriteMessage(websocket.TextMessage, []byte(drop)); err != nil {
		t.Fatalf("write drop3: %v", err)
	}

	// 4) Valid upsert → MUST be the first thing we read, seq=1, because
	// every earlier send should have been dropped pre-sequencer.
	good := fmt.Sprintf(
		`{"namespace":%q,"role":"provider","op":"upsert","table":"todos","payload":{"id":"t1","title":"buy milk"},"ts":%d}`,
		u.UserID, time.Now().UnixMilli())
	if err := c.WriteMessage(websocket.TextMessage, []byte(good)); err != nil {
		t.Fatalf("write valid: %v", err)
	}
	m, ok := schemaReadOne(c, 3*time.Second)
	if !ok {
		t.Fatalf("valid upsert was dropped (no echo)")
	}
	if m.Op != pb.OpUpsert {
		t.Fatalf("op = %s", m.Op.String())
	}
	if m.Seq != 1 {
		t.Fatalf("seq = %d, want 1 (earlier writes should have been dropped)", m.Seq)
	}

	// And no second message should be queued (everything else was dropped).
	if extra, ok := schemaReadOne(c, 300*time.Millisecond); ok {
		t.Fatalf("unexpected extra message: %+v", extra)
	}
}

// TestSchemaBroadcastReachesAllDevices: mutating schema via HTTP fans out to
// every live WS connection for the mutating user (and ONLY that user).
func TestSchemaBroadcastReachesAllDevices(t *testing.T) {
	rig := newAccountRig(t)
	a := schemaSignup(t, rig, "multi@example.com")
	devA := schemaRegisterDevice(t, rig, a.SessionToken, "dA1")
	devB := schemaRegisterDevice(t, rig, a.SessionToken, "dA2")

	// A second user so we can assert cross-user isolation on schema events.
	other := schemaSignup(t, rig, "other@example.com")
	devO := schemaRegisterDevice(t, rig, other.SessionToken, "dO1")

	cA := schemaDial(t, rig, devA)
	cB := schemaDial(t, rig, devB)
	cO := schemaDial(t, rig, devO)
	defer func() { _ = cA.Close(); _ = cB.Close(); _ = cO.Close() }()

	// Create a table as user A.
	r := postJSON(t, rig.server.URL+"/schema/tables",
		map[string]string{"name": "projects"}, a.SessionToken)
	if r.StatusCode != 200 {
		t.Fatalf("create: %d", r.StatusCode)
	}
	_ = r.Body.Close()

	for _, name := range []string{"A", "B"} {
		var c *websocket.Conn
		if name == "A" {
			c = cA
		} else {
			c = cB
		}
		m, ok := schemaReadOne(c, 2*time.Second)
		if !ok {
			t.Fatalf("%s: missed schema broadcast", name)
		}
		if m.Op != pb.OpTableCreate {
			t.Fatalf("%s: op = %s", name, m.Op.String())
		}
		if m.Table != "_schema" {
			t.Fatalf("%s: table = %q, want _schema", name, m.Table)
		}
		var p struct {
			Name string `json:"name"`
		}
		if err := json.Unmarshal(m.Payload, &p); err != nil || p.Name != "projects" {
			t.Fatalf("%s: payload = %s err=%v", name, m.Payload, err)
		}
	}
	// The other user must NOT see it.
	if m, ok := schemaReadOne(cO, 500*time.Millisecond); ok {
		t.Fatalf("cross-user leak: other saw %+v", m)
	}

	// Drop the table — again, A's two devices see table_delete, O sees nothing.
	dreq, _ := http.NewRequest("DELETE", rig.server.URL+"/schema/tables/projects", nil)
	dreq.Header.Set("Authorization", "Bearer "+a.SessionToken)
	dresp, err := http.DefaultClient.Do(dreq)
	if err != nil {
		t.Fatalf("delete: %v", err)
	}
	_ = dresp.Body.Close()
	for _, name := range []string{"A", "B"} {
		var c *websocket.Conn
		if name == "A" {
			c = cA
		} else {
			c = cB
		}
		m, ok := schemaReadOne(c, 2*time.Second)
		if !ok {
			t.Fatalf("%s: missed table_delete", name)
		}
		if m.Op != pb.OpTableDelete {
			t.Fatalf("%s: op = %s", name, m.Op.String())
		}
	}
	if m, ok := schemaReadOne(cO, 500*time.Millisecond); ok {
		t.Fatalf("cross-user leak (delete): %+v", m)
	}
}

// TestSchemaCrossUserIsolation: user A declares table T; user B's upsert
// against T must be dropped (T doesn't exist for B).
func TestSchemaCrossUserIsolation(t *testing.T) {
	rig := newAccountRig(t)

	a := schemaSignup(t, rig, "ax@example.com")
	b := schemaSignup(t, rig, "bx@example.com")
	devB := schemaRegisterDevice(t, rig, b.SessionToken, "dBx1")

	// A declares "secrets"; seed via store so no seq is consumed.
	if _, err := rig.st.Schema().CreateTable(context.Background(), a.UserID, "secrets"); err != nil {
		t.Fatalf("seed a.secrets: %v", err)
	}
	// B has NO "secrets".  B's upsert must be dropped.
	c := schemaDial(t, rig, devB)
	defer func() { _ = c.Close() }()

	body := fmt.Sprintf(
		`{"namespace":%q,"role":"provider","op":"upsert","table":"secrets","payload":{"id":"x"},"ts":%d}`,
		b.UserID, time.Now().UnixMilli())
	if err := c.WriteMessage(websocket.TextMessage, []byte(body)); err != nil {
		t.Fatalf("write: %v", err)
	}
	if m, ok := schemaReadOne(c, 500*time.Millisecond); ok {
		t.Fatalf("cross-user upsert should have been dropped; got %+v", m)
	}
}
