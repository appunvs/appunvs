package hub

import (
	"encoding/json"
	"errors"
	"sync"
	"time"

	"github.com/gorilla/websocket"
	"go.uber.org/zap"

	"github.com/appunvs/appunvs/relay/internal/pb"
)

// Timings used by readLoop/writeLoop.
const (
	WriteWait      = 10 * time.Second
	PongWait       = 60 * time.Second
	PingPeriod     = 30 * time.Second
	MaxMessageSize = 1 << 20 // 1 MiB

	sendBuffer = 64
)

// ReadHandler is invoked for every decoded inbound Message.
type ReadHandler func(c *Conn, msg *pb.Message)

// Conn wraps a single websocket connection.
type Conn struct {
	ws        *websocket.Conn
	send      chan *pb.Message
	closeOnce sync.Once

	DeviceID  string
	UserID    string
	Namespace string
	Role      pb.Role

	log *zap.Logger
}

// NewConn builds a Conn around an already-upgraded websocket.
// Send buffer is fixed to keep broadcast fanout cheap.
func NewConn(ws *websocket.Conn, deviceID, userID, namespace string, role pb.Role, log *zap.Logger) *Conn {
	return &Conn{
		ws:        ws,
		send:      make(chan *pb.Message, sendBuffer),
		DeviceID:  deviceID,
		UserID:    userID,
		Namespace: namespace,
		Role:      role,
		log:       log,
	}
}

// Send enqueues msg onto the outbound channel.  Returns false if the buffer
// is full (caller typically evicts the Conn in that case).
func (c *Conn) Send(msg *pb.Message) bool {
	select {
	case c.send <- msg:
		return true
	default:
		return false
	}
}

// SendBlocking blocks until the message is accepted or the send channel is
// closed.  Used during the catch-up replay where dropping would break
// ordering guarantees.
func (c *Conn) SendBlocking(msg *pb.Message) error {
	defer func() {
		// If send is closed (conn was evicted mid-catchup) a panic bubbles;
		// recover so the handler can bail out cleanly.
		_ = recover()
	}()
	c.send <- msg
	return nil
}

// ReadLoop reads frames, decodes them as pb.Message, and invokes handler.
// It returns once the socket errors or closes.
func (c *Conn) ReadLoop(handler ReadHandler) {
	c.ws.SetReadLimit(MaxMessageSize)
	_ = c.ws.SetReadDeadline(time.Now().Add(PongWait))
	c.ws.SetPongHandler(func(string) error {
		_ = c.ws.SetReadDeadline(time.Now().Add(PongWait))
		return nil
	})
	for {
		_, data, err := c.ws.ReadMessage()
		if err != nil {
			if !websocket.IsCloseError(err, websocket.CloseNormalClosure, websocket.CloseGoingAway) {
				c.log.Info("ws read error",
					zap.String("device_id", c.DeviceID),
					zap.String("namespace", c.Namespace),
					zap.Error(err))
			}
			return
		}
		var msg pb.Message
		if err := json.Unmarshal(data, &msg); err != nil {
			c.log.Info("ws decode error",
				zap.String("device_id", c.DeviceID),
				zap.Error(err))
			continue
		}
		handler(c, &msg)
	}
}

// WriteLoop pumps the send channel onto the socket and issues keepalive pings.
// Exits when send is closed or a write errors.
func (c *Conn) WriteLoop() {
	ticker := time.NewTicker(PingPeriod)
	defer ticker.Stop()
	for {
		select {
		case msg, ok := <-c.send:
			_ = c.ws.SetWriteDeadline(time.Now().Add(WriteWait))
			if !ok {
				_ = c.ws.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}
			body, err := json.Marshal(msg)
			if err != nil {
				c.log.Warn("ws marshal error", zap.Error(err))
				continue
			}
			if err := c.ws.WriteMessage(websocket.TextMessage, body); err != nil {
				c.log.Info("ws write error",
					zap.String("device_id", c.DeviceID),
					zap.Error(err))
				return
			}
		case <-ticker.C:
			_ = c.ws.SetWriteDeadline(time.Now().Add(WriteWait))
			if err := c.ws.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		}
	}
}

// Close closes the underlying socket.  Safe to call more than once.
func (c *Conn) Close() error {
	err := c.ws.Close()
	if errors.Is(err, websocket.ErrCloseSent) {
		return nil
	}
	return err
}

// closeSend is internal to the hub; it closes send exactly once so WriteLoop
// can drain and exit.
func (c *Conn) closeSend() {
	c.closeOnce.Do(func() { close(c.send) })
}
