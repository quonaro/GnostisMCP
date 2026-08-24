package web

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
)

// Server is the HTTP dashboard server. It serves the SPA, the MCP streamable
// HTTP endpoint, and the WebSocket endpoint for the dashboard.
type Server struct {
	mcpHandler http.Handler
	wsHandler  http.Handler
	mux        *http.ServeMux
}

// New creates a new web server. If mcpHandler is non-nil it is mounted at /mcp
// to serve MCP clients over the Streamable HTTP transport. If wsHandler is
// non-nil it is mounted at /ws to serve the dashboard over WebSocket.
func New(mcpHandler http.Handler, wsHandler http.Handler) *Server {
	s := &Server{
		mcpHandler: mcpHandler,
		wsHandler:  wsHandler,
		mux:        http.NewServeMux(),
	}
	s.registerRoutes()
	return s
}

func (s *Server) registerRoutes() {
	if s.mcpHandler != nil {
		s.mux.Handle("/mcp", s.mcpHandler)
	}
	if s.wsHandler != nil {
		s.mux.Handle("/ws", s.wsHandler)
	}
	s.mux.Handle("/", s.handleSPA())
}

// Start launches the HTTP server. It blocks until ctx is cancelled.
func (s *Server) Start(ctx context.Context, port int) error {
	addr := fmt.Sprintf(":%d", port)
	srv := &http.Server{
		Addr:    addr,
		Handler: s.mux,
	}

	go func() {
		<-ctx.Done()
		slog.InfoContext(ctx, "shutting down web server")
		_ = srv.Shutdown(context.WithoutCancel(ctx))
	}()

	slog.InfoContext(ctx, "starting web server", "addr", addr)
	if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		return fmt.Errorf("web server: %w", err)
	}
	return nil
}
