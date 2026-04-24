package handler

import (
	"encoding/json"
	"errors"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"

	"github.com/appunvs/appunvs/relay/internal/auth"
	"github.com/appunvs/appunvs/relay/internal/box"
	"github.com/appunvs/appunvs/relay/internal/pb"
	"github.com/appunvs/appunvs/relay/internal/sandbox"
	"github.com/appunvs/appunvs/relay/internal/store"
)

// BoxDeps groups everything the /box endpoints touch.
type BoxDeps struct {
	Signer  *auth.Signer
	Service *box.Service
	Log     *zap.Logger
}

// RegisterBoxRoutes wires GET / POST / DELETE on /box and POST /box/:id/publish.
// All routes require a device token (the provider device that owns the box).
func RegisterBoxRoutes(r gin.IRouter, d BoxDeps) {
	r.POST("/box", boxCreate(d))
	r.GET("/box", boxList(d))
	r.GET("/box/:id", boxGet(d))
	r.POST("/box/:id/publish", boxPublish(d))
	r.DELETE("/box/:id", boxArchive(d))
}

// publishRequestBody is the optional payload accepted by POST /box/:id/publish.
// `entry_point` defaults to "index.tsx"; `files` is the AI-authored project
// snapshot that gets fed straight into the sandbox.  Real production will
// take the source from a server-side workspace rather than the request body,
// but the explicit shape here lets the very first end-to-end test flow run
// without standing up that workspace.
type publishRequestBody struct {
	EntryPoint string            `json:"entry_point,omitempty"`
	Files      map[string]string `json:"files,omitempty"`
}

func boxCreate(d BoxDeps) gin.HandlerFunc {
	return func(c *gin.Context) {
		claims, ok := requireDevice(c, d.Signer)
		if !ok {
			return
		}
		var req pb.BoxCreateRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid body"})
			return
		}
		if strings.TrimSpace(req.Title) == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "title required"})
			return
		}
		b, err := d.Service.Create(c.Request.Context(), claims.UserID, claims.DeviceID, req.Title, req.Runtime)
		if err != nil {
			d.Log.Error("box.create", zap.Error(err))
			c.JSON(http.StatusInternalServerError, gin.H{"error": "internal"})
			return
		}
		c.JSON(http.StatusOK, box.ToPB(b, nil))
	}
}

func boxGet(d BoxDeps) gin.HandlerFunc {
	return func(c *gin.Context) {
		claims, ok := requireDevice(c, d.Signer)
		if !ok {
			return
		}
		id := c.Param("id")
		b, current, err := d.Service.Get(c.Request.Context(), claims.UserID, id)
		if err != nil {
			if errors.Is(err, store.ErrBoxNotFound) {
				c.JSON(http.StatusNotFound, gin.H{"error": "not found"})
				return
			}
			d.Log.Error("box.get", zap.Error(err))
			c.JSON(http.StatusInternalServerError, gin.H{"error": "internal"})
			return
		}
		c.JSON(http.StatusOK, box.ToPB(b, current))
	}
}

func boxList(d BoxDeps) gin.HandlerFunc {
	return func(c *gin.Context) {
		claims, ok := requireDevice(c, d.Signer)
		if !ok {
			return
		}
		boxes, err := d.Service.List(c.Request.Context(), claims.UserID)
		if err != nil {
			d.Log.Error("box.list", zap.Error(err))
			c.JSON(http.StatusInternalServerError, gin.H{"error": "internal"})
			return
		}
		out := make([]pb.Box, 0, len(boxes))
		for _, b := range boxes {
			out = append(out, box.ToPB(b, nil).Box)
		}
		c.JSON(http.StatusOK, pb.BoxListResponse{Boxes: out})
	}
}

func boxPublish(d BoxDeps) gin.HandlerFunc {
	return func(c *gin.Context) {
		claims, ok := requireDevice(c, d.Signer)
		if !ok {
			return
		}
		id := c.Param("id")

		var body publishRequestBody
		// Accept empty body — useful when the source has been pre-staged
		// server-side (a future workspace API).  When body.Files is empty
		// the sandbox stub still produces a (mostly empty) bundle so the
		// downstream wiring can be exercised.
		if c.Request.ContentLength > 0 {
			if err := c.ShouldBindJSON(&body); err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": "invalid body"})
				return
			}
		}
		if body.EntryPoint == "" {
			body.EntryPoint = "index.tsx"
		}
		files := make(map[string][]byte, len(body.Files))
		for k, v := range body.Files {
			files[k] = []byte(v)
		}

		bundle, err := d.Service.BuildAndPublish(c.Request.Context(), claims.UserID, sandbox.Source{
			BoxID:      id,
			EntryPoint: body.EntryPoint,
			Files:      files,
		})
		if err != nil {
			if errors.Is(err, store.ErrBoxNotFound) {
				c.JSON(http.StatusNotFound, gin.H{"error": "not found"})
				return
			}
			d.Log.Error("box.publish", zap.Error(err), zap.String("box_id", id))
			c.JSON(http.StatusInternalServerError, gin.H{"error": "build failed"})
			return
		}

		// Re-read the box so the response carries the freshly bumped state.
		b, _, err := d.Service.Get(c.Request.Context(), claims.UserID, id)
		if err != nil {
			d.Log.Error("box.publish: reload", zap.Error(err))
			c.JSON(http.StatusInternalServerError, gin.H{"error": "internal"})
			return
		}
		bRef := box.BundleToPB(bundle)
		c.JSON(http.StatusOK, box.ToPB(b, &bundle))

		// TODO(stage-fanout): emit a BoxVersionUpdate over the hub to every
		// connector currently subscribed to this box.  Wire-level message
		// shape is fixed (pb.BoxVersionUpdate); the hub-side subscriber
		// registry is the next deliverable.
		_ = bRef
	}
}

func boxArchive(d BoxDeps) gin.HandlerFunc {
	return func(c *gin.Context) {
		claims, ok := requireDevice(c, d.Signer)
		if !ok {
			return
		}
		id := c.Param("id")
		if err := d.Service.Archive(c.Request.Context(), claims.UserID, id); err != nil {
			if errors.Is(err, store.ErrBoxNotFound) {
				c.JSON(http.StatusNotFound, gin.H{"error": "not found"})
				return
			}
			d.Log.Error("box.archive", zap.Error(err))
			c.JSON(http.StatusInternalServerError, gin.H{"error": "internal"})
			return
		}
		c.Status(http.StatusNoContent)
	}
}

// requireDevice extracts a device JWT from the Authorization header or
// 401s.  Pattern intentionally mirrors requireSession in apikey.go.
func requireDevice(c *gin.Context, signer *auth.Signer) (*auth.Claims, bool) {
	h := c.GetHeader("Authorization")
	if !strings.HasPrefix(h, "Bearer ") {
		c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "missing bearer token"})
		return nil, false
	}
	tok := strings.TrimSpace(strings.TrimPrefix(h, "Bearer "))
	claims, err := signer.Verify(tok, auth.TokenDevice)
	if err != nil {
		c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "invalid device token"})
		return nil, false
	}
	return claims, true
}

// reserved — kept for the next slice when we forward a serialized BundleRef
// over the WebSocket fanout to subscribed connectors.
var _ = json.Marshal
