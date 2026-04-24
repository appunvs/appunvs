package handler

import (
	"errors"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"

	"github.com/appunvs/appunvs/relay/internal/auth"
	"github.com/appunvs/appunvs/relay/internal/store"
)

// APIKeyDeps is the set of collaborators the /keys handlers need. Only the
// signer is required to verify session JWTs — API-key endpoints must be
// session-authenticated (creating a key via another api key would be a
// privilege-escalation hole).
type APIKeyDeps struct {
	Signer *auth.Signer
	Store  *store.Store
	Log    *zap.Logger
}

// APIKeyRoutes registers the /keys endpoints on the given gin router group.
// All routes require a session JWT; device tokens and api keys are rejected.
func APIKeyRoutes(r gin.IRouter, d APIKeyDeps) {
	r.POST("/keys", createAPIKey(d))
	r.GET("/keys", listAPIKeys(d))
	r.DELETE("/keys/:id", revokeAPIKey(d))
}

type createKeyBody struct {
	Name string `json:"name"`
}

type createKeyResponse struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	Prefix    string `json:"prefix"`
	Secret    string `json:"secret"`
	CreatedAt int64  `json:"created_at"`
}

type listKeyItem struct {
	ID         string `json:"id"`
	Name       string `json:"name"`
	Prefix     string `json:"prefix"`
	CreatedAt  int64  `json:"created_at"`
	LastUsedAt *int64 `json:"last_used_at"`
	RevokedAt  *int64 `json:"revoked_at"`
}

func createAPIKey(d APIKeyDeps) gin.HandlerFunc {
	return func(c *gin.Context) {
		userID, ok := requireSession(c, d.Signer)
		if !ok {
			return
		}
		var body createKeyBody
		if err := c.ShouldBindJSON(&body); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid body"})
			return
		}
		name := strings.TrimSpace(body.Name)
		if name == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "name required"})
			return
		}
		key, secret, err := d.Store.APIKeys().Create(c.Request.Context(), userID, name)
		if err != nil {
			d.Log.Error("keys: create", zap.Error(err), zap.String("user_id", userID))
			c.JSON(http.StatusInternalServerError, gin.H{"error": "internal"})
			return
		}
		d.Log.Info("keys: created", zap.String("user_id", userID), zap.String("key_id", key.ID))
		c.JSON(http.StatusOK, createKeyResponse{
			ID:        key.ID,
			Name:      key.Name,
			Prefix:    key.Prefix,
			Secret:    secret,
			CreatedAt: key.CreatedAt.UnixMilli(),
		})
	}
}

func listAPIKeys(d APIKeyDeps) gin.HandlerFunc {
	return func(c *gin.Context) {
		userID, ok := requireSession(c, d.Signer)
		if !ok {
			return
		}
		rows, err := d.Store.APIKeys().List(c.Request.Context(), userID)
		if err != nil {
			d.Log.Error("keys: list", zap.Error(err), zap.String("user_id", userID))
			c.JSON(http.StatusInternalServerError, gin.H{"error": "internal"})
			return
		}
		out := make([]listKeyItem, 0, len(rows))
		for _, r := range rows {
			item := listKeyItem{
				ID:        r.ID,
				Name:      r.Name,
				Prefix:    r.Prefix,
				CreatedAt: r.CreatedAt.UnixMilli(),
			}
			if r.LastUsedAt != nil {
				v := r.LastUsedAt.UnixMilli()
				item.LastUsedAt = &v
			}
			if r.RevokedAt != nil {
				v := r.RevokedAt.UnixMilli()
				item.RevokedAt = &v
			}
			out = append(out, item)
		}
		c.JSON(http.StatusOK, out)
	}
}

func revokeAPIKey(d APIKeyDeps) gin.HandlerFunc {
	return func(c *gin.Context) {
		userID, ok := requireSession(c, d.Signer)
		if !ok {
			return
		}
		id := c.Param("id")
		if id == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "missing id"})
			return
		}
		err := d.Store.APIKeys().Revoke(c.Request.Context(), userID, id)
		if err != nil {
			if errors.Is(err, store.ErrKeyNotFound) {
				c.JSON(http.StatusNotFound, gin.H{"error": "not found"})
				return
			}
			d.Log.Error("keys: revoke", zap.Error(err), zap.String("user_id", userID), zap.String("key_id", id))
			c.JSON(http.StatusInternalServerError, gin.H{"error": "internal"})
			return
		}
		d.Log.Info("keys: revoked", zap.String("user_id", userID), zap.String("key_id", id))
		c.Status(http.StatusNoContent)
	}
}

// requireSession pulls the Authorization header, accepts only a session
// JWT, and writes 401 + aborts on any failure. Returns (userID, true) on
// success. API keys are rejected here on purpose: self-service key
// management is an account-admin operation that must require a fresh
// (logged-in) human, not a long-lived automation credential.
func requireSession(c *gin.Context, signer *auth.Signer) (string, bool) {
	h := c.GetHeader("Authorization")
	if !strings.HasPrefix(h, "Bearer ") {
		c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "missing bearer token"})
		return "", false
	}
	tok := strings.TrimSpace(strings.TrimPrefix(h, "Bearer "))
	if strings.HasPrefix(tok, store.APIKeyNamespace) {
		c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "session token required"})
		return "", false
	}
	claims, err := signer.Verify(tok, auth.TokenSession)
	if err != nil {
		c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "invalid session token"})
		return "", false
	}
	return claims.UserID, true
}
