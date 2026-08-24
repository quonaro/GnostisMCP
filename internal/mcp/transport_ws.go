package mcp

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"sync"

	"github.com/coder/websocket"
	"github.com/mark3labs/mcp-go/mcp"
	mcpserver "github.com/mark3labs/mcp-go/server"
)

// wsSession implements mcpserver.ClientSession for a single WebSocket connection.
type wsSession struct {
	id          string
	initialized bool
	notifyCh    chan mcp.JSONRPCNotification
}

func newWsSession(id string) *wsSession {
	return &wsSession{
		id:          id,
		notifyCh:    make(chan mcp.JSONRPCNotification, 64),
		initialized: true,
	}
}

func (s *wsSession) Initialize()                                         { s.initialized = true }
func (s *wsSession) Initialized() bool                                   { return true }
func (s *wsSession) NotificationChannel() chan<- mcp.JSONRPCNotification { return s.notifyCh }
func (s *wsSession) SessionID() string                                   { return s.id }

// DashboardHandler handles dashboard-only JSON-RPC methods that are not MCP tools.
type DashboardHandler func(ctx context.Context, params json.RawMessage) (any, error)

// WSHandler returns an http.Handler that serves MCP over WebSocket.
// Dashboard-specific JSON-RPC methods are routed to the provided handlers map
// (method name → handler). Methods not in the map are forwarded to the MCP server.
func (s *Server) WSHandler(dashboardHandlers map[string]DashboardHandler) http.Handler {
	return &wsHandler{server: s, dashboardHandlers: dashboardHandlers}
}

type wsHandler struct {
	server            *Server
	dashboardHandlers map[string]DashboardHandler
}

func (h *wsHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	c, err := websocket.Accept(w, r, &websocket.AcceptOptions{
		InsecureSkipVerify: true,
	})
	if err != nil {
		slog.Error("ws accept", "error", err)
		return
	}
	defer func() {
		_ = c.Close(websocket.StatusNormalClosure, "")
	}()

	sessionID := fmt.Sprintf("ws-%p", c)
	session := newWsSession(sessionID)

	if err := h.server.server.RegisterSession(r.Context(), session); err != nil {
		slog.Error("ws register session", "error", err)
		return
	}
	defer h.server.server.UnregisterSession(context.Background(), sessionID)

	ctx, cancel := context.WithCancel(r.Context())
	defer cancel()

	// Goroutine: drain notifications from the session channel and write to WS.
	go h.forwardNotifications(ctx, c, session)

	// Read loop: read JSON-RPC messages from the WebSocket.
	h.readLoop(ctx, c, session)
}

func (h *wsHandler) forwardNotifications(ctx context.Context, c *websocket.Conn, session *wsSession) {
	for {
		select {
		case <-ctx.Done():
			return
		case notif, ok := <-session.notifyCh:
			if !ok {
				return
			}
			data, err := json.Marshal(notif)
			if err != nil {
				slog.Error("ws marshal notification", "error", err)
				continue
			}
			if err := c.Write(ctx, websocket.MessageText, data); err != nil {
				slog.Error("ws write notification", "error", err)
				return
			}
		}
	}
}

func (h *wsHandler) readLoop(ctx context.Context, c *websocket.Conn, session *wsSession) {
	var writeMu sync.Mutex

	for {
		_, data, err := c.Read(ctx)
		if err != nil {
			return
		}

		// Parse the base message to determine the method.
		var base struct {
			Method string          `json:"method"`
			ID     json.RawMessage `json:"id,omitempty"`
		}
		if err := json.Unmarshal(data, &base); err != nil {
			h.writeError(ctx, c, &writeMu, nil, mcp.PARSE_ERROR, "Parse error")
			continue
		}

		// Route dashboard-specific methods (no "id" means notification; with "id" means request).
		if handler, ok := h.dashboardHandlers[base.Method]; ok {
			var params json.RawMessage
			var raw struct {
				Params json.RawMessage `json:"params"`
			}
			_ = json.Unmarshal(data, &raw)
			params = raw.Params

			result, err := handler(ctx, params)
			if err != nil {
				if len(base.ID) > 0 {
					h.writeError(ctx, c, &writeMu, base.ID, mcp.INTERNAL_ERROR, err.Error())
				}
				continue
			}
			if len(base.ID) > 0 {
				h.writeResult(ctx, c, &writeMu, base.ID, result)
			}
			continue
		}

		// Forward to MCP server for standard MCP method handling.
		msgCtx := h.server.server.WithContext(ctx, session)
		response := h.server.server.HandleMessage(msgCtx, data)
		if response != nil {
			respData, err := json.Marshal(response)
			if err != nil {
				slog.Error("ws marshal response", "error", err)
				continue
			}
			writeMu.Lock()
			if err := c.Write(ctx, websocket.MessageText, respData); err != nil {
				writeMu.Unlock()
				slog.Error("ws write response", "error", err)
				return
			}
			writeMu.Unlock()
		}
	}
}

func (h *wsHandler) writeResult(ctx context.Context, c *websocket.Conn, mu *sync.Mutex, id json.RawMessage, result any) {
	resp := map[string]any{
		"jsonrpc": "2.0",
		"id":      id,
		"result":  result,
	}
	data, err := json.Marshal(resp)
	if err != nil {
		slog.Error("ws marshal result", "error", err)
		return
	}
	mu.Lock()
	defer mu.Unlock()
	if err := c.Write(ctx, websocket.MessageText, data); err != nil {
		slog.Error("ws write result", "error", err)
	}
}

func (h *wsHandler) writeError(ctx context.Context, c *websocket.Conn, mu *sync.Mutex, id json.RawMessage, code int, message string) {
	resp := map[string]any{
		"jsonrpc": "2.0",
		"error": map[string]any{
			"code":    code,
			"message": message,
		},
	}
	if id != nil {
		resp["id"] = id
	}
	data, err := json.Marshal(resp)
	if err != nil {
		slog.Error("ws marshal error", "error", err)
		return
	}
	mu.Lock()
	defer mu.Unlock()
	if err := c.Write(ctx, websocket.MessageText, data); err != nil {
		slog.Error("ws write error", "error", err)
	}
}

// SendNotificationToAll sends a JSON-RPC notification to all connected WebSocket sessions.
// This is used by the push-notification ticker (Step 4) to broadcast status updates.
func (s *Server) SendNotificationToAll(method string, params map[string]any) {
	s.server.SendNotificationToAllClients(method, params)
}

// MCPServer exposes the underlying mcp-go server for the notification ticker.
func (s *Server) MCPServer() *mcpserver.MCPServer { return s.server }
