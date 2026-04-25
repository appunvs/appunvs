// Package auth's account.go holds the multi-tenant HTTP handlers that
// replaced the scaffold's auto-minting /auth/register.
//
// Surface:
//
//	POST /auth/signup     public   -> session token
//	POST /auth/login      public   -> session token
//	POST /auth/register   session  -> device token
//	GET  /auth/me         session  -> profile + devices
//
// See docs/auth.md for the full contract.
package auth

import (
	"errors"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"go.uber.org/zap"

	"github.com/appunvs/appunvs/relay/internal/pb"
	"github.com/appunvs/appunvs/relay/internal/store"
)

// Deps bundles the collaborators the account handlers need.
type Deps struct {
	Signer *Signer
	Store  *store.Store
	Log    *zap.Logger
}

type signupBody struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

type sessionResponse struct {
	UserID       string `json:"user_id"`
	SessionToken string `json:"session_token"`
}

// Signup creates a user and returns a session token.
func Signup(d Deps) gin.HandlerFunc {
	return func(c *gin.Context) {
		var body signupBody
		if err := c.ShouldBindJSON(&body); err != nil || body.Email == "" || body.Password == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid body"})
			return
		}
		u, err := d.Store.Users().Create(c.Request.Context(), body.Email, body.Password)
		if err != nil {
			if errors.Is(err, store.ErrUserExists) {
				c.JSON(http.StatusConflict, gin.H{"error": "email already registered"})
				return
			}
			d.Log.Info("signup: create user", zap.Error(err))
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		tok, err := d.Signer.IssueSession(u.ID)
		if err != nil {
			d.Log.Error("signup: sign session", zap.Error(err))
			c.JSON(http.StatusInternalServerError, gin.H{"error": "sign failed"})
			return
		}
		// Auto-subscribe every new account to the free tier. Non-fatal on
		// failure — we log and continue so a transient DB hiccup doesn't lock
		// the user out of their brand-new account; the next /billing/status
		// or ws send will heal by re-running EnsureFree behind the middleware.
		if err := d.Store.Subscriptions().EnsureFree(c.Request.Context(), u.ID); err != nil {
			d.Log.Error("signup: ensure free sub", zap.Error(err), zap.String("user_id", u.ID))
		}
		d.Log.Info("signup", zap.String("user_id", u.ID))
		c.JSON(http.StatusOK, sessionResponse{UserID: u.ID, SessionToken: tok})
	}
}

// Login verifies credentials and returns a session token.
func Login(d Deps) gin.HandlerFunc {
	return func(c *gin.Context) {
		var body signupBody
		if err := c.ShouldBindJSON(&body); err != nil || body.Email == "" || body.Password == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid body"})
			return
		}
		u, err := d.Store.Users().VerifyLogin(c.Request.Context(), body.Email, body.Password)
		if err != nil {
			if errors.Is(err, store.ErrBadCredentials) {
				c.JSON(http.StatusUnauthorized, gin.H{"error": "bad credentials"})
				return
			}
			d.Log.Error("login: verify", zap.Error(err))
			c.JSON(http.StatusInternalServerError, gin.H{"error": "internal"})
			return
		}
		tok, err := d.Signer.IssueSession(u.ID)
		if err != nil {
			d.Log.Error("login: sign session", zap.Error(err))
			c.JSON(http.StatusInternalServerError, gin.H{"error": "sign failed"})
			return
		}
		c.JSON(http.StatusOK, sessionResponse{UserID: u.ID, SessionToken: tok})
	}
}

// Register persists a device for the logged-in user and returns a device token.
// Requires a session JWT in the Authorization header.
func Register(d Deps) gin.HandlerFunc {
	return func(c *gin.Context) {
		claims, err := extractSession(c, d.Signer)
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
			return
		}
		var req pb.RegisterRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid body"})
			return
		}
		if req.DeviceID == "" {
			req.DeviceID = "d_" + strings.ReplaceAll(uuid.NewString(), "-", "")
		}
		platform := pb.ParsePlatform(req.Platform).String()

		if _, err := d.Store.Devices().Register(c.Request.Context(), req.DeviceID, claims.UserID, platform); err != nil {
			if errors.Is(err, store.ErrDeviceConflict) {
				c.JSON(http.StatusConflict, gin.H{"error": "device belongs to another user"})
				return
			}
			d.Log.Error("register: persist device", zap.Error(err))
			c.JSON(http.StatusInternalServerError, gin.H{"error": "internal"})
			return
		}
		tok, err := d.Signer.IssueDevice(claims.UserID, req.DeviceID, platform)
		if err != nil {
			d.Log.Error("register: sign device", zap.Error(err))
			c.JSON(http.StatusInternalServerError, gin.H{"error": "sign failed"})
			return
		}
		c.JSON(http.StatusOK, pb.RegisterResponse{Token: tok, UserID: claims.UserID})
	}
}

// deviceView is the on-the-wire shape for `MeResponse.devices`, matching
// shared/proto/*.proto's Device message.  Keys are snake_case and
// timestamps are unix millis (0 when never seen).
type deviceView struct {
	ID        string `json:"id"`
	UserID    string `json:"user_id"`
	Platform  string `json:"platform"`
	CreatedAt int64  `json:"created_at"`
	LastSeen  int64  `json:"last_seen"`
}

// meResponse matches appunvs.MeResponse.
type meResponse struct {
	UserID    string       `json:"user_id"`
	Email     string       `json:"email"`
	CreatedAt int64        `json:"created_at"`
	Devices   []deviceView `json:"devices"`
}

// Me returns the logged-in user's profile and devices.
func Me(d Deps) gin.HandlerFunc {
	return func(c *gin.Context) {
		claims, err := extractSession(c, d.Signer)
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
			return
		}
		u, err := d.Store.Users().Get(c.Request.Context(), claims.UserID)
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "unknown user"})
			return
		}
		rows, err := d.Store.Devices().ListByUser(c.Request.Context(), claims.UserID)
		if err != nil {
			d.Log.Error("me: devices", zap.Error(err))
			c.JSON(http.StatusInternalServerError, gin.H{"error": "internal"})
			return
		}
		devices := make([]deviceView, 0, len(rows))
		for _, dev := range rows {
			var lastSeen int64
			if dev.LastSeen != nil {
				lastSeen = dev.LastSeen.UnixMilli()
			}
			devices = append(devices, deviceView{
				ID:        dev.ID,
				UserID:    dev.UserID,
				Platform:  dev.Platform,
				CreatedAt: dev.CreatedAt.UnixMilli(),
				LastSeen:  lastSeen,
			})
		}
		c.JSON(http.StatusOK, meResponse{
			UserID:    u.ID,
			Email:     u.Email,
			CreatedAt: u.CreatedAt.UnixMilli(),
			Devices:   devices,
		})
	}
}

// extractSession pulls a session JWT from the Authorization header and
// verifies it. Returns a human-safe error message for response bodies.
func extractSession(c *gin.Context, signer *Signer) (*Claims, error) {
	h := c.GetHeader("Authorization")
	if !strings.HasPrefix(h, "Bearer ") {
		return nil, errors.New("missing Authorization bearer token")
	}
	claims, err := signer.Verify(strings.TrimPrefix(h, "Bearer "), TokenSession)
	if err != nil {
		return nil, errors.New("invalid session token")
	}
	return claims, nil
}
