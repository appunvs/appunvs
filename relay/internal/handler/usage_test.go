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

// usageRig spins up a minimal /usage/me server with all three quota
// dimensions wired against a real SQLite store.
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
		Quota:   usage.NewQuota(st.Turns(), st.Boxes(), st.Boxes()),
		PlanFor: func(_ string) usage.Plan { return plan },
		Log:     log,
	})
	srv := httptest.NewServer(r)
	t.Cleanup(srv.Close)

	return &usageRig{srv: srv, token: token, store: st}
}

func (r *usageRig) seedTurns(t *testing.T, n int, ageEach time.Duration, costEach int64) {
	t.Helper()
	now := time.Now().UTC()
	for i := 0; i < n; i++ {
		if err := r.store.Turns().Insert(context.Background(), store.Turn{
			ID:         "seed-" + strconv.Itoa(i),
			BoxID:      "box-u",
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

func TestUsageMe_ReturnsThreeQuanta(t *testing.T) {
	rig := newUsageRig(t, usage.Plans[usage.PlanFree])
	rig.seedTurns(t, 7, time.Hour, 3) // 7 turns × 3 cents = 21 cents

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
		LLM struct {
			UsedCentsLast5d     int64 `json:"used_cents_last_5d"`
			UsedCentsThisMonth  int64 `json:"used_cents_this_month"`
			BudgetCentsPer5d    int   `json:"budget_cents_per_5d"`
			BudgetCentsPerMonth int   `json:"budget_cents_per_month"`
		} `json:"llm"`
		Sandbox struct {
			UsedThisMonth int64 `json:"used_this_month"`
			PerMonth      int   `json:"per_month"`
		} `json:"sandbox"`
		Storage struct {
			Active      int64 `json:"active"`
			ActiveLimit int   `json:"active_limit"`
		} `json:"storage"`
	}
	if err := json.NewDecoder(rsp.Body).Decode(&body); err != nil {
		t.Fatalf("decode: %v", err)
	}

	if body.Plan.ID != "free" {
		t.Errorf("plan.id = %q, want free", body.Plan.ID)
	}
	if body.LLM.UsedCentsLast5d != 21 {
		t.Errorf("llm.used_cents_last_5d = %d, want 21", body.LLM.UsedCentsLast5d)
	}
	if body.LLM.UsedCentsThisMonth < 21 {
		t.Errorf("llm.used_cents_this_month = %d, want >= 21", body.LLM.UsedCentsThisMonth)
	}
	free := usage.Plans[usage.PlanFree]
	if body.LLM.BudgetCentsPer5d != free.LLMBudgetCentsPer5d {
		t.Errorf("llm.budget_cents_per_5d = %d, want %d",
			body.LLM.BudgetCentsPer5d, free.LLMBudgetCentsPer5d)
	}
	if body.LLM.BudgetCentsPerMonth != free.LLMBudgetCentsPerMonth {
		t.Errorf("llm.budget_cents_per_month = %d, want %d",
			body.LLM.BudgetCentsPerMonth, free.LLMBudgetCentsPerMonth)
	}
	// We seeded 1 box; storage active should be 1.
	if body.Storage.Active != 1 {
		t.Errorf("storage.active = %d, want 1", body.Storage.Active)
	}
	if body.Storage.ActiveLimit != free.ActiveBoxes {
		t.Errorf("storage.active_limit = %d, want %d",
			body.Storage.ActiveLimit, free.ActiveBoxes)
	}
}

func TestUsageMe_PropagatesUnlimitedSentinel(t *testing.T) {
	// Max+ has -1 in 5d budget AND active boxes — host-shell decoder
	// treats negatives as "no cap".
	rig := newUsageRig(t, usage.Plans[usage.PlanMaxPlus])

	rsp := rig.get(t)
	defer func() { _ = rsp.Body.Close() }()
	if rsp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want 200", rsp.StatusCode)
	}

	var body struct {
		LLM struct {
			BudgetCentsPer5d int `json:"budget_cents_per_5d"`
		} `json:"llm"`
		Storage struct {
			ActiveLimit int `json:"active_limit"`
		} `json:"storage"`
	}
	if err := json.NewDecoder(rsp.Body).Decode(&body); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if body.LLM.BudgetCentsPer5d != -1 {
		t.Errorf("llm.budget_cents_per_5d = %d, want -1", body.LLM.BudgetCentsPer5d)
	}
	if body.Storage.ActiveLimit != -1 {
		t.Errorf("storage.active_limit = %d, want -1", body.Storage.ActiveLimit)
	}
}
