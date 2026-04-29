package handler_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"

	"github.com/appunvs/appunvs/relay/internal/auth"
	"github.com/appunvs/appunvs/relay/internal/handler"
	"github.com/appunvs/appunvs/relay/internal/store"
	"github.com/appunvs/appunvs/relay/internal/usage"
)

// usageRig spins up a minimal /usage/me server.  Box / engine / etc.
// aren't needed — the endpoint only cares about the user's plan and
// turn counts.
type usageRig struct {
	srv   *httptest.Server
	token string
	store *store.Store
}

func newUsageRig(t *testing.T, plan usage.Plan) *usageRig {
	t.Helper()
	gin.SetMode(gin.TestMode)
	ctx := context.Background()

	tmp := t.TempDir()
	st, err := store.Open(ctx, tmp+"/relay.db")
	if err != nil {
		t.Fatalf("store: %v", err)
	}
	t.Cleanup(func() { _ = st.Close() })

	if _, err := st.DB.ExecContext(ctx,
		`INSERT INTO users(id, email, password_hash, created_at) VALUES(?,?,?,?)`,
		"u_usage", "u@example.com", "x", int64(1)); err != nil {
		t.Fatalf("seed user: %v", err)
	}
	// Box owned by the user — we need at least one for the
	// CountByNamespace JOIN to find anything when we seed turns.
	if _, err := st.DB.ExecContext(ctx,
		`INSERT INTO app_boxes(id, namespace, provider_device_id, title, runtime, state, current_version, created_at, updated_at) VALUES(?,?,?,?,?,?,?,?,?)`,
		"box-u", "u_usage", "dev_u", "demo", "rn_bundle", "draft", "", int64(1), int64(1)); err != nil {
		t.Fatalf("seed box: %v", err)
	}

	log := zap.NewNop()
	signer, err := auth.NewSigner("", "", "appunvs-test", "appunvs-test", time.Hour, time.Hour, log)
	if err != nil {
		t.Fatalf("signer: %v", err)
	}
	token, err := signer.IssueDevice("u_usage", "dev_u", "browser")
	if err != nil {
		t.Fatalf("token: %v", err)
	}

	r := gin.New()
	handler.RegisterUsageRoutes(r, handler.UsageDeps{
		Signer:  signer,
		Quota:   usage.NewQuota(st.Turns()),
		PlanFor: func(_ string) usage.Plan { return plan },
		Log:     log,
	})
	srv := httptest.NewServer(r)
	t.Cleanup(srv.Close)

	return &usageRig{srv: srv, token: token, store: st}
}

func (r *usageRig) seedTurns(t *testing.T, n int, ageEach time.Duration) {
	t.Helper()
	now := time.Now().UTC()
	for i := 0; i < n; i++ {
		if err := r.store.Turns().Insert(context.Background(), store.Turn{
			ID:         "seed-" + strconv.Itoa(i),
			BoxID:      "box-u",
			UserText:   "seed",
			Messages:   "[]",
			StopReason: "end_turn",
			CreatedAt:  now.Add(-ageEach * time.Duration(i+1)).UnixMilli(),
		}); err != nil {
			t.Fatalf("seed turn %d: %v", i, err)
		}
	}
}

func (r *usageRig) get(t *testing.T) *http.Response {
	t.Helper()
	req, _ := http.NewRequest(http.MethodGet, r.srv.URL+"/usage/me", nil)
	req.Header.Set("Authorization", "Bearer "+r.token)
	rsp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	return rsp
}

func TestUsageMe_RequiresAuth(t *testing.T) {
	rig := newUsageRig(t, usage.Plans[usage.PlanFree])
	rsp, err := http.Get(rig.srv.URL + "/usage/me")
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	defer func() { _ = rsp.Body.Close() }()
	if rsp.StatusCode != http.StatusUnauthorized {
		t.Errorf("status = %d, want 401", rsp.StatusCode)
	}
}

func TestUsageMe_ReturnsPlanAndCounts(t *testing.T) {
	rig := newUsageRig(t, usage.Plans[usage.PlanFree])
	rig.seedTurns(t, 7, time.Hour) // all within 5d window

	rsp := rig.get(t)
	defer func() { _ = rsp.Body.Close() }()
	if rsp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want 200", rsp.StatusCode)
	}

	var body struct {
		Plan struct {
			ID    string `json:"id"`
			Label string `json:"label"`
		} `json:"plan"`
		Used struct {
			Last5d    int64 `json:"last_5d"`
			ThisMonth int64 `json:"this_month"`
		} `json:"used"`
		Limits struct {
			TurnsPer5d        int `json:"turns_per_5d"`
			TurnsPerMonth     int `json:"turns_per_month"`
			ActiveBoxes       int `json:"active_boxes"`
			PublishesPerMonth int `json:"publishes_per_month"`
		} `json:"limits"`
	}
	if err := json.NewDecoder(rsp.Body).Decode(&body); err != nil {
		t.Fatalf("decode: %v", err)
	}

	if body.Plan.ID != "free" {
		t.Errorf("plan.id = %q, want free", body.Plan.ID)
	}
	if body.Plan.Label != "Free" {
		t.Errorf("plan.label = %q, want Free", body.Plan.Label)
	}
	if body.Used.Last5d != 7 {
		t.Errorf("used.last_5d = %d, want 7", body.Used.Last5d)
	}
	if body.Used.ThisMonth < 7 {
		t.Errorf("used.this_month = %d, want >= 7", body.Used.ThisMonth)
	}
	free := usage.Plans[usage.PlanFree]
	if body.Limits.TurnsPer5d != free.TurnsPer5d {
		t.Errorf("limits.turns_per_5d = %d, want %d", body.Limits.TurnsPer5d, free.TurnsPer5d)
	}
	if body.Limits.TurnsPerMonth != free.TurnsPerMonth {
		t.Errorf("limits.turns_per_month = %d, want %d", body.Limits.TurnsPerMonth, free.TurnsPerMonth)
	}
}

func TestUsageMe_PropagatesUnlimitedSentinel(t *testing.T) {
	// Max+ has TurnsPer5d = -1 (unlimited).  The handler should pass
	// the -1 straight through so the host-shell decoder can render
	// "—" / hide the bar.
	rig := newUsageRig(t, usage.Plans[usage.PlanMaxPlus])

	rsp := rig.get(t)
	defer func() { _ = rsp.Body.Close() }()
	if rsp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want 200", rsp.StatusCode)
	}

	var body struct {
		Limits struct {
			TurnsPer5d  int `json:"turns_per_5d"`
			ActiveBoxes int `json:"active_boxes"`
		} `json:"limits"`
	}
	if err := json.NewDecoder(rsp.Body).Decode(&body); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if body.Limits.TurnsPer5d != -1 {
		t.Errorf("turns_per_5d = %d, want -1 (unlimited sentinel)", body.Limits.TurnsPer5d)
	}
	if body.Limits.ActiveBoxes != -1 {
		t.Errorf("active_boxes = %d, want -1 (unlimited sentinel)", body.Limits.ActiveBoxes)
	}
}
