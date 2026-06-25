// Package socket implements the client-facing WebSocket protocol described in
// the Lanyard README (Opcodes 0-4, optional zlib_json compression). It mirrors
// Lanyard.SocketHandler.
package socket

import (
	"bytes"
	"compress/zlib"
	"context"
	"encoding/json"
	"sync"

	"github.com/coder/websocket"

	"zruvix/internal/metrics"
	"zruvix/internal/presence"
)

const heartbeatInterval = 30000

// Conn is a single client WebSocket connection. It implements
// presence.Subscriber so presences can fan updates out to it.
type Conn struct {
	ws          *websocket.Conn
	compression string // "zlib" or "json"

	sendCh chan presence.SocketMessage

	mu            sync.Mutex
	closed        bool
	global        bool
	subscribedIDs map[string]struct{}

	cleanupOnce sync.Once
}

// Handle runs the lifecycle of a client connection: it sends Hello, pumps
// outbound messages, reads client opcodes, and cleans up on disconnect.
func Handle(ctx context.Context, ws *websocket.Conn, compression string) {
	c := &Conn{
		ws:            ws,
		compression:   compression,
		sendCh:        make(chan presence.SocketMessage, 256),
		subscribedIDs: make(map[string]struct{}),
	}

	metrics.ConnectedSessions.Inc()
	go c.writeLoop(ctx)

	// Opcode 1: Hello.
	c.SendEvent(presence.SocketMessage{Op: 1, D: map[string]any{"heartbeat_interval": heartbeatInterval}})

	c.readLoop(ctx)
	c.cleanup()
}

// SendEvent queues a message for delivery. It is safe to call concurrently and
// never blocks the caller (the presence fan-out path).
func (c *Conn) SendEvent(msg presence.SocketMessage) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.closed {
		return
	}
	select {
	case c.sendCh <- msg:
	default:
		// Slow consumer; drop rather than stall the fan-out.
	}
}

func (c *Conn) writeLoop(ctx context.Context) {
	for msg := range c.sendCh {
		typ, data, err := c.encode(msg)
		if err != nil {
			continue
		}
		metrics.MessagesOutbound.Inc()
		if err := c.ws.Write(ctx, typ, data); err != nil {
			return
		}
	}
}

func (c *Conn) encode(msg presence.SocketMessage) (websocket.MessageType, []byte, error) {
	raw, err := json.Marshal(msg)
	if err != nil {
		return 0, nil, err
	}
	if c.compression == "zlib" {
		var buf bytes.Buffer
		zw := zlib.NewWriter(&buf)
		if _, err := zw.Write(raw); err != nil {
			zw.Close()
			return 0, nil, err
		}
		zw.Close()
		return websocket.MessageBinary, buf.Bytes(), nil
	}
	return websocket.MessageText, raw, nil
}

func (c *Conn) readLoop(ctx context.Context) {
	for {
		_, data, err := c.ws.Read(ctx)
		if err != nil {
			return
		}
		metrics.MessagesInbound.Inc()

		var m map[string]any
		if json.Unmarshal(data, &m) != nil {
			c.closeWith(4006, "invalid_payload")
			return
		}

		opf, ok := m["op"].(float64)
		if !ok {
			c.closeWith(4004, "unknown_opcode")
			return
		}

		switch int(opf) {
		case 2: // Initialize
			if !c.handleInit(m) {
				return
			}
		case 3: // Heartbeat — no-op
		case 4: // Unsubscribe
			c.handleUnsubscribe(m)
		default:
			c.closeWith(4004, "unknown_opcode")
			return
		}
	}
}

// handleInit processes Opcode 2. It returns false if the connection should
// close (the close frame has already been sent).
func (c *Conn) handleInit(m map[string]any) bool {
	d, ok := m["d"].(map[string]any)
	if !ok || len(d) == 0 {
		c.closeWith(4005, "requires_data_object")
		return false
	}

	var initState any

	switch {
	case d["subscribe_to_ids"] != nil:
		ids := toStringSlice(d["subscribe_to_ids"])
		c.trackIDs(ids)
		initState = presence.SubscribeToIDsAndBuild(ids, c)

	case d["subscribe_to_id"] != nil:
		id, _ := d["subscribe_to_id"].(string)
		if p := presence.SubscribeToID(id, c); p != nil {
			c.trackIDs([]string{id})
			initState = p
		} else {
			initState = map[string]any{}
		}

	case d["subscribe_to_all"] == true:
		c.mu.Lock()
		c.global = true
		c.mu.Unlock()
		presence.AddGlobalSubscriber(c)
		ids := presence.Reg.IDs()
		c.trackIDs(ids)
		initState = presence.SubscribeToIDsAndBuild(ids, c)

	default:
		c.closeWith(4006, "invalid_payload")
		return false
	}

	c.SendEvent(presence.SocketMessage{Op: 0, T: "INIT_STATE", D: initState})
	return true
}

func (c *Conn) handleUnsubscribe(m map[string]any) {
	d, ok := m["d"].(map[string]any)
	if !ok {
		return
	}
	if id, ok := d["unsubscribe_from_id"].(string); ok {
		presence.Unsubscribe(id, c)
		c.mu.Lock()
		delete(c.subscribedIDs, id)
		c.mu.Unlock()
	}
}

func (c *Conn) trackIDs(ids []string) {
	c.mu.Lock()
	for _, id := range ids {
		c.subscribedIDs[id] = struct{}{}
	}
	c.mu.Unlock()
}

func (c *Conn) closeWith(code int, reason string) {
	c.ws.Close(websocket.StatusCode(code), reason)
}

func (c *Conn) cleanup() {
	c.cleanupOnce.Do(func() {
		c.mu.Lock()
		c.closed = true
		global := c.global
		ids := make([]string, 0, len(c.subscribedIDs))
		for id := range c.subscribedIDs {
			ids = append(ids, id)
		}
		close(c.sendCh)
		c.mu.Unlock()

		if global {
			presence.RemoveGlobalSubscriber(c)
		}
		for _, id := range ids {
			presence.Unsubscribe(id, c)
		}
		metrics.ConnectedSessions.Dec()
	})
}

func toStringSlice(v any) []string {
	list, ok := v.([]any)
	if !ok {
		return nil
	}
	out := make([]string, 0, len(list))
	for _, item := range list {
		if s, ok := item.(string); ok {
			out = append(out, s)
		}
	}
	return out
}
