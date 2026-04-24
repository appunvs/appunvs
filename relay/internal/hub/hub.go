// Package hub is the in-memory fanout layer.  It tracks one set of
// websocket Conns per namespace and broadcasts Messages to them.
//
// The Hub is safe for concurrent use; callers may Register/Unregister and
// Broadcast from any goroutine.  Broadcasts never block: if a Conn's send
// buffer is full we drop that Conn rather than stalling fanout.
package hub

import (
	"sync"

	"go.uber.org/zap"

	"github.com/appunvs/appunvs/relay/internal/pb"
)

// RoleFilter selects which Conns receive a broadcast.
type RoleFilter int

const (
	// AllRoles delivers to every Conn in the namespace.
	AllRoles RoleFilter = iota
	// ProvidersOnly delivers only to Conns whose Role IsProvider.
	ProvidersOnly
	// ConnectorsOnly delivers only to Conns whose Role IsConnector.
	ConnectorsOnly
)

// Hub tracks namespace -> {Conn} membership.
type Hub struct {
	log *zap.Logger

	mu    sync.RWMutex
	nsMap map[string]map[*Conn]struct{}
}

// New constructs an empty Hub.
func New(log *zap.Logger) *Hub {
	return &Hub{log: log, nsMap: make(map[string]map[*Conn]struct{})}
}

// Register adds c to its namespace's membership set.
func (h *Hub) Register(c *Conn) {
	h.mu.Lock()
	defer h.mu.Unlock()
	set, ok := h.nsMap[c.Namespace]
	if !ok {
		set = make(map[*Conn]struct{})
		h.nsMap[c.Namespace] = set
	}
	set[c] = struct{}{}
	h.log.Info("ws accept",
		zap.String("device_id", c.DeviceID),
		zap.String("user_id", c.UserID),
		zap.String("namespace", c.Namespace),
		zap.String("role", c.Role.String()),
		zap.Int("ns_size", len(set)))
}

// Unregister removes c and closes its send channel.
// Safe to call multiple times.
func (h *Hub) Unregister(c *Conn) {
	h.mu.Lock()
	defer h.mu.Unlock()
	set, ok := h.nsMap[c.Namespace]
	if !ok {
		return
	}
	if _, present := set[c]; !present {
		return
	}
	delete(set, c)
	if len(set) == 0 {
		delete(h.nsMap, c.Namespace)
	}
	c.closeSend()
	h.log.Info("ws close",
		zap.String("device_id", c.DeviceID),
		zap.String("user_id", c.UserID),
		zap.String("namespace", c.Namespace))
}

// Broadcast sends msg to every Conn in its namespace that matches filter.
// The origin Conn (if non-nil) is excluded so senders don't echo.
func (h *Hub) Broadcast(msg *pb.Message, filter RoleFilter, origin *Conn) {
	h.mu.RLock()
	set := h.nsMap[msg.Namespace]
	targets := make([]*Conn, 0, len(set))
	for c := range set {
		if c == origin {
			continue
		}
		if !roleMatches(c.Role, filter) {
			continue
		}
		targets = append(targets, c)
	}
	h.mu.RUnlock()

	for _, c := range targets {
		select {
		case c.send <- msg:
		default:
			h.log.Warn("broadcast drop: send buffer full",
				zap.String("device_id", c.DeviceID),
				zap.String("user_id", c.UserID),
				zap.String("namespace", c.Namespace),
				zap.Int64("seq", msg.Seq))
			// Slow consumer: evict so the remaining broadcast stays cheap.
			go h.Unregister(c)
		}
	}
}

func roleMatches(r pb.Role, f RoleFilter) bool {
	switch f {
	case ProvidersOnly:
		return r.IsProvider()
	case ConnectorsOnly:
		return r.IsConnector()
	default:
		return true
	}
}
