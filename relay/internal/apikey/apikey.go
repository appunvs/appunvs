// Package apikey provides the gin middleware that accepts either an
// "apvs_..." API key or a session JWT in the Authorization header and
// sets the resolved user on the gin.Context. It is intentionally separate
// from the store layer: handlers that require "user-level" auth compose
// this middleware, and the store interface below lets tests stub it.
package apikey

import (
	"context"
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"

	"github.com/appunvs/appunvs/relay/internal/auth"
	"github.com/appunvs/appunvs/relay/internal/store"
)

// Context keys the middleware sets on the *gin.Context. Handlers look these
// up with c.GetString / c.MustGet to learn who the caller is.
const (
	// CtxUserID is the authenticated user's primary key ("u_...").
	CtxUserID = "user_id"
	// CtxAuthKind is "apikey" or "session". Handlers that want to deny
	// account-admin ops over an api key check this value.
	CtxAuthKind = "auth_kind"
	// CtxAPIKeyID is set only when auth_kind == "apikey"; it's the key's
	// own id ("ak_..."), useful for audit logging.
	CtxAPIKeyID = "api_key_id"
)

// Auth kind values for CtxAuthKind.
const (
	KindAPIKey  = "apikey"
	KindSession = "session"
)

// Verifier is the narrow slice of the API-key store the middleware needs.
// Defining it here keeps apikey.Authenticate unit-testable without a real
// SQLite database.
type Verifier interface {
	VerifySecret(ctx context.Context, secret string) (*store.APIKey, error)
	Touch(ctx context.Context, keyID string) error
}

// Authenticate builds a gin middleware that accepts either flavor of bearer
// token on the Authorization header:
//
//   - "Bearer apvs_..." — an API key. Looked up in keys. On hit the
//     middleware sets user_id, auth_kind=apikey, api_key_id and bumps
//     last_used_at in a background goroutine.
//   - "Bearer <jwt>" — a session JWT. Verified by signer with typ=session.
//     Device tokens are rejected so programmatic access via a stolen
//     device token is not possible here.
//
// On any failure the middleware writes 401 and aborts the chain.
func Authenticate(signer *auth.Signer, keys Verifier, log *zap.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		raw := c.GetHeader("Authorization")
		if !strings.HasPrefix(raw, "Bearer ") {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "missing bearer token"})
			return
		}
		tok := strings.TrimSpace(strings.TrimPrefix(raw, "Bearer "))
		if tok == "" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "missing bearer token"})
			return
		}

		if strings.HasPrefix(tok, store.APIKeyNamespace) {
			key, err := keys.VerifySecret(c.Request.Context(), tok)
			if err != nil {
				if !errors.Is(err, store.ErrInvalidKey) {
					log.Error("apikey: verify", zap.Error(err))
				}
				c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "invalid api key"})
				return
			}
			// Fire-and-forget touch; failure is fine (last_used_at is a hint,
			// not a correctness invariant).
			go func(id string) {
				ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
				defer cancel()
				if err := keys.Touch(ctx, id); err != nil {
					log.Debug("apikey: touch", zap.Error(err), zap.String("key_id", id))
				}
			}(key.ID)

			c.Set(CtxUserID, key.UserID)
			c.Set(CtxAuthKind, KindAPIKey)
			c.Set(CtxAPIKeyID, key.ID)
			c.Next()
			return
		}

		claims, err := signer.Verify(tok, auth.TokenSession)
		if err != nil {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "invalid session token"})
			return
		}
		c.Set(CtxUserID, claims.UserID)
		c.Set(CtxAuthKind, KindSession)
		c.Next()
	}
}
