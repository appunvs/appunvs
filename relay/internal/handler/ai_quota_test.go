package handler_test

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strconv"
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
	"github.com/appunvs/appunvs/relay/internal/usage"
	"github.com/appunvs/appunvs/relay/internal/workspace"
)

// quotaTestRig is the same harness ai_test.go builds, lifted out so
// quota tests don't duplicate 60 lines of setup.
type quotaTestRig struct {
	srv    *httptest.Server
	box    string
	token  string
	store  *store.Store
	plan   usage.Plan
	deps   handler.AIDeps
	signer *auth.Signer
}

func newQuotaRig(t *testing.T, plan usage.Plan) *quotaTestRig {
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
		"u_quota", "q@example.com", "x", int64(1)); err != nil {
		t.Fatalf("seed user: %v", err)
	}

	ws, _ := workspace.NewStore(workspace.Config{Root: tmp + "/ws"})
	art, _ := artifact.NewLocalFS(tmp+"/art", "http://localhost:8080/_artifacts")
	boxSvc := box.New(st.Boxes(), sandbox.NewLocalStub(), art, ws, nil)

	boxObj, err := boxSvc.Create(ctx, "u_quota", "dev_q", "demo", pb.RuntimeKindRNBundle)
	if err != nil {
		t.Fatalf("create box: %v", err)
	}

	log := zap.NewNop()
	signer, err := auth.NewSigner("", "", "appunvs-test", "appunvs-test", time.Hour, time.Hour, log)
	if err != nil {
		t.Fatalf("signer: %v", err)
	}
	token, err := signer.IssueDevice("u_quota", "dev_q", "browser")
	if err != nil {
		t.Fatalf("token: %v", err)
	}

	deps := handler.AIDeps{
		Signer:  signer,
		Engine:  ai.NewStub(),
		Box:     boxSvc,
		Log:     log,
		Quota:   usage.NewQuota(st.Turns(), st.Boxes(), st.Boxes()),
		PlanFor: func(_ string) usage.Plan { return plan },
	}

	r := gin.New()
	handler.RegisterAIRoutes(r, deps)
	srv := httptest.NewServer(r)
	t.Cleanup(srv.Close)

	return &quotaTestRig{
		srv: srv, box: boxObj.ID, token: token,
		store: st, plan: plan, deps: deps, signer: signer,
	}
}

// seedTurns inserts n turn rows under the rig's box at given offsets
// in the past, so the rolling counts can be exercised.  costEach is
// the per-turn cost_cents — total seeded LLM spend is n * costEach.
func (r *quotaTestRig) seedTurns(t *testing.T, n int, ageEach time.Duration, costEach int64) {
	t.Helper()
	now := time.Now().UTC()
	for i := 0; i < n; i++ {
		if err := r.store.Turns().Insert(context.Background(), store.Turn{
			ID:         "seed-" + strconv.Itoa(i),
			BoxID:      r.box,
			UserText:   "seed",
			Messages:   "[]",
			Model:      "deepseek-chat",
			CostCents:  costEach,
			StopReason: "end_turn",
			CreatedAt:  now.Add(-ageEach * time.Duration(i+1)).UnixMilli(),
		}); err != nil {
			t.Fatalf("seed turn %d: %v", i, err)
		}
	}
}

func (r *quotaTestRig) post(t *testing.T) *http.Response {
	t.Helper()
	req, _ := http.NewRequest(http.MethodPost, r.srv.URL+"/ai/turn",
		strings.NewReader(`{"box_id":"`+r.box+`","text":"hi"}`))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+r.token)
	rsp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("post: %v", err)
	}
	return rsp
}

// TestAITurn_QuotaAllowsBelowCap — under cap, the gate is a no-op and
// the SSE stream behaves like the unguarded path.
func TestAITurn_QuotaAllowsBelowCap(t *testing.T) {
	rig := newQuotaRig(t, usage.Plans[usage.PlanFree])
	// Free's LLM 5d budget is 50 cents (¥0.50); 5 turns × 1 cent
	// = 5 cents leaves plenty of headroom.
	rig.seedTurns(t, 5, time.Hour, 1)

	rsp := rig.post(t)
	defer func() { _ = rsp.Body.Close() }()
	if rsp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(rsp.Body)
		t.Fatalf("status = %d, want 200; body=%s", rsp.StatusCode, body)
	}
	if got := rsp.Header.Get("Content-Type"); !strings.HasPrefix(got, "text/event-stream") {
		t.Fatalf("content-type = %q, want SSE", got)
	}
}

// TestAITurn_QuotaLLM5dCapReturns429 — once the LLM 5d budget is
// saturated, the gate must short-circuit before invoking the engine.
func TestAITurn_QuotaLLM5dCapReturns429(t *testing.T) {
	plan := usage.Plans[usage.PlanFree]
	rig := newQuotaRig(t, plan)
	// Fill the 5d window up to the cap; one big "expensive" turn
	// equal to the entire budget is enough to trip the gate.
	rig.seedTurns(t, 1, time.Hour, int64(plan.LLMBudgetCentsPer5d))

	rsp := rig.post(t)
	defer func() { _ = rsp.Body.Close() }()
	if rsp.StatusCode != http.StatusTooManyRequests {
		body, _ := io.ReadAll(rsp.Body)
		t.Fatalf("status = %d, want 429; body=%s", rsp.StatusCode, body)
	}
	ra := rsp.Header.Get("Retry-After")
	n, err := strconv.Atoi(ra)
	if err != nil {
		t.Fatalf("Retry-After header = %q, parse err: %v", ra, err)
	}
	if n < 1 {
		t.Errorf("Retry-After = %d, want >=1", n)
	}

	var body struct {
		Error      string `json:"error"`
		LimitedBy  string `json:"limited_by"`
		Plan       string `json:"plan"`
		RetryAfter int    `json:"retry_after"`
		Used       struct {
			LLMCentsLast5d    int64 `json:"llm_cents_last_5d"`
			LLMCentsThisMonth int64 `json:"llm_cents_this_month"`
		} `json:"used"`
	}
	if err := json.NewDecoder(rsp.Body).Decode(&body); err != nil {
		t.Fatalf("decode body: %v", err)
	}
	if body.Error != "plan_exhausted" {
		t.Errorf("error = %q, want plan_exhausted", body.Error)
	}
	if body.LimitedBy != "llm_5d" {
		t.Errorf("limited_by = %q, want llm_5d", body.LimitedBy)
	}
	if body.Plan != "free" {
		t.Errorf("plan = %q, want free", body.Plan)
	}
	if body.Used.LLMCentsLast5d < int64(plan.LLMBudgetCentsPer5d) {
		t.Errorf("used.llm_cents_last_5d = %d, want >= %d",
			body.Used.LLMCentsLast5d, plan.LLMBudgetCentsPer5d)
	}
}

// TestAITurn_QuotaDisabledWhenNil — leaving Quota nil keeps the
// pre-Phase-B behavior so dev / CI / dogfood paths don't break.
func TestAITurn_QuotaDisabledWhenNil(t *testing.T) {
	rig := newQuotaRig(t, usage.Plans[usage.PlanFree])
	// Drop the gate after the rig built it.  Same plumbing minus the
	// quota check.
	rig.deps.Quota = nil
	rig.deps.PlanFor = nil

	r := gin.New()
	handler.RegisterAIRoutes(r, rig.deps)
	srv := httptest.NewServer(r)
	defer srv.Close()

	// Seed enough cost to bust Free's caps — without the gate, the
	// engine still serves the request normally.
	rig.seedTurns(t, 100, time.Hour, 100)

	req, _ := http.NewRequest(http.MethodPost, srv.URL+"/ai/turn",
		strings.NewReader(`{"box_id":"`+rig.box+`","text":"hi"}`))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+rig.token)
	rsp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("post: %v", err)
	}
	defer func() { _ = rsp.Body.Close() }()
	if rsp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(rsp.Body)
		t.Fatalf("status = %d, want 200; body=%s", rsp.StatusCode, body)
	}
}

// TestAITurn_BYOKBypassesGate — BYOK plan has -1 caps; even with the
// gate enabled, the request must succeed.
func TestAITurn_BYOKBypassesGate(t *testing.T) {
	rig := newQuotaRig(t, usage.Plans[usage.PlanBYOK])
	// Massive seeded cost would bust every other tier; -1 LLM cap
	// on BYOK should pass through anyway.
	rig.seedTurns(t, 100, time.Hour, 10000)

	rsp := rig.post(t)
	defer func() { _ = rsp.Body.Close() }()
	if rsp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(rsp.Body)
		t.Fatalf("status = %d, want 200; body=%s", rsp.StatusCode, body)
	}
}
