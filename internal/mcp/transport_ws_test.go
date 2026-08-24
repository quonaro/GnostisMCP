package mcp

import (
	"context"
	"encoding/json"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/coder/websocket"
)

func TestWSHandler_ToolCallRoundTrip(t *testing.T) {
	srv := New(&mockSearcher{}, nil, &mockIndexer{}, nil)
	handler := srv.WSHandler(nil)

	ts := httptest.NewServer(handler)
	defer ts.Close()

	wsURL := "ws" + strings.TrimPrefix(ts.URL, "http") + "/ws"

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	conn, _, err := websocket.Dial(ctx, wsURL, nil)
	if err != nil {
		t.Fatalf("dial: %v", err)
	}
	defer func() {
		_ = conn.Close(websocket.StatusNormalClosure, "")
	}()

	// Send a tools/call request for "list_projects"
	req := map[string]any{
		"jsonrpc": "2.0",
		"id":      1,
		"method":  "tools/call",
		"params": map[string]any{
			"name":      "list_projects",
			"arguments": map[string]any{},
		},
	}
	data, _ := json.Marshal(req)
	if err := conn.Write(ctx, websocket.MessageText, data); err != nil {
		t.Fatalf("write: %v", err)
	}

	// Read response
	_, respData, err := conn.Read(ctx)
	if err != nil {
		t.Fatalf("read: %v", err)
	}

	var resp struct {
		JSONRPC string          `json:"jsonrpc"`
		ID      int             `json:"id"`
		Result  json.RawMessage `json:"result"`
		Error   *struct {
			Code    int    `json:"code"`
			Message string `json:"message"`
		} `json:"error"`
	}
	if err := json.Unmarshal(respData, &resp); err != nil {
		t.Fatalf("unmarshal response: %v\nraw: %s", err, respData)
	}
	if resp.Error != nil {
		t.Fatalf("unexpected error: %s", resp.Error.Message)
	}
	if resp.ID != 1 {
		t.Errorf("expected id=1, got %d", resp.ID)
	}
	if resp.Result == nil {
		t.Fatal("expected non-nil result")
	}
}

func TestWSHandler_DashboardMethod(t *testing.T) {
	srv := New(&mockSearcher{}, nil, &mockIndexer{}, nil)

	// Register a custom dashboard handler
	called := false
	handlers := map[string]DashboardHandler{
		"test_echo": func(_ context.Context, params json.RawMessage) (any, error) {
			called = true
			var p struct {
				Msg string `json:"msg"`
			}
			_ = json.Unmarshal(params, &p)
			return map[string]string{"echo": p.Msg}, nil
		},
	}

	handler := srv.WSHandler(handlers)
	ts := httptest.NewServer(handler)
	defer ts.Close()

	wsURL := "ws" + strings.TrimPrefix(ts.URL, "http") + "/ws"

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	conn, _, err := websocket.Dial(ctx, wsURL, nil)
	if err != nil {
		t.Fatalf("dial: %v", err)
	}
	defer func() {
		_ = conn.Close(websocket.StatusNormalClosure, "")
	}()

	// Send a dashboard-specific method call
	req := map[string]any{
		"jsonrpc": "2.0",
		"id":      2,
		"method":  "test_echo",
		"params":  map[string]string{"msg": "hello"},
	}
	data, _ := json.Marshal(req)
	if err := conn.Write(ctx, websocket.MessageText, data); err != nil {
		t.Fatalf("write: %v", err)
	}

	_, respData, err := conn.Read(ctx)
	if err != nil {
		t.Fatalf("read: %v", err)
	}

	var resp struct {
		JSONRPC string         `json:"jsonrpc"`
		ID      int            `json:"id"`
		Result  map[string]any `json:"result"`
		Error   *struct {
			Code    int    `json:"code"`
			Message string `json:"message"`
		} `json:"error"`
	}
	if err := json.Unmarshal(respData, &resp); err != nil {
		t.Fatalf("unmarshal response: %v\nraw: %s", err, respData)
	}
	if resp.Error != nil {
		t.Fatalf("unexpected error: %s", resp.Error.Message)
	}
	if !called {
		t.Fatal("dashboard handler was not called")
	}
	if resp.Result["echo"] != "hello" {
		t.Errorf("expected echo=hello, got %v", resp.Result["echo"])
	}
}

func TestWSHandler_Notification(t *testing.T) {
	srv := New(&mockSearcher{}, nil, &mockIndexer{}, nil)
	handler := srv.WSHandler(nil)

	ts := httptest.NewServer(handler)
	defer ts.Close()

	wsURL := "ws" + strings.TrimPrefix(ts.URL, "http") + "/ws"

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	conn, _, err := websocket.Dial(ctx, wsURL, nil)
	if err != nil {
		t.Fatalf("dial: %v", err)
	}
	defer func() {
		_ = conn.Close(websocket.StatusNormalClosure, "")
	}()

	// Broadcast a notification — session is always initialized so it should arrive.
	go func() {
		time.Sleep(100 * time.Millisecond)
		srv.SendNotificationToAll("gnostis/test", map[string]any{"value": 42})
	}()

	// Read messages until we find the notification
	for {
		_, msgData, err := conn.Read(ctx)
		if err != nil {
			t.Fatalf("read notification: %v", err)
		}

		var msg struct {
			Method string         `json:"method"`
			Params map[string]any `json:"params"`
		}
		if err := json.Unmarshal(msgData, &msg); err != nil {
			continue
		}
		if msg.Method == "gnostis/test" {
			if msg.Params["value"] != float64(42) {
				t.Errorf("expected value=42, got %v", msg.Params["value"])
			}
			return
		}
	}
}
