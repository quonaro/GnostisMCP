package main

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"strings"
)

// runStdioProxy reads JSON-RPC messages from stdin and forwards them to the
// primary gnostis instance's Streamable HTTP MCP endpoint, writing responses
// back to stdout. This allows multiple editor windows to share a single
// gnostis process while keeping the stdio MCP config format.
func runStdioProxy(mcpURL string) error {
	slog.Info("starting stdio proxy to primary gnostis", "url", mcpURL)

	scanner := bufio.NewScanner(os.Stdin)
	scanner.Buffer(make([]byte, 0, 64*1024), 10*1024*1024)

	client := &http.Client{}
	var sessionID string

	for scanner.Scan() {
		line := bytes.TrimSpace(scanner.Bytes())
		if len(line) == 0 {
			continue
		}

		resp, err := forwardMessage(client, mcpURL, line, sessionID)
		if err != nil {
			fmt.Fprintf(os.Stderr, "proxy: %v\n", err)
			continue
		}

		if sid := resp.Header.Get("Mcp-Session-Id"); sid != "" {
			sessionID = sid
		}

		if err := writeResponse(resp, os.Stdout); err != nil {
			fmt.Fprintf(os.Stderr, "proxy: write response: %v\n", err)
		}
		if err := resp.Body.Close(); err != nil {
			fmt.Fprintf(os.Stderr, "proxy: close response body: %v\n", err)
		}
	}
	return scanner.Err()
}

func forwardMessage(client *http.Client, url string, body []byte, sessionID string) (*http.Response, error) {
	req, err := http.NewRequest(http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	if sessionID != "" {
		req.Header.Set("Mcp-Session-Id", sessionID)
	}
	return client.Do(req)
}

func writeResponse(resp *http.Response, w io.Writer) error {
	if resp.StatusCode == http.StatusAccepted {
		return nil
	}

	contentType := resp.Header.Get("Content-Type")
	if strings.Contains(contentType, "text/event-stream") {
		return writeSSEEvents(resp.Body, w)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("read body: %w", err)
	}
	if _, err := w.Write(body); err != nil {
		return err
	}
	if !bytes.HasSuffix(body, []byte("\n")) {
		if _, err := w.Write([]byte("\n")); err != nil {
			return err
		}
	}
	return nil
}

// writeSSEEvents parses an SSE stream and writes each data payload as a
// newline-delimited JSON message to w.
func writeSSEEvents(r io.Reader, w io.Writer) error {
	scanner := bufio.NewScanner(r)
	scanner.Buffer(make([]byte, 0, 64*1024), 10*1024*1024)
	for scanner.Scan() {
		line := scanner.Text()
		if !strings.HasPrefix(line, "data: ") {
			continue
		}
		data := strings.TrimPrefix(line, "data: ")
		if data == "" || data == "[DONE]" {
			continue
		}
		if _, err := fmt.Fprintf(w, "%s\n", data); err != nil {
			return err
		}
	}
	return scanner.Err()
}
