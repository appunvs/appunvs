package handler_test

import (
	"bufio"
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"

	"github.com/appunvs/appunvs/relay/internal/ai"
	"github.com/appunvs/appunvs/relay/internal/artifact"
	"github.com/appunvs/appunvs/relay/internal/auth"
	"github.com/appunvs/appunvs/relay/internal/box"
	"github.com/appunvs/appunvs/relay/internal/handler"
	"github.com/appunvs/appunvs/relay/internal/pb"
	"github.com/appunvs/appunvs/relay/internal/sandbox"
	"github.com/appunvs/appunvs/relay/internal/store"
	"github.com/appunvs/appunvs/relay/internal/workspace"
)

// TestAITurnSSEWithStub boots a tiny server wired to the echo engine and
// asserts the wire format of POST /ai/turn: correct SSE framing
// (`event: token` + `data: {...}`), finished frame terminates the stream,
// and the call requires a device token.
func TestAITurnSSEWithStub(t *testing.T) {
	gin.SetMode(gin.TestMode)
	ctx := context.Background()

	tmp := t.TempDir()
	st, err := store.Open(ctx, tmp+"/relay.db")
	if err != nil {
		t.Fatalf("store: %v", err)
	}
	defer func() { _ = st.Close() }()

	if _, err := st.DB.ExecContext(ctx,
		`INSERT INTO users(id, email, password_hash, created_at) VALUES(?,?,?,?)`,
		"u_test", "t@example.com", "x", int64(1)); err != nil {
		t.Fatalf("seed user: %v", err)
	}

	ws, _ := workspace.NewStore(workspace.Config{Root: tmp + "/ws"})
	art, _ := artifact.NewLocalFS(tmp+"/art", "http://localhost:8080/_artifacts")
	boxSvc := box.New(st.Boxes(), sandbox.NewLocalStub(), art, ws)

	boxObj, err := boxSvc.Create(ctx, "u_test", "dev_a", "demo", pb.RuntimeKindRNBundle)
	if err != nil {
		t.Fatalf("create box: %v", err)
	}

	log := zap.NewNop()
	signer, err := auth.NewSigner("", "", "appunvs-test", "appunvs-test", time.Hour, time.Hour, log)
	if err != nil {
		t.Fatalf("signer: %v", err)
	}
	token, err := signer.IssueDevice("u_test", "dev_a", "browser")
	if err != nil {
		t.Fatalf("token: %v", err)
	}

	r := gin.New()
	handler.RegisterAIRoutes(r, handler.AIDeps{
		Signer: signer,
		Engine: ai.NewStub(),
		Box:    boxSvc,
		Log:    log,
	})
	srv := httptest.NewServer(r)
	defer srv.Close()

	// --- 1. auth gate
	rsp, err := http.Post(srv.URL+"/ai/turn", "application/json",
		strings.NewReader(`{"box_id":"`+boxObj.ID+`","text":"hi"}`))
	if err != nil {
		t.Fatalf("unauth post: %v", err)
	}
	_ = rsp.Body.Close()
	if rsp.StatusCode != http.StatusUnauthorized {
		t.Fatalf("no-auth status = %d, want 401", rsp.StatusCode)
	}

	// --- 2. happy path: SSE frames round-trip end-to-end
	req, _ := http.NewRequest(http.MethodPost, srv.URL+"/ai/turn",
		strings.NewReader(`{"box_id":"`+boxObj.ID+`","text":"hello"}`))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)
	rsp, err = http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("do: %v", err)
	}
	defer func() { _ = rsp.Body.Close() }()

	if got := rsp.Header.Get("Content-Type"); !strings.HasPrefix(got, "text/event-stream") {
		t.Fatalf("content-type = %q, want text/event-stream*", got)
	}

	var (
		sawToken    bool
		sawFinished bool
		lastEvent   string
	)
	scanner := bufio.NewScanner(rsp.Body)
	scanner.Buffer(make([]byte, 64*1024), 1024*1024)
	for scanner.Scan() {
		line := scanner.Text()
		switch {
		case strings.HasPrefix(line, "event: "):
			lastEvent = strings.TrimPrefix(line, "event: ")
		case strings.HasPrefix(line, "data: "):
			switch lastEvent {
			case "token":
				sawToken = true
			case "finished":
				sawFinished = true
			}
		}
		if sawFinished {
			break
		}
	}
	if err := scanner.Err(); err != nil {
		t.Fatalf("scan: %v", err)
	}
	if !sawToken {
		t.Fatalf("no token event observed")
	}
	if !sawFinished {
		t.Fatalf("no finished event observed")
	}

	// --- 3. cross-namespace box access must 404
	otherToken, _ := signer.IssueDevice("u_other", "dev_x", "browser")
	req2, _ := http.NewRequest(http.MethodPost, srv.URL+"/ai/turn",
		strings.NewReader(`{"box_id":"`+boxObj.ID+`","text":"hi"}`))
	req2.Header.Set("Content-Type", "application/json")
	req2.Header.Set("Authorization", "Bearer "+otherToken)
	rsp2, err := http.DefaultClient.Do(req2)
	if err != nil {
		t.Fatalf("cross-ns do: %v", err)
	}
	_ = rsp2.Body.Close()
	if rsp2.StatusCode != http.StatusNotFound {
		t.Fatalf("cross-ns status = %d, want 404", rsp2.StatusCode)
	}
}
