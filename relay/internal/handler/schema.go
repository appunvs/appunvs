// Package handler's schema.go wires the /schema/... HTTP surface: user
// self-service CRUD over app_tables and app_columns, plus a broadcast to
// the user's live devices on every mutation so clients can invalidate
// their local caches.
package handler

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"

	"github.com/appunvs/appunvs/relay/internal/auth"
	"github.com/appunvs/appunvs/relay/internal/hub"
	"github.com/appunvs/appunvs/relay/internal/pb"
	"github.com/appunvs/appunvs/relay/internal/sequencer"
	"github.com/appunvs/appunvs/relay/internal/store"
	"github.com/appunvs/appunvs/relay/internal/stream"
)

// SchemaMetaTable is the reserved logical table name used when broadcasting
// schema-change events to the user's connected devices.
const SchemaMetaTable = "_schema"

// SchemaDeps bundles the collaborators the /schema/... handlers need.
// The set is deliberately a superset of handler.Deps because a schema
// mutation both writes to the DB and broadcasts a message.
type SchemaDeps struct {
	Signer *auth.Signer
	Store  *store.Store
	Hub    *hub.Hub
	Seq    *sequencer.Seq
	Stream *stream.Store
	Log    *zap.Logger
}

type createTableBody struct {
	Name string `json:"name"`
}

type addColumnBody struct {
	Name     string `json:"name"`
	Type     string `json:"type"`
	Required bool   `json:"required"`
}

// tableView is the HTTP shape for a table (created_at as unix ms int).
type tableView struct {
	Name      string       `json:"name"`
	CreatedAt int64        `json:"created_at"`
	Columns   []columnView `json:"columns"`
}

type columnView struct {
	Name      string `json:"name"`
	Type      string `json:"type"`
	Required  bool   `json:"required"`
	CreatedAt int64  `json:"created_at"`
}

// RegisterSchemaRoutes registers POST/GET/DELETE /schema/tables and its
// nested /columns routes on r.  Kept as a single function so cmd/server
// and the integration tests can wire identical trees.
func RegisterSchemaRoutes(r gin.IRouter, d SchemaDeps) {
	r.POST("/schema/tables", CreateTable(d))
	r.GET("/schema/tables", ListTables(d))
	r.DELETE("/schema/tables/:name", DeleteTable(d))
	r.POST("/schema/tables/:name/columns", AddColumn(d))
	r.DELETE("/schema/tables/:name/columns/:column", DeleteColumn(d))
}

// CreateTable handles POST /schema/tables.
func CreateTable(d SchemaDeps) gin.HandlerFunc {
	return func(c *gin.Context) {
		userID, ok := requireSession(c, d.Signer)
		if !ok {
			return
		}
		var body createTableBody
		if err := c.ShouldBindJSON(&body); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid body"})
			return
		}
		t, err := d.Store.Schema().CreateTable(c.Request.Context(), userID, body.Name)
		if err != nil {
			writeSchemaErr(c, d.Log, "create table", err)
			return
		}
		payload, _ := json.Marshal(map[string]string{"name": t.Name})
		d.broadcastSchema(c.Request.Context(), userID, pb.OpTableCreate, payload)
		c.JSON(http.StatusOK, tableView{
			Name:      t.Name,
			CreatedAt: t.CreatedAt.UnixMilli(),
			Columns:   []columnView{},
		})
	}
}

// ListTables handles GET /schema/tables.
func ListTables(d SchemaDeps) gin.HandlerFunc {
	return func(c *gin.Context) {
		userID, ok := requireSession(c, d.Signer)
		if !ok {
			return
		}
		tables, err := d.Store.Schema().ListTables(c.Request.Context(), userID)
		if err != nil {
			d.Log.Error("schema: list tables", zap.Error(err))
			c.JSON(http.StatusInternalServerError, gin.H{"error": "internal"})
			return
		}
		out := make([]tableView, 0, len(tables))
		for _, t := range tables {
			out = append(out, toTableView(t))
		}
		c.JSON(http.StatusOK, out)
	}
}

// DeleteTable handles DELETE /schema/tables/:name.
func DeleteTable(d SchemaDeps) gin.HandlerFunc {
	return func(c *gin.Context) {
		userID, ok := requireSession(c, d.Signer)
		if !ok {
			return
		}
		name := c.Param("name")
		if err := d.Store.Schema().DeleteTable(c.Request.Context(), userID, name); err != nil {
			writeSchemaErr(c, d.Log, "delete table", err)
			return
		}
		payload, _ := json.Marshal(map[string]string{"name": name})
		d.broadcastSchema(c.Request.Context(), userID, pb.OpTableDelete, payload)
		c.Status(http.StatusNoContent)
	}
}

// AddColumn handles POST /schema/tables/:name/columns.
func AddColumn(d SchemaDeps) gin.HandlerFunc {
	return func(c *gin.Context) {
		userID, ok := requireSession(c, d.Signer)
		if !ok {
			return
		}
		tableName := c.Param("name")
		var body addColumnBody
		if err := c.ShouldBindJSON(&body); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid body"})
			return
		}
		col, err := d.Store.Schema().AddColumn(c.Request.Context(), userID, tableName, body.Name, body.Type, body.Required)
		if err != nil {
			writeSchemaErr(c, d.Log, "add column", err)
			return
		}
		payload, _ := json.Marshal(map[string]interface{}{
			"table":    tableName,
			"name":     col.Name,
			"type":     col.Type,
			"required": col.Required,
		})
		d.broadcastSchema(c.Request.Context(), userID, pb.OpColumnAdd, payload)
		c.JSON(http.StatusOK, columnView{
			Name:      col.Name,
			Type:      col.Type,
			Required:  col.Required,
			CreatedAt: col.CreatedAt.UnixMilli(),
		})
	}
}

// DeleteColumn handles DELETE /schema/tables/:name/columns/:column.
func DeleteColumn(d SchemaDeps) gin.HandlerFunc {
	return func(c *gin.Context) {
		userID, ok := requireSession(c, d.Signer)
		if !ok {
			return
		}
		tableName := c.Param("name")
		colName := c.Param("column")
		if err := d.Store.Schema().DeleteColumn(c.Request.Context(), userID, tableName, colName); err != nil {
			writeSchemaErr(c, d.Log, "delete column", err)
			return
		}
		payload, _ := json.Marshal(map[string]string{
			"table": tableName,
			"name":  colName,
		})
		d.broadcastSchema(c.Request.Context(), userID, pb.OpColumnDelete, payload)
		c.Status(http.StatusNoContent)
	}
}

// broadcastSchema assigns a sequence number, appends to the Redis Stream,
// and fanouts to all live sockets in the user's namespace.  Schema changes
// are authoritative, so the broadcast wears role=provider.  Failures are
// logged but do not fail the HTTP mutation — the DB change already
// committed.
func (d SchemaDeps) broadcastSchema(parent context.Context, userID string, op pb.Op, payload json.RawMessage) {
	ns := userID
	ctx, cancel := context.WithTimeout(parent, 3*time.Second)
	defer cancel()
	seq, err := d.Seq.Next(ctx, ns)
	if err != nil {
		d.Log.Error("schema: seq next failed", zap.String("user_id", userID), zap.Error(err))
		return
	}
	msg := &pb.Message{
		Seq:       seq,
		UserID:    userID,
		Namespace: ns,
		Role:      pb.RoleProvider,
		Op:        op,
		Table:     SchemaMetaTable,
		Payload:   append([]byte(nil), payload...),
		TS:        time.Now().UnixMilli(),
	}
	if err := d.Stream.Append(ctx, ns, msg); err != nil {
		d.Log.Error("schema: stream append failed", zap.String("user_id", userID), zap.Error(err))
		return
	}
	d.Hub.Broadcast(msg, hub.AllRoles, nil)
	d.Log.Info("schema broadcast",
		zap.String("user_id", userID),
		zap.String("op", op.String()),
		zap.Int64("seq", seq))
}

// writeSchemaErr maps store sentinels to HTTP statuses.  Validation errors
// (reserved/invalid) are 400; duplicates are 409; missing rows are 404;
// anything else is 500.
func writeSchemaErr(c *gin.Context, log *zap.Logger, op string, err error) {
	switch {
	case errors.Is(err, store.ErrReservedName),
		errors.Is(err, store.ErrInvalidName),
		errors.Is(err, store.ErrInvalidType):
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
	case errors.Is(err, store.ErrTableExists), errors.Is(err, store.ErrColumnExists):
		c.JSON(http.StatusConflict, gin.H{"error": err.Error()})
	case errors.Is(err, store.ErrTableNotFound), errors.Is(err, store.ErrColumnNotFound):
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
	default:
		log.Error("schema: "+op, zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal"})
	}
}

// toTableView projects a store.Table to its HTTP shape.
func toTableView(t store.Table) tableView {
	cols := make([]columnView, 0, len(t.Columns))
	for _, c := range t.Columns {
		cols = append(cols, columnView{
			Name:      c.Name,
			Type:      c.Type,
			Required:  c.Required,
			CreatedAt: c.CreatedAt.UnixMilli(),
		})
	}
	return tableView{
		Name:      t.Name,
		CreatedAt: t.CreatedAt.UnixMilli(),
		Columns:   cols,
	}
}
