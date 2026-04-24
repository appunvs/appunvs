package integration_test

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
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
	"github.com/appunvs/appunvs/relay/internal/store"
	"github.com/appunvs/appunvs/relay/internal/stream"
)

// accountTestRig wires the full relay (signup/login/register/ws) against a
// fresh SQLite file and the shared test Redis.
type accountTestRig struct {
	server *httptest.Server
	signer *auth.Signer
	rdb    *redis.Client
	st     *store.Store
}

func newAccountRig(t *testing.T) *accountTestRig {
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

	dbPath := filepath.Join(t.TempDir(), "test.db")
	st, err := store.Open(context.Background(), dbPath)
	if err != nil {
		t.Fatalf("store open: %v", err)
	}

	log := zap.NewNop()
	signer, err := auth.NewSigner("", "", "appunvs-test", "appunvs-test", time.Hour, time.Hour, log)
	if err != nil {
		t.Fatalf("signer: %v", err)
	}
	h := hub.New(log)
	seq := sequencer.New(rdb)
	streamSvc := stream.New(rdb, 1000)
	accountDeps := auth.Deps{Signer: signer, Store: st, Log: log}

	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.POST("/auth/signup", auth.Signup(accountDeps))
	r.POST("/auth/login", auth.Login(accountDeps))
	r.POST("/auth/register", auth.Register(accountDeps))
	r.GET("/auth/me", auth.Me(accountDeps))
	handler.RegisterSchemaRoutes(r, handler.SchemaDeps{
		Signer: signer,
		Store:  st,
		Hub:    h,
		Seq:    seq,
		Stream: streamSvc,
		Log:    log,
	})
	r.GET("/ws", handler.WS(handler.Deps{
		Signer: signer,
		Hub:    h,
		Seq:    seq,
		Stream: streamSvc,
		Store:  st,
		Log:    log,
	}))
	ts := httptest.NewServer(r)
	t.Cleanup(func() {
		ts.Close()
		_ = rdb.Close()
		_ = st.Close()
	})
	return &accountTestRig{server: ts, signer: signer, rdb: rdb, st: st}
}

func postJSON(t *testing.T, url string, bodyObj any, authHeader string) *http.Response {
	t.Helper()
	b, _ := json.Marshal(bodyObj)
	req, _ := http.NewRequest("POST", url, bytes.NewReader(b))
	req.Header.Set("content-type", "application/json")
	if authHeader != "" {
		req.Header.Set("Authorization", "Bearer "+authHeader)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("POST %s: %v", url, err)
	}
	return resp
}

// signup→login→register→ws is the happy path a real client walks on first run.
func TestAccountHappyPath(t *testing.T) {
	rig := newAccountRig(t)

	// Signup.
	resp := postJSON(t, rig.server.URL+"/auth/signup",
		map[string]string{"email": "alice@example.com", "password": "hunter22"}, "")
	if resp.StatusCode != 200 {
		t.Fatalf("signup status=%d", resp.StatusCode)
	}
	var signup struct {
		UserID       string `json:"user_id"`
		SessionToken string `json:"session_token"`
	}
	_ = json.NewDecoder(resp.Body).Decode(&signup)
	_ = resp.Body.Close()
	if signup.UserID == "" || signup.SessionToken == "" {
		t.Fatalf("signup missing fields: %+v", signup)
	}

	// /auth/me with the session token.
	req, _ := http.NewRequest("GET", rig.server.URL+"/auth/me", nil)
	req.Header.Set("Authorization", "Bearer "+signup.SessionToken)
	meResp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("me: %v", err)
	}
	if meResp.StatusCode != 200 {
		t.Fatalf("me status=%d", meResp.StatusCode)
	}
	_ = meResp.Body.Close()

	// Login with the same credentials returns a different-but-valid session.
	loginResp := postJSON(t, rig.server.URL+"/auth/login",
		map[string]string{"email": "alice@example.com", "password": "hunter22"}, "")
	if loginResp.StatusCode != 200 {
		t.Fatalf("login status=%d", loginResp.StatusCode)
	}
	var login struct {
		UserID       string `json:"user_id"`
		SessionToken string `json:"session_token"`
	}
	_ = json.NewDecoder(loginResp.Body).Decode(&login)
	_ = loginResp.Body.Close()
	if login.UserID != signup.UserID {
		t.Fatalf("login user_id=%s, want %s", login.UserID, signup.UserID)
	}

	// Register a device using the session token.
	regResp := postJSON(t, rig.server.URL+"/auth/register",
		map[string]string{"device_id": "d1", "platform": "browser"},
		signup.SessionToken)
	if regResp.StatusCode != 200 {
		t.Fatalf("register status=%d", regResp.StatusCode)
	}
	var reg pb.RegisterResponse
	_ = json.NewDecoder(regResp.Body).Decode(&reg)
	_ = regResp.Body.Close()
	if reg.Token == "" {
		t.Fatalf("register missing device token")
	}
	if reg.UserID != signup.UserID {
		t.Fatalf("register user_id=%s, want %s", reg.UserID, signup.UserID)
	}

	// The WS validator now requires "records" to be a declared table; seed
	// it directly via the store so the broadcast seq stays 1 (the HTTP
	// path would consume a seq for the table_create broadcast).
	if _, err := rig.st.Schema().CreateTable(context.Background(), signup.UserID, "records"); err != nil {
		t.Fatalf("seed table: %v", err)
	}

	// Open /ws with the device token and send a provider upsert; expect echo.
	u, _ := url.Parse(rig.server.URL)
	u.Scheme = "ws"
	u.Path = "/ws"
	q := u.Query()
	q.Set("token", reg.Token)
	u.RawQuery = q.Encode()
	c, _, err := websocket.DefaultDialer.Dial(u.String(), nil)
	if err != nil {
		t.Fatalf("ws dial: %v", err)
	}
	defer func() { _ = c.Close() }()

	body := fmt.Sprintf(
		`{"device_id":"d1","user_id":%q,"namespace":%q,"role":"provider","op":"upsert","table":"records","payload":{"id":"r1"},"ts":%d}`,
		signup.UserID, signup.UserID, time.Now().UnixMilli(),
	)
	_ = c.SetWriteDeadline(time.Now().Add(time.Second))
	if err := c.WriteMessage(websocket.TextMessage, []byte(body)); err != nil {
		t.Fatalf("ws write: %v", err)
	}
	_ = c.SetReadDeadline(time.Now().Add(2 * time.Second))
	_, data, err := c.ReadMessage()
	if err != nil {
		t.Fatalf("ws read: %v", err)
	}
	var echo pb.Message
	if err := json.Unmarshal(data, &echo); err != nil {
		t.Fatalf("ws decode: %v", err)
	}
	if echo.Seq != 1 {
		t.Fatalf("echo seq = %d, want 1", echo.Seq)
	}
	if echo.Namespace != signup.UserID {
		t.Fatalf("echo namespace = %q, want %q", echo.Namespace, signup.UserID)
	}
}

// Signup twice with the same email → second attempt is 409.
func TestSignupDuplicate(t *testing.T) {
	rig := newAccountRig(t)
	a := postJSON(t, rig.server.URL+"/auth/signup",
		map[string]string{"email": "bob@example.com", "password": "hunter22"}, "")
	if a.StatusCode != 200 {
		t.Fatalf("first signup status=%d", a.StatusCode)
	}
	_ = a.Body.Close()
	b := postJSON(t, rig.server.URL+"/auth/signup",
		map[string]string{"email": "BOB@example.com", "password": "hunter22"}, "")
	if b.StatusCode != 409 {
		t.Fatalf("duplicate signup status=%d, want 409", b.StatusCode)
	}
	_ = b.Body.Close()
}

// Login with wrong password → 401 with no timing oracle.
func TestLoginBadPassword(t *testing.T) {
	rig := newAccountRig(t)
	_ = postJSON(t, rig.server.URL+"/auth/signup",
		map[string]string{"email": "carol@example.com", "password": "hunter22"}, "").Body.Close()
	resp := postJSON(t, rig.server.URL+"/auth/login",
		map[string]string{"email": "carol@example.com", "password": "wrong"}, "")
	if resp.StatusCode != 401 {
		t.Fatalf("login status=%d, want 401", resp.StatusCode)
	}
	_ = resp.Body.Close()
}

// /auth/register without a session token is rejected.
func TestRegisterRequiresSession(t *testing.T) {
	rig := newAccountRig(t)
	resp := postJSON(t, rig.server.URL+"/auth/register",
		map[string]string{"device_id": "d2", "platform": "browser"}, "")
	if resp.StatusCode != 401 {
		t.Fatalf("register w/o session status=%d, want 401", resp.StatusCode)
	}
	_ = resp.Body.Close()
}

// /auth/register with a DEVICE token (wrong flavor) is rejected — session only.
func TestRegisterRejectsDeviceToken(t *testing.T) {
	rig := newAccountRig(t)
	deviceTok, err := rig.signer.IssueDevice("u1", "d1", "browser")
	if err != nil {
		t.Fatalf("issue device: %v", err)
	}
	resp := postJSON(t, rig.server.URL+"/auth/register",
		map[string]string{"device_id": "d3", "platform": "browser"}, deviceTok)
	if resp.StatusCode != 401 {
		t.Fatalf("device-token register status=%d, want 401", resp.StatusCode)
	}
	_ = resp.Body.Close()
}

// Two devices for the same user land in the same namespace; a third user's
// messages do NOT leak across.
func TestMultiDeviceNamespaceIsolation(t *testing.T) {
	rig := newAccountRig(t)

	// User A signs up, registers two devices.
	sa := postJSON(t, rig.server.URL+"/auth/signup",
		map[string]string{"email": "a@x.com", "password": "hunter22"}, "")
	var aResp struct {
		UserID       string `json:"user_id"`
		SessionToken string `json:"session_token"`
	}
	_ = json.NewDecoder(sa.Body).Decode(&aResp)
	_ = sa.Body.Close()
	// Normalize field names (go doesn't see the json tag on anonymous struct this way).
	// Use a second decode pass.
	sa2, _ := http.NewRequest("GET", rig.server.URL+"/auth/me", nil)
	sa2.Header.Set("Authorization", "Bearer "+aResp.SessionToken)
	_ = sa2
	// Simpler: just call /auth/me and parse properly.
	var meA struct {
		UserID string `json:"user_id"`
	}
	{
		r, _ := http.NewRequest("GET", rig.server.URL+"/auth/me", nil)
		r.Header.Set("Authorization", "Bearer "+aResp.SessionToken)
		resp, _ := http.DefaultClient.Do(r)
		_ = json.NewDecoder(resp.Body).Decode(&meA)
		_ = resp.Body.Close()
	}

	var regA1, regA2 pb.RegisterResponse
	{
		r := postJSON(t, rig.server.URL+"/auth/register",
			map[string]string{"device_id": "dA1", "platform": "browser"}, aResp.SessionToken)
		_ = json.NewDecoder(r.Body).Decode(&regA1)
		_ = r.Body.Close()
	}
	{
		r := postJSON(t, rig.server.URL+"/auth/register",
			map[string]string{"device_id": "dA2", "platform": "desktop"}, aResp.SessionToken)
		_ = json.NewDecoder(r.Body).Decode(&regA2)
		_ = r.Body.Close()
	}
	if regA1.UserID != regA2.UserID || regA1.UserID != meA.UserID {
		t.Fatalf("same user's devices must share user_id; got %+v %+v", regA1, regA2)
	}

	// User B signs up and registers a device.
	sb := postJSON(t, rig.server.URL+"/auth/signup",
		map[string]string{"email": "b@x.com", "password": "hunter22"}, "")
	var bResp struct {
		UserID       string `json:"user_id"`
		SessionToken string `json:"session_token"`
	}
	_ = json.NewDecoder(sb.Body).Decode(&bResp)
	_ = sb.Body.Close()
	var regB pb.RegisterResponse
	{
		r := postJSON(t, rig.server.URL+"/auth/register",
			map[string]string{"device_id": "dB1", "platform": "browser"}, bResp.SessionToken)
		_ = json.NewDecoder(r.Body).Decode(&regB)
		_ = r.Body.Close()
	}

	// Seed the "records" table for user A so the upsert below passes
	// schema validation without consuming a seq on a table_create
	// broadcast.
	if _, err := rig.st.Schema().CreateTable(context.Background(), meA.UserID, "records"); err != nil {
		t.Fatalf("seed table: %v", err)
	}

	// User A's two devices + User B's one device all dial /ws.
	dial := func(token string) *websocket.Conn {
		u, _ := url.Parse(rig.server.URL)
		u.Scheme = "ws"
		u.Path = "/ws"
		q := u.Query()
		q.Set("token", token)
		u.RawQuery = q.Encode()
		c, _, err := websocket.DefaultDialer.Dial(u.String(), nil)
		if err != nil {
			t.Fatalf("ws dial: %v", err)
		}
		return c
	}
	a1 := dial(regA1.Token)
	a2 := dial(regA2.Token)
	b1 := dial(regB.Token)
	defer func() { _ = a1.Close(); _ = a2.Close(); _ = b1.Close() }()

	// A1 publishes. A1 + A2 both receive. B1 does NOT.
	body := fmt.Sprintf(
		`{"namespace":%q,"role":"provider","op":"upsert","table":"records","payload":{"id":"shared"},"ts":%d}`,
		regA1.UserID, time.Now().UnixMilli(),
	)
	if err := a1.WriteMessage(websocket.TextMessage, []byte(body)); err != nil {
		t.Fatalf("a1 write: %v", err)
	}

	readOne := func(c *websocket.Conn, d time.Duration) (pb.Message, bool) {
		_ = c.SetReadDeadline(time.Now().Add(d))
		_, data, err := c.ReadMessage()
		if err != nil {
			return pb.Message{}, false
		}
		var m pb.Message
		_ = json.Unmarshal(data, &m)
		return m, true
	}
	if m, ok := readOne(a1, 2*time.Second); !ok || m.Seq != 1 {
		t.Fatalf("a1 didn't receive its own broadcast: %+v ok=%v", m, ok)
	}
	if m, ok := readOne(a2, 2*time.Second); !ok || m.Seq != 1 {
		t.Fatalf("a2 didn't receive A's broadcast: %+v ok=%v", m, ok)
	}
	if m, ok := readOne(b1, 300*time.Millisecond); ok {
		t.Fatalf("b1 received cross-namespace leak: %+v", m)
	}
}
