// Command server is the appunvs relay entrypoint.
package main

import (
	"context"
	"flag"
	"log"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"

	"github.com/appunvs/appunvs/relay/internal/ai"
	"github.com/appunvs/appunvs/relay/internal/artifact"
	"github.com/appunvs/appunvs/relay/internal/auth"
	"github.com/appunvs/appunvs/relay/internal/billing"
	"github.com/appunvs/appunvs/relay/internal/box"
	"github.com/appunvs/appunvs/relay/internal/config"
	"github.com/appunvs/appunvs/relay/internal/handler"
	"github.com/appunvs/appunvs/relay/internal/hub"
	"github.com/appunvs/appunvs/relay/internal/pairing"
	"github.com/appunvs/appunvs/relay/internal/sandbox"
	"github.com/appunvs/appunvs/relay/internal/sequencer"
	"github.com/appunvs/appunvs/relay/internal/store"
	"github.com/appunvs/appunvs/relay/internal/stream"
	"github.com/appunvs/appunvs/relay/internal/workspace"
)

func main() {
	cfgPath := flag.String("config", "config/config.yaml", "path to YAML config")
	flag.Parse()

	cfg, err := config.Load(*cfgPath)
	if err != nil {
		log.Fatalf("config: %v", err)
	}

	logger, err := newLogger(cfg.Log.Level)
	if err != nil {
		log.Fatalf("logger: %v", err)
	}
	defer logger.Sync() //nolint:errcheck

	rdb := redis.NewClient(&redis.Options{
		Addr:     cfg.Redis.Addr,
		Password: cfg.Redis.Password,
		DB:       cfg.Redis.DB,
	})
	defer func() { _ = rdb.Close() }()

	// Best-effort ping; failure is logged but not fatal so the process can
	// boot before Redis in compose ordering.
	{
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		if err := rdb.Ping(ctx).Err(); err != nil {
			logger.Warn("redis ping failed at startup", zap.Error(err))
		}
		cancel()
	}

	signer, err := auth.NewSigner(
		cfg.Auth.PrivateKeyPath,
		cfg.Auth.PublicKeyPath,
		cfg.Auth.Issuer,
		cfg.Auth.Audience,
		time.Duration(cfg.Auth.SessionHours)*time.Hour,
		time.Duration(cfg.Auth.DeviceDays)*24*time.Hour,
		logger,
	)
	if err != nil {
		logger.Fatal("auth signer", zap.Error(err))
	}

	// Persistent relay state (users, devices, etc).
	if err := ensureParentDir(cfg.DB.Path); err != nil {
		logger.Fatal("db path prepare", zap.Error(err))
	}
	dbCtx, dbCancel := context.WithTimeout(context.Background(), 10*time.Second)
	st, err := store.Open(dbCtx, cfg.DB.Path)
	dbCancel()
	if err != nil {
		logger.Fatal("store open", zap.Error(err))
	}
	defer func() { _ = st.Close() }()

	h := hub.New(logger)
	seqSvc := sequencer.New(rdb)
	streamSvc := stream.New(rdb, cfg.Stream.MaxLen)

	gin.SetMode(gin.ReleaseMode)
	r := gin.New()
	r.Use(gin.Recovery())

	accountDeps := auth.Deps{Signer: signer, Store: st, Log: logger}

	r.GET("/health", func(c *gin.Context) { c.String(http.StatusOK, "ok") })
	r.POST("/auth/signup", auth.Signup(accountDeps))
	r.POST("/auth/login", auth.Login(accountDeps))
	r.POST("/auth/register", auth.Register(accountDeps))
	r.GET("/auth/me", auth.Me(accountDeps))
	handler.APIKeyRoutes(r, handler.APIKeyDeps{
		Signer: signer,
		Store:  st,
		Log:    logger,
	})
	handler.RegisterSchemaRoutes(r, handler.SchemaDeps{
		Signer: signer,
		Store:  st,
		Hub:    h,
		Seq:    seqSvc,
		Stream: streamSvc,
		Log:    logger,
	})

	// Billing: Stripe checkout + webhook + plan/status readouts. Webhook
	// is intentionally OUTSIDE the session-auth middleware — Stripe posts
	// it unauthenticated and we verify the signature inline.
	billingSvc := billing.New(st, logger,
		cfg.Billing.StripeSecretKey, cfg.Billing.StripeWebhookSecret,
		cfg.Billing.CheckoutSuccessURL, cfg.Billing.CheckoutCancelURL,
	)
	billingDeps := handler.BillingDeps{Signer: signer, Billing: billingSvc, Log: logger}
	r.GET("/billing/plans", handler.BillingPlans(billingDeps))
	r.GET("/billing/status", handler.BillingStatus(billingDeps))
	r.POST("/billing/checkout", handler.BillingCheckout(billingDeps))
	r.POST("/billing/webhook", handler.BillingWebhook(billingDeps))
	logger.Info("billing wired", zap.String("mode", billingSvc.Mode()))

	r.GET("/ws", handler.WS(handler.Deps{
		Signer: signer,
		Hub:    h,
		Seq:    seqSvc,
		Stream: streamSvc,
		Store:  st,
		Quota:  billing.NewGate(st, logger),
		Log:    logger,
	}))

	// Stage pipeline: artifact storage + sandbox builder + box service.  The
	// LocalFS / LocalStub combo gives a working end-to-end loop without any
	// cloud dependency; production swaps in a real object store and a real
	// Metro builder via the same interfaces.
	artStore, err := artifact.NewLocalFS(cfg.Artifact.Root, cfg.Artifact.BaseURL)
	if err != nil {
		logger.Fatal("artifact store", zap.Error(err))
	}
	// Serve the LocalFS bundles under /_artifacts so the runner can fetch
	// them in dev.  Production binds this URL to a CDN edge instead.
	r.Static("/_artifacts", cfg.Artifact.Root)

	// Per-Box git workspace.  AI fs tools commit into this, publish reads
	// its HEAD snapshot.  Durable on the relay's local disk; future slice
	// moves this behind an object-store-mounted filesystem.
	ws, err := workspace.NewStore(workspace.Config{Root: cfg.Workspace.Root})
	if err != nil {
		logger.Fatal("workspace store", zap.Error(err))
	}
	boxSvc := box.New(st.Boxes(), sandbox.NewLocalStub(), artStore, ws)
	handler.RegisterBoxRoutes(r, handler.BoxDeps{
		Signer:  signer,
		Service: boxSvc,
		Log:     logger,
	})

	pairSvc := pairing.New(rdb)
	handler.RegisterPairingRoutes(r, handler.PairingDeps{
		Signer:  signer,
		Service: pairSvc,
		Boxes:   boxSvc,
		Log:     logger,
	})
	logger.Info("stage pipeline wired",
		zap.String("artifact_backend", cfg.Artifact.Backend),
		zap.String("artifact_root", cfg.Artifact.Root))

	// AI agent: DeepSeek by default via OpenAI-compatible protocol.  Set
	// AI_BACKEND=stub (or leave blank without an APIKey) to run the echo
	// engine — useful for UI work and CI where we don't want to burn
	// provider tokens.  Real production config flips backend=deepseek
	// (or any OpenAI-compatible endpoint) and supplies APIKey.
	var aiEngine ai.Engine
	switch cfg.AI.Backend {
	case "deepseek", "openai-compatible":
		de, err := ai.NewDeepSeekEngine(ai.Config{
			BaseURL:   cfg.AI.BaseURL,
			APIKey:    cfg.AI.APIKey,
			Model:     cfg.AI.Model,
			MaxIters:  cfg.AI.MaxIters,
			MaxTokens: cfg.AI.MaxTokens,
		}, ws, boxSvc, st.Turns(), logger)
		if err != nil {
			logger.Fatal("ai engine", zap.Error(err))
		}
		aiEngine = de
		logger.Info("ai engine wired",
			zap.String("backend", cfg.AI.Backend),
			zap.String("model", cfg.AI.Model),
			zap.String("base_url", cfg.AI.BaseURL))
	default:
		aiEngine = ai.NewStub()
		logger.Info("ai engine wired", zap.String("backend", "stub"))
	}
	handler.RegisterAIRoutes(r, handler.AIDeps{
		Signer: signer,
		Engine: aiEngine,
		Box:    boxSvc,
		Log:    logger,
	})

	srv := &http.Server{
		Addr:              cfg.Listen,
		Handler:           r,
		ReadHeaderTimeout: 10 * time.Second,
	}

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	go func() {
		logger.Info("relay listening", zap.String("addr", cfg.Listen))
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Fatal("listen", zap.Error(err))
		}
	}()

	<-ctx.Done()
	logger.Info("shutdown: draining")
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := srv.Shutdown(shutdownCtx); err != nil {
		logger.Error("shutdown", zap.Error(err))
	}
}

func newLogger(level string) (*zap.Logger, error) {
	cfg := zap.NewProductionConfig()
	cfg.DisableStacktrace = true
	if lvl, err := zap.ParseAtomicLevel(level); err == nil {
		cfg.Level = lvl
	}
	return cfg.Build()
}

// ensureParentDir makes sure the directory holding path exists (0755).
// SQLite won't auto-create it, and having the relay crash on first run
// because data/ is missing is a bad operator experience.
func ensureParentDir(path string) error {
	dir := filepath.Dir(path)
	if dir == "" || dir == "." || dir == "/" {
		return nil
	}
	return os.MkdirAll(dir, 0o755)
}
