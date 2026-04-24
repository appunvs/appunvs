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
	"github.com/appunvs/appunvs/relay/internal/billing"
	"github.com/appunvs/appunvs/relay/internal/handler"
	"github.com/appunvs/appunvs/relay/internal/hub"
	"github.com/appunvs/appunvs/relay/internal/pb"
	"github.com/appunvs/appunvs/relay/internal/sequencer"
	"github.com/appunvs/appunvs/relay/internal/store"
	"github.com/appunvs/appunvs/relay/internal/stream"
)

// billingRig is a superset of accountTestRig that also wires the billing
// HTTP routes and the quota gate in front of /ws. We keep it local to this
// file so the account tests don't inherit billing coupling.
type billingRig struct {
	server  *httptest.Server
	signer  *auth.Signer
	rdb     *redis.Client
	st      *store.Store
	billing *billing.Service
}

func newBillingRig(t *testing.T, opts ...billingOpt) *billingRig {
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

	dbPath := filepath.Join(t.TempDir(), "billing_test.db")
	st, err := store.Open(context.Background(), dbPath)
	if err != nil {
		t.Fatalf("store open: %v", err)
	}

	cfg := billingRigConfig{
		stripeSecret: "",
		webhookSig:   "",
	}
	for _, o := range opts {
		o(&cfg)
	}

	log := zap.NewNop()
	signer, err := auth.NewSigner("", "", "appunvs-test", "appunvs-test", time.Hour, time.Hour, log)
	if err != nil {
		t.Fatalf("signer: %v", err)
	}
	h := hub.New(log)
	seq := sequencer.New(rdb)
	streamSvc := stream.New(rdb, 1000)
	billingSvc := billing.New(st, log, cfg.stripeSecret, cfg.webhookSig,
		"https://test/success", "https://test/cancel")
	accountDeps := auth.Deps{Signer: signer, Store: st, Log: log}
	billingDeps := handler.BillingDeps{Signer: signer, Billing: billingSvc, Log: log}

	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.POST("/auth/signup", auth.Signup(accountDeps))
	r.POST("/auth/login", auth.Login(accountDeps))
	r.POST("/auth/register", auth.Register(accountDeps))
	r.GET("/auth/me", auth.Me(accountDeps))
	r.GET("/billing/plans", handler.BillingPlans(billingDeps))
	r.GET("/billing/status", handler.BillingStatus(billingDeps))
	r.POST("/billing/checkout", handler.BillingCheckout(billingDeps))
	r.POST("/billing/webhook", handler.BillingWebhook(billingDeps))
	r.GET("/ws", handler.WS(handler.Deps{
		Signer: signer,
		Hub:    h,
		Seq:    seq,
		Stream: streamSvc,
		Store:  st,
		Quota:  billing.NewGate(st, log),
		Log:    log,
	}))
	ts := httptest.NewServer(r)
	t.Cleanup(func() {
		ts.Close()
		_ = rdb.Close()
		_ = st.Close()
	})
	return &billingRig{server: ts, signer: signer, rdb: rdb, st: st, billing: billingSvc}
}

type billingRigConfig struct {
	stripeSecret string
	webhookSig   string
}

type billingOpt func(*billingRigConfig)

func withStripeSecrets(secret, webhookSig string) billingOpt {
	return func(c *billingRigConfig) {
		c.stripeSecret = secret
		c.webhookSig = webhookSig
	}
}

// rigSignup signs up a test user and returns (userID, sessionToken).
func rigSignup(t *testing.T, base, email string) (string, string) {
	t.Helper()
	resp := postJSON(t, base+"/auth/signup",
		map[string]string{"email": email, "password": "hunter22"}, "")
	if resp.StatusCode != 200 {
		t.Fatalf("signup status=%d", resp.StatusCode)
	}
	var out struct {
		UserID       string `json:"user_id"`
		SessionToken string `json:"session_token"`
	}
	_ = json.NewDecoder(resp.Body).Decode(&out)
	_ = resp.Body.Close()
	return out.UserID, out.SessionToken
}

func getJSON(t *testing.T, u, auth string, target any) int {
	t.Helper()
	req, _ := http.NewRequest("GET", u, nil)
	if auth != "" {
		req.Header.Set("Authorization", "Bearer "+auth)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("GET %s: %v", u, err)
	}
	defer func() { _ = resp.Body.Close() }()
	if target != nil {
		_ = json.NewDecoder(resp.Body).Decode(target)
	}
	return resp.StatusCode
}

// TestSignupAutoSubscribesToFree: a fresh user's /billing/status returns
// plan=free with the seeded limits.
func TestSignupAutoSubscribesToFree(t *testing.T) {
	rig := newBillingRig(t)
	_, sess := rigSignup(t, rig.server.URL, "auto@x.com")

	var st struct {
		Plan         string     `json:"plan"`
		Status       string     `json:"status"`
		MessagesUsed int64      `json:"messages_used"`
		Limits       store.Plan `json:"limits"`
	}
	code := getJSON(t, rig.server.URL+"/billing/status", sess, &st)
	if code != 200 {
		t.Fatalf("status code = %d", code)
	}
	if st.Plan != "free" {
		t.Fatalf("plan=%q, want free", st.Plan)
	}
	if st.Status != "active" {
		t.Fatalf("status=%q, want active", st.Status)
	}
	if st.Limits.ID != "free" || st.Limits.MessagesPerDay != 1_000 {
		t.Fatalf("limits=%+v, want free/1000", st.Limits)
	}
	if st.MessagesUsed != 0 {
		t.Fatalf("messages_used=%d, want 0", st.MessagesUsed)
	}
}

// TestBillingPlansPublic: /billing/plans returns 3 plans and requires no auth.
func TestBillingPlansPublic(t *testing.T) {
	rig := newBillingRig(t)
	var body struct {
		Plans []store.Plan `json:"plans"`
	}
	code := getJSON(t, rig.server.URL+"/billing/plans", "", &body)
	if code != 200 {
		t.Fatalf("plans code=%d", code)
	}
	if len(body.Plans) != 4 {
		t.Fatalf("plans count=%d, want 4 (free/pro/max/team)", len(body.Plans))
	}
}

// TestQuotaBlocksAfterLimit: shrink the free-plan quota to 2 via a direct
// DB write, publish 3 provider messages, and assert the 3rd is refused
// with an op=quota_exceeded echo instead of a normal fanout.
func TestQuotaBlocksAfterLimit(t *testing.T) {
	rig := newBillingRig(t)
	userID, sess := rigSignup(t, rig.server.URL, "quota@x.com")

	// Direct-DB surgery: clamp the free plan's messages_per_day to 2 for
	// this test's DB only.
	_, err := rig.st.DB.ExecContext(context.Background(),
		`UPDATE plans SET messages_per_day = 2 WHERE id = 'free'`)
	if err != nil {
		t.Fatalf("clamp free: %v", err)
	}

	// Register a device so we can dial /ws.
	regResp := postJSON(t, rig.server.URL+"/auth/register",
		map[string]string{"device_id": "dq", "platform": "browser"}, sess)
	if regResp.StatusCode != 200 {
		t.Fatalf("register status=%d", regResp.StatusCode)
	}
	var reg pb.RegisterResponse
	_ = json.NewDecoder(regResp.Body).Decode(&reg)
	_ = regResp.Body.Close()

	// Dial /ws with the device token.
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

	send := func(i int) {
		body := fmt.Sprintf(
			`{"namespace":%q,"role":"provider","op":"upsert","table":"records","payload":{"id":"r%d"},"ts":%d}`,
			userID, i, time.Now().UnixMilli(),
		)
		_ = c.SetWriteDeadline(time.Now().Add(time.Second))
		if err := c.WriteMessage(websocket.TextMessage, []byte(body)); err != nil {
			t.Fatalf("write %d: %v", i, err)
		}
	}

	readOne := func() pb.Message {
		_ = c.SetReadDeadline(time.Now().Add(2 * time.Second))
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

	// Create a minimal schema so the validator lets the upserts through.
	if _, err := rig.st.Schema().CreateTable(context.Background(), userID, "records"); err != nil {
		t.Fatalf("create table: %v", err)
	}

	// First two: normal seq=1 and seq=2.
	send(1)
	got := readOne()
	if got.Op != pb.OpUpsert || got.Seq != 1 {
		t.Fatalf("msg1: op=%s seq=%d, want upsert seq=1", got.Op.String(), got.Seq)
	}
	send(2)
	got = readOne()
	if got.Op != pb.OpUpsert || got.Seq != 2 {
		t.Fatalf("msg2: op=%s seq=%d, want upsert seq=2", got.Op.String(), got.Seq)
	}
	// Third: quota exceeded.
	send(3)
	got = readOne()
	if got.Op != pb.OpQuotaExceeded {
		t.Fatalf("msg3 op=%s seq=%d, want quota_exceeded", got.Op.String(), got.Seq)
	}
	if got.Role != pb.RoleProvider {
		t.Fatalf("msg3 role=%s, want provider", got.Role.String())
	}
}

// TestMockCheckoutUpgradesUser: POST /billing/checkout returns a mock URL,
// POST /billing/webhook with X-Mock-Upgrade: pro flips the user's plan,
// and GET /billing/status reflects it.
func TestMockCheckoutUpgradesUser(t *testing.T) {
	rig := newBillingRig(t)
	userID, sess := rigSignup(t, rig.server.URL, "mock@x.com")

	// Checkout request (mock mode).
	resp := postJSON(t, rig.server.URL+"/billing/checkout",
		map[string]string{"plan_id": "pro"}, sess)
	if resp.StatusCode != 200 {
		t.Fatalf("checkout status=%d", resp.StatusCode)
	}
	var co struct {
		URL  string `json:"url"`
		Mode string `json:"mode"`
	}
	_ = json.NewDecoder(resp.Body).Decode(&co)
	_ = resp.Body.Close()
	if co.Mode != "mock" {
		t.Fatalf("mode=%q, want mock", co.Mode)
	}
	if co.URL == "" || co.URL[:20] != "https://stripe.mock/" {
		t.Fatalf("url=%q", co.URL)
	}

	// Simulate Stripe firing the webhook in mock mode.
	req, _ := http.NewRequest("POST", rig.server.URL+"/billing/webhook", bytes.NewReader([]byte("{}")))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Mock-Upgrade", "pro")
	req.Header.Set("X-Mock-User", userID)
	wresp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("webhook: %v", err)
	}
	if wresp.StatusCode != 200 {
		t.Fatalf("webhook status=%d", wresp.StatusCode)
	}
	_ = wresp.Body.Close()

	// Status should now show pro.
	var st struct {
		Plan     string     `json:"plan"`
		PlanName string     `json:"plan_name"`
		Limits   store.Plan `json:"limits"`
	}
	code := getJSON(t, rig.server.URL+"/billing/status", sess, &st)
	if code != 200 {
		t.Fatalf("status code=%d", code)
	}
	if st.Plan != "pro" {
		t.Fatalf("plan=%q, want pro", st.Plan)
	}
	if st.Limits.MessagesPerDay != 100_000 {
		t.Fatalf("limits=%+v, want pro limits", st.Limits)
	}
}

// TestMockCheckoutBadPlan: unknown plan ids are rejected.
func TestMockCheckoutBadPlan(t *testing.T) {
	rig := newBillingRig(t)
	_, sess := rigSignup(t, rig.server.URL, "badplan@x.com")
	resp := postJSON(t, rig.server.URL+"/billing/checkout",
		map[string]string{"plan_id": "enterprise"}, sess)
	if resp.StatusCode != 400 {
		t.Fatalf("bad plan status=%d, want 400", resp.StatusCode)
	}
	_ = resp.Body.Close()
}

// TestMockWebhookRequiresHeader: mock mode rejects webhook POSTs that
// don't carry the X-Mock-Upgrade + X-Mock-User shortcut headers.
func TestMockWebhookRequiresHeader(t *testing.T) {
	rig := newBillingRig(t)
	resp, err := http.Post(rig.server.URL+"/billing/webhook", "application/json", bytes.NewReader([]byte("{}")))
	if err != nil {
		t.Fatalf("post: %v", err)
	}
	if resp.StatusCode != 400 {
		t.Fatalf("bare webhook status=%d, want 400", resp.StatusCode)
	}
	_ = resp.Body.Close()
}

// TestRealWebhookSignatureRejectsBadSig: when real Stripe creds are set,
// the webhook refuses payloads with a missing/bad Stripe-Signature.
func TestRealWebhookSignatureRejectsBadSig(t *testing.T) {
	rig := newBillingRig(t, withStripeSecrets("sk_test_fake", "whsec_fake"))

	// No Stripe-Signature: rejected.
	resp, err := http.Post(rig.server.URL+"/billing/webhook", "application/json", bytes.NewReader([]byte(`{"type":"x"}`)))
	if err != nil {
		t.Fatalf("post: %v", err)
	}
	if resp.StatusCode != 400 {
		t.Fatalf("unsigned status=%d, want 400", resp.StatusCode)
	}
	_ = resp.Body.Close()

	// Bogus signature: rejected.
	req, _ := http.NewRequest("POST", rig.server.URL+"/billing/webhook", bytes.NewReader([]byte(`{"type":"x"}`)))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Stripe-Signature", "t=1,v1=deadbeef")
	resp, err = http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("post: %v", err)
	}
	if resp.StatusCode != 400 {
		t.Fatalf("bad sig status=%d, want 400", resp.StatusCode)
	}
	_ = resp.Body.Close()

	// Real-mode rig must report "real" via Service.Mode().
	if rig.billing.Mode() != "real" {
		t.Fatalf("mode=%q, want real", rig.billing.Mode())
	}
}
