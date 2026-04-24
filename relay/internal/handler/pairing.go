package handler

import (
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"

	"github.com/appunvs/appunvs/relay/internal/auth"
	"github.com/appunvs/appunvs/relay/internal/box"
	"github.com/appunvs/appunvs/relay/internal/pairing"
	"github.com/appunvs/appunvs/relay/internal/pb"
	"github.com/appunvs/appunvs/relay/internal/store"
)

// PairingDeps groups everything the /pair endpoints touch.
type PairingDeps struct {
	Signer  *auth.Signer
	Service *pairing.Service
	Boxes   *box.Service
	Log     *zap.Logger
}

// RegisterPairingRoutes wires:
//
//	POST /pair               - provider mints a short code bound to a box
//	POST /pair/:code/claim   - connector redeems the code, gets bundle URI
//
// Both endpoints require a device token; the box ownership / namespace
// checks live inside box.Service.
func RegisterPairingRoutes(r gin.IRouter, d PairingDeps) {
	r.POST("/pair", pairIssue(d))
	r.POST("/pair/:code/claim", pairClaim(d))
}

func pairIssue(d PairingDeps) gin.HandlerFunc {
	return func(c *gin.Context) {
		claims, ok := requireDevice(c, d.Signer)
		if !ok {
			return
		}
		var req pb.PairRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid body"})
			return
		}
		if strings.TrimSpace(req.BoxID) == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "box_id required"})
			return
		}
		// Confirm the box exists and is owned by the caller's namespace.
		b, _, err := d.Boxes.Get(c.Request.Context(), claims.UserID, req.BoxID)
		if err != nil {
			if errors.Is(err, store.ErrBoxNotFound) {
				c.JSON(http.StatusNotFound, gin.H{"error": "box not found"})
				return
			}
			d.Log.Error("pair.issue: lookup", zap.Error(err))
			c.JSON(http.StatusInternalServerError, gin.H{"error": "internal"})
			return
		}
		if b.State == pb.PublishStateArchived {
			c.JSON(http.StatusConflict, gin.H{"error": "box archived"})
			return
		}
		ttl := time.Duration(req.TTLSec) * time.Second
		code, expires, err := d.Service.Issue(c.Request.Context(), pairing.Grant{
			BoxID:            b.ID,
			Namespace:        b.Namespace,
			ProviderDeviceID: b.ProviderDeviceID,
		}, ttl)
		if err != nil {
			d.Log.Error("pair.issue: store", zap.Error(err))
			c.JSON(http.StatusInternalServerError, gin.H{"error": "internal"})
			return
		}
		c.JSON(http.StatusOK, pb.PairResponse{
			ShortCode: code,
			ExpiresAt: expires.UnixMilli(),
		})
	}
}

func pairClaim(d PairingDeps) gin.HandlerFunc {
	return func(c *gin.Context) {
		// The connector's device token is required so the relay can attribute
		// the claim and (in the next slice) issue a scoped namespace_token.
		claims, ok := requireDevice(c, d.Signer)
		if !ok {
			return
		}
		code := strings.ToUpper(strings.TrimSpace(c.Param("code")))
		if code == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "missing code"})
			return
		}
		grant, err := d.Service.Claim(c.Request.Context(), code)
		if err != nil {
			if errors.Is(err, pairing.ErrNotFound) {
				c.JSON(http.StatusNotFound, gin.H{"error": "code expired or unknown"})
				return
			}
			d.Log.Error("pair.claim", zap.Error(err))
			c.JSON(http.StatusInternalServerError, gin.H{"error": "internal"})
			return
		}

		// Look up current bundle so the claimer can immediately load Stage.
		// Note: namespace gate uses the *grant's* namespace, not the
		// claimer's — cross-account paring is allowed by construction
		// (that's the whole point of "scan another person's QR").
		b, current, err := d.Boxes.Get(c.Request.Context(), grant.Namespace, grant.BoxID)
		if err != nil {
			d.Log.Error("pair.claim: box lookup", zap.Error(err))
			c.JSON(http.StatusInternalServerError, gin.H{"error": "internal"})
			return
		}
		var bundlePB *pb.BundleRef
		if current != nil {
			ref := box.BundleToPB(*current)
			bundlePB = &ref
		}

		d.Log.Info("pair: claimed",
			zap.String("box_id", b.ID),
			zap.String("namespace", b.Namespace),
			zap.String("claimer_user_id", claims.UserID),
			zap.String("claimer_device_id", claims.DeviceID))

		// TODO(stage-namespace-token): mint a JWT scoped to (box_id, claimer_device_id)
		// so /ws can verify subscription requests.  Empty for v1.
		c.JSON(http.StatusOK, pb.PairClaimResponse{
			BoxID:          b.ID,
			Bundle:         bundlePB,
			NamespaceToken: "",
		})
	}
}
