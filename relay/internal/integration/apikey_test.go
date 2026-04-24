package integration_test

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"

	"github.com/appunvs/appunvs/relay/internal/apikey"
	"github.com/appunvs/appunvs/relay/internal/auth"
	"github.com/appunvs/appunvs/relay/internal/handler"
	"github.com/appunvs/appunvs/relay/internal/store"
)

// apiKeyRig is a trimmed version of newAccountRig that mounts the account +
// /keys surface plus one protected endpoint guarded by apikey.Authenticate.
// It deliberately avoids Redis and WebSockets — the api-key surface is pure
// HTTP + SQLite.
type apiKeyRig struct {
	server *httptest.Server
	signer *auth.Signer
	st     *store.Store
}

func newAPIKeyRig(t *testing.T) *apiKeyRig {
	t.Helper()
	dbPath := filepath.Join(t.TempDir(), "apikey.db")
	st, err := store.Open(context.Background(), dbPath)
	if err != nil {
		t.Fatalf("store open: %v", err)
	}
	log := zap.NewNop()
	signer, err := auth.NewSigner("", "", "appunvs-test", "appunvs-test", time.Hour, time.Hour, log)
	if err != nil {
		t.Fatalf("signer: %v", err)
	}

	accountDeps := auth.Deps{Signer: signer, Store: st, Log: log}
	keyDeps := handler.APIKeyDeps{Signer: signer, Store: st, Log: log}

	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.POST("/auth/signup", auth.Signup(accountDeps))
	r.POST("/auth/login", auth.Login(accountDeps))
	r.GET("/auth/me", auth.Me(accountDeps))
	handler.APIKeyRoutes(r, keyDeps)

	// A protected endpoint that accepts either auth flavor — mirror
	// /auth/me's "who am I?" shape so we can assert on user_id.
	r.GET("/whoami", apikey.Authenticate(signer, st.APIKeys(), log), func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"user_id":   c.GetString(apikey.CtxUserID),
			"auth_kind": c.GetString(apikey.CtxAuthKind),
		})
	})

	ts := httptest.NewServer(r)
	t.Cleanup(func() {
		ts.Close()
		_ = st.Close()
	})
	return &apiKeyRig{server: ts, signer: signer, st: st}
}

type signupOut struct {
	UserID       string `json:"user_id"`
	SessionToken string `json:"session_token"`
}

func signupUser(t *testing.T, rig *apiKeyRig, email string) signupOut {
	t.Helper()
	body, _ := json.Marshal(map[string]string{"email": email, "password": "hunter22"})
	resp, err := http.Post(rig.server.URL+"/auth/signup", "application/json", bytes.NewReader(body))
	if err != nil {
		t.Fatalf("signup: %v", err)
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != 200 {
		raw, _ := io.ReadAll(resp.Body)
		t.Fatalf("signup status=%d body=%s", resp.StatusCode, raw)
	}
	var out signupOut
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		t.Fatalf("decode signup: %v", err)
	}
	return out
}

// jsonRequest issues a JSON request with an optional bearer.
func jsonRequest(t *testing.T, method, url, bearer string, body any) *http.Response {
	t.Helper()
	var reader io.Reader
	if body != nil {
		b, _ := json.Marshal(body)
		reader = bytes.NewReader(b)
	}
	req, err := http.NewRequest(method, url, reader)
	if err != nil {
		t.Fatalf("new request: %v", err)
	}
	if body != nil {
		req.Header.Set("content-type", "application/json")
	}
	if bearer != "" {
		req.Header.Set("Authorization", "Bearer "+bearer)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("%s %s: %v", method, url, err)
	}
	return resp
}

// TestCreateListRevoke walks the /keys surface from end to end.
func TestCreateListRevoke(t *testing.T) {
	rig := newAPIKeyRig(t)
	user := signupUser(t, rig, "create-list@example.com")

	// Create.
	resp := jsonRequest(t, "POST", rig.server.URL+"/keys", user.SessionToken, map[string]string{"name": "cli-1"})
	if resp.StatusCode != 200 {
		raw, _ := io.ReadAll(resp.Body)
		_ = resp.Body.Close()
		t.Fatalf("create status=%d body=%s", resp.StatusCode, raw)
	}
	var created struct {
		ID     string `json:"id"`
		Name   string `json:"name"`
		Prefix string `json:"prefix"`
		Secret string `json:"secret"`
	}
	_ = json.NewDecoder(resp.Body).Decode(&created)
	_ = resp.Body.Close()
	if created.ID == "" || created.Secret == "" || created.Prefix == "" {
		t.Fatalf("create response missing fields: %+v", created)
	}
	if created.Name != "cli-1" {
		t.Fatalf("create name = %q, want cli-1", created.Name)
	}

	// List.
	resp = jsonRequest(t, "GET", rig.server.URL+"/keys", user.SessionToken, nil)
	if resp.StatusCode != 200 {
		t.Fatalf("list status=%d", resp.StatusCode)
	}
	var list []struct {
		ID        string `json:"id"`
		Prefix    string `json:"prefix"`
		Secret    string `json:"secret"`
		RevokedAt *int64 `json:"revoked_at"`
	}
	_ = json.NewDecoder(resp.Body).Decode(&list)
	_ = resp.Body.Close()
	if len(list) != 1 || list[0].ID != created.ID {
		t.Fatalf("list rows = %+v, want 1 matching create", list)
	}
	if list[0].Secret != "" {
		t.Fatalf("list must not leak secret, got %q", list[0].Secret)
	}
	if list[0].Prefix != created.Prefix {
		t.Fatalf("list prefix mismatch: %q vs %q", list[0].Prefix, created.Prefix)
	}

	// Revoke.
	resp = jsonRequest(t, "DELETE", rig.server.URL+"/keys/"+created.ID, user.SessionToken, nil)
	if resp.StatusCode != 204 {
		t.Fatalf("revoke status=%d", resp.StatusCode)
	}
	_ = resp.Body.Close()

	// Revoke again → 404.
	resp = jsonRequest(t, "DELETE", rig.server.URL+"/keys/"+created.ID, user.SessionToken, nil)
	if resp.StatusCode != 404 {
		t.Fatalf("double revoke status=%d, want 404", resp.StatusCode)
	}
	_ = resp.Body.Close()
}

// TestAuthenticateViaKey hits /whoami with (a) a session JWT and (b) an API
// key; both must be accepted and report the correct auth_kind.
func TestAuthenticateViaKey(t *testing.T) {
	rig := newAPIKeyRig(t)
	user := signupUser(t, rig, "via-key@example.com")

	// Mint an API key for that user.
	resp := jsonRequest(t, "POST", rig.server.URL+"/keys", user.SessionToken, map[string]string{"name": "automation"})
	if resp.StatusCode != 200 {
		t.Fatalf("create status=%d", resp.StatusCode)
	}
	var created struct {
		Secret string `json:"secret"`
	}
	_ = json.NewDecoder(resp.Body).Decode(&created)
	_ = resp.Body.Close()

	type who struct {
		UserID   string `json:"user_id"`
		AuthKind string `json:"auth_kind"`
	}

	// Via session JWT.
	r1 := jsonRequest(t, "GET", rig.server.URL+"/whoami", user.SessionToken, nil)
	if r1.StatusCode != 200 {
		t.Fatalf("session whoami status=%d", r1.StatusCode)
	}
	var got1 who
	_ = json.NewDecoder(r1.Body).Decode(&got1)
	_ = r1.Body.Close()
	if got1.UserID != user.UserID || got1.AuthKind != "session" {
		t.Fatalf("session whoami = %+v, want user=%s kind=session", got1, user.UserID)
	}

	// Via API key.
	r2 := jsonRequest(t, "GET", rig.server.URL+"/whoami", created.Secret, nil)
	if r2.StatusCode != 200 {
		t.Fatalf("apikey whoami status=%d", r2.StatusCode)
	}
	var got2 who
	_ = json.NewDecoder(r2.Body).Decode(&got2)
	_ = r2.Body.Close()
	if got2.UserID != user.UserID || got2.AuthKind != "apikey" {
		t.Fatalf("apikey whoami = %+v, want user=%s kind=apikey", got2, user.UserID)
	}

	// Device tokens must be rejected on the shared endpoint — they are
	// the wrong flavor for user-level HTTP ops.
	deviceTok, err := rig.signer.IssueDevice(user.UserID, "d1", "browser")
	if err != nil {
		t.Fatalf("issue device: %v", err)
	}
	r3 := jsonRequest(t, "GET", rig.server.URL+"/whoami", deviceTok, nil)
	if r3.StatusCode != 401 {
		t.Fatalf("device token whoami status=%d, want 401", r3.StatusCode)
	}
	_ = r3.Body.Close()
}

// TestRevokedKeyRejected — once revoked, the secret no longer authenticates.
func TestRevokedKeyRejected(t *testing.T) {
	rig := newAPIKeyRig(t)
	user := signupUser(t, rig, "revoked@example.com")

	resp := jsonRequest(t, "POST", rig.server.URL+"/keys", user.SessionToken, map[string]string{"name": "short-lived"})
	var created struct {
		ID     string `json:"id"`
		Secret string `json:"secret"`
	}
	_ = json.NewDecoder(resp.Body).Decode(&created)
	_ = resp.Body.Close()

	// Authenticate once successfully to confirm the happy path.
	ok := jsonRequest(t, "GET", rig.server.URL+"/whoami", created.Secret, nil)
	if ok.StatusCode != 200 {
		t.Fatalf("pre-revoke whoami status=%d", ok.StatusCode)
	}
	_ = ok.Body.Close()

	// Revoke.
	rv := jsonRequest(t, "DELETE", rig.server.URL+"/keys/"+created.ID, user.SessionToken, nil)
	if rv.StatusCode != 204 {
		t.Fatalf("revoke status=%d", rv.StatusCode)
	}
	_ = rv.Body.Close()

	// Authentication with the revoked key is now 401.
	bad := jsonRequest(t, "GET", rig.server.URL+"/whoami", created.Secret, nil)
	if bad.StatusCode != 401 {
		t.Fatalf("post-revoke whoami status=%d, want 401", bad.StatusCode)
	}
	_ = bad.Body.Close()
}

// TestWrongSecretRejected — a well-formed but wrong secret gives 401 the
// same way an unknown prefix does; the error shape must not distinguish
// "unknown prefix" from "known prefix but bad secret".
func TestWrongSecretRejected(t *testing.T) {
	rig := newAPIKeyRig(t)
	user := signupUser(t, rig, "wrong-secret@example.com")

	// Create a key so the table has a row to look up.
	resp := jsonRequest(t, "POST", rig.server.URL+"/keys", user.SessionToken, map[string]string{"name": "k"})
	var created struct {
		Prefix string `json:"prefix"`
		Secret string `json:"secret"`
	}
	_ = json.NewDecoder(resp.Body).Decode(&created)
	_ = resp.Body.Close()

	// Swap the last char of the secret — same prefix, bad hash.
	tampered := created.Secret[:len(created.Secret)-1] + flipChar(created.Secret[len(created.Secret)-1])
	if tampered == created.Secret {
		t.Fatalf("flipChar is a no-op for %q", created.Secret)
	}
	r1 := jsonRequest(t, "GET", rig.server.URL+"/whoami", tampered, nil)
	if r1.StatusCode != 401 {
		t.Fatalf("tampered whoami status=%d, want 401", r1.StatusCode)
	}
	_ = r1.Body.Close()

	// An entirely unknown but well-formed key shape gives the same 401.
	unknown := "apvs_" + "aaaaaaaaaaaaaaaaaaaaaa"
	r2 := jsonRequest(t, "GET", rig.server.URL+"/whoami", unknown, nil)
	if r2.StatusCode != 401 {
		t.Fatalf("unknown whoami status=%d, want 401", r2.StatusCode)
	}
	_ = r2.Body.Close()

	// And the real secret still works — we haven't broken anything.
	r3 := jsonRequest(t, "GET", rig.server.URL+"/whoami", created.Secret, nil)
	if r3.StatusCode != 200 {
		t.Fatalf("real whoami status=%d, want 200", r3.StatusCode)
	}
	_ = r3.Body.Close()
}

// flipChar returns a different base64-url-safe character. Used so the
// tampered secret retains a valid shape (same length, same namespace).
func flipChar(c byte) string {
	if c == 'a' {
		return "b"
	}
	return "a"
}

// /keys must reject device tokens and API keys (session-only surface).
func TestCreateKeyRejectsNonSession(t *testing.T) {
	rig := newAPIKeyRig(t)
	user := signupUser(t, rig, "gatekeep@example.com")

	// Mint an api key so we can try to use it against /keys.
	resp := jsonRequest(t, "POST", rig.server.URL+"/keys", user.SessionToken, map[string]string{"name": "bootstrap"})
	var created struct {
		Secret string `json:"secret"`
	}
	_ = json.NewDecoder(resp.Body).Decode(&created)
	_ = resp.Body.Close()

	// POST /keys with an API key is forbidden — must be a session JWT.
	r1 := jsonRequest(t, "POST", rig.server.URL+"/keys", created.Secret, map[string]string{"name": "child"})
	if r1.StatusCode != 401 {
		t.Fatalf("apikey->POST /keys status=%d, want 401", r1.StatusCode)
	}
	_ = r1.Body.Close()

	// Device token also rejected.
	deviceTok, _ := rig.signer.IssueDevice(user.UserID, "d1", "browser")
	r2 := jsonRequest(t, "POST", rig.server.URL+"/keys", deviceTok, map[string]string{"name": "device"})
	if r2.StatusCode != 401 {
		t.Fatalf("device->POST /keys status=%d, want 401", r2.StatusCode)
	}
	_ = r2.Body.Close()
}
