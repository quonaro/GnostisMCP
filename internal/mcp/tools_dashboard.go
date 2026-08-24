package mcp

import (
	"context"
	"encoding/json"

	"github.com/mark3labs/mcp-go/mcp"
)

// --- graph_layout ---

func graphLayoutTool() mcp.Tool {
	return mcp.NewTool("graph_layout",
		mcp.WithDescription("Return a force-directed graph layout for the given project. Includes node positions, edges, and layout metadata."),
		mcp.WithString("project", mcp.Description("Project name (empty for all projects)")),
		mcp.WithBoolean("connected_only", mcp.Description("Exclude isolated nodes (default true)"), mcp.DefaultBool(true)),
		mcp.WithNumber("max_nodes", mcp.Description("Maximum nodes to render (default 800)"), mcp.DefaultNumber(800)),
	)
}

type graphLayoutArgs struct {
	Project       string `json:"project"`
	ConnectedOnly bool   `json:"connected_only"`
	MaxNodes      int    `json:"max_nodes"`
}

func (s *Server) graphLayout(ctx context.Context, _ mcp.CallToolRequest, args graphLayoutArgs) (*mcp.CallToolResult, error) {
	if s.indexer == nil {
		return toolError(errReasonNotConfigured, "indexer is not configured", "add a project and run rebuild_index first"), nil
	}

	result, err := s.indexer.GraphLayout(args.Project, args.ConnectedOnly, args.MaxNodes)
	if err != nil {
		return toolError(errReasonNotFound, err.Error(), ""), nil
	}

	data, err := json.Marshal(result)
	if err != nil {
		return toolError(errReasonSearchFailed, "marshal graph_layout result", ""), nil
	}
	return mcp.NewToolResultText(string(data)), nil
}

// --- memory_files ---

func memoryFilesTool() mcp.Tool {
	return mcp.NewTool("memory_files",
		mcp.WithDescription("List exported memory files with metadata (path, provider, title, date, size)."),
	)
}

type memoryFilesArgs struct{}

func (s *Server) memoryFiles(ctx context.Context, _ mcp.CallToolRequest, _ memoryFilesArgs) (*mcp.CallToolResult, error) {
	if s.indexer == nil {
		return toolError(errReasonNotConfigured, "indexer is not configured", ""), nil
	}

	files := s.indexer.MemoryFiles(ctx)
	if len(files) == 0 {
		return mcp.NewToolResultText("[]"), nil
	}

	data, err := json.Marshal(files)
	if err != nil {
		return toolError(errReasonSearchFailed, "marshal memory_files result", ""), nil
	}
	return mcp.NewToolResultText(string(data)), nil
}
