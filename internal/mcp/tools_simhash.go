package mcp

import (
	"context"
	"encoding/json"

	"github.com/mark3labs/mcp-go/mcp"
)

func findSimilarTool() mcp.Tool {
	return mcp.NewTool("find_similar",
		mcp.WithDescription("Find near-duplicate code blocks in the indexed codebase using simhash fingerprinting. Reads a file, chunks it, and compares each chunk against the index."),
		mcp.WithString("path", mcp.Required(), mcp.Description("Absolute path to the file to check for duplicates")),
		mcp.WithString("project", mcp.Description("Project name to scope the search (optional)")),
		mcp.WithNumber("threshold", mcp.Description("Similarity threshold 0.0–1.0 (default 0.85)"), mcp.DefaultNumber(0.85)),
		mcp.WithNumber("top_k", mcp.Description("Max matches per chunk (default 5)"), mcp.DefaultNumber(5)),
	)
}

type findSimilarArgs struct {
	Path      string  `json:"path"`
	Project   string  `json:"project"`
	Threshold float64 `json:"threshold"`
	TopK      int     `json:"top_k"`
}

func (s *Server) findSimilar(ctx context.Context, _ mcp.CallToolRequest, args findSimilarArgs) (*mcp.CallToolResult, error) {
	if args.Path == "" {
		return toolError(errReasonInvalidArgument, "path is required", "provide an absolute file path"), nil
	}

	if s.indexer == nil {
		return toolError(errReasonNotConfigured, "indexer is not configured", "add a project and run rebuild_index first"), nil
	}

	matches, err := s.indexer.FindSimilar(ctx, args.Path, args.Project, args.Threshold, args.TopK)
	if err != nil {
		return toolError(errReasonNotFound, err.Error(), ""), nil
	}

	if len(matches) == 0 {
		return mcp.NewToolResultText("[]"), nil
	}

	data, err := json.Marshal(matches)
	if err != nil {
		return toolError(errReasonSearchFailed, "marshal find_similar result", ""), nil
	}
	return mcp.NewToolResultText(string(data)), nil
}
