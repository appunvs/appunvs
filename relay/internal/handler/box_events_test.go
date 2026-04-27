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

	"github.com/appunvs/appunvs/relay/internal/auth"
	"github.com/appunvs/appunvs/relay/internal/box"
	"github.com/appunvs/appunvs/relay/internal/handler"
)

// TestBoxEventsSSEFanout boots a server wired to /box/events, opens an
// authenticated stream, then publishes a bundle_ready event and asserts
// the SSE record arrives in the canonical event:/data: framing.  Also
// asserts the handler 401s without a device token.
func TestBoxEventsSSEFanout(t *testing.T) {
	gin.SetMode(gin.TestMode)

	log := zap.NewNop()
	signer, err := auth.NewSigner("", "", "appunvs-test", "appunvs-test", time.Hour, time.Hour, log)
	if err != nil {
		t.Fatalf("signer: %v", err)
	}
	token, err := signer.IssueDevice("u_test", "dev_a", "browser")
	if err != nil {
		t.Fatalf("token: %v", err)
	}

	events := box.NewEvents()
	r := gin.New()
	handler.RegisterBoxEventsRoutes(r, handler.BoxEventsDeps{
		Signer: signer,
		Events: events,
		Log:    log,
	})
	srv := httptest.NewServer(r)
	defer srv.Close()

	// --- 1. auth gate
	rsp, err := http.Get(srv.URL + "/box/events")
	if err != nil {
		t.Fatalf("unauth get: %v", err)
	}
	_ = rsp.Body.Close()
	if rsp.StatusCode != http.StatusUnauthorized {
		t.Fatalf("no-auth status = %d, want 401", rsp.StatusCode)
	}

	// --- 2. happy path: open stream, publish, see the record
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	req, _ := http.NewRequestWithContext(ctx, http.MethodGet, srv.URL+"/box/events", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	rsp, err = http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("do: %v", err)
	}
	defer func() { _ = rsp.Body.Close() }()
	if got := rsp.Header.Get("Content-Type"); !strings.HasPrefix(got, "text/event-stream") {
		t.Fatalf("content-type = %q, want text/event-stream*", got)
	}

	// Give the handler a moment to register its subscription before we
	// publish — otherwise Publish runs against an empty subscriber set
	// and the event is dropped on the floor (deterministic deadlock from
	// the test's point of view).
	time.Sleep(50 * time.Millisecond)

	delivered := events.Publish("u_test", box.Event{
		Type:        box.EventBundleReady,
		BoxID:       "box_42",
		Version:     "v9",
		URI:         "https://example.invalid/bundle",
		ContentHash: "sha256:abc",
		SizeBytes:   123,
	})
	if delivered == 0 {
		t.Fatal("publish reached no subscribers")
	}

	// Consume lines until we see the bundle_ready data line, then close.
	type result struct {
		event string
		data  string
	}
	got := make(chan result, 1)
	go func() {
		scanner := bufio.NewScanner(rsp.Body)
		scanner.Buffer(make([]byte, 64*1024), 1024*1024)
		var lastEvent string
		for scanner.Scan() {
			line := scanner.Text()
			switch {
			case strings.HasPrefix(line, "event: "):
				lastEvent = strings.TrimPrefix(line, "event: ")
			case strings.HasPrefix(line, "data: "):
				if lastEvent == string(box.EventBundleReady) {
					got <- result{event: lastEvent, data: strings.TrimPrefix(line, "data: ")}
					return
				}
			}
		}
	}()

	select {
	case r := <-got:
		if r.event != "bundle_ready" {
			t.Errorf("event = %q, want bundle_ready", r.event)
		}
		if !strings.Contains(r.data, `"box_id":"box_42"`) {
			t.Errorf("data missing box_id: %s", r.data)
		}
		if !strings.Contains(r.data, `"version":"v9"`) {
			t.Errorf("data missing version: %s", r.data)
		}
		if !strings.Contains(r.data, `"uri":"https://example.invalid/bundle"`) {
			t.Errorf("data missing uri: %s", r.data)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for bundle_ready record")
	}
}
