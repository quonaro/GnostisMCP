package mcp

import (
	"context"
	"encoding/json"

	"github.com/mark3labs/mcp-go/mcp"
)

func checkIndexCoverageTool() mcp.Tool {
	return mcp.NewTool("check_index_coverage",
		mcp.WithDescription("Check whether files are indexed and up to date"),
		mcp.WithArray("paths", mcp.Required(), mcp.Description("Absolute file paths to check")),
		mcp.WithNumber("top_k", mcp.Description("Max results"), mcp.DefaultNumber(100)),
	)
}

type checkCoverageArgs struct {
	Paths []string `json:"paths"`
	TopK  int      `json:"top_k,omitempty"`
}

func (s *Server) checkIndexCoverage(ctx context.Context, _ mcp.CallToolRequest, args checkCoverageArgs) (*mcp.CallToolResult, error) {
	if len(args.Paths) == 0 {
		return toolError(errReasonInvalidArgument, "paths is required", "provide at least one absolute file path"), nil
	}

	if s.indexer == nil {
		return toolError(errReasonNotConfigured, "indexer is not configured", "add a project and run rebuild_index first"), nil
	}

	statuses := s.indexer.CheckCoverage(ctx, args.Paths)

	topK := args.TopK
	if topK <= 0 || topK > len(statuses) {
		topK = len(statuses)
	}
	if topK > 100 {
		topK = 100
	}
	statuses = statuses[:topK]

	data, err := json.Marshal(statuses)
	if err != nil {
		return toolError(errReasonSearchFailed, "marshal coverage result", ""), nil
	}
	return mcp.NewToolResultText(string(data)), nil
}

func detectChangesTool() mcp.Tool {
	return mcp.NewTool("detect_changes",
		mcp.WithDescription("List files that are new, modified, or deleted since the last index"),
		mcp.WithString("project", mcp.Required(), mcp.Description("Project name from list_projects")),
	)
}

type detectChangesArgs struct {
	Project string `json:"project"`
}

func (s *Server) detectChanges(ctx context.Context, _ mcp.CallToolRequest, args detectChangesArgs) (*mcp.CallToolResult, error) {
	if args.Project == "" {
		return toolError(errReasonInvalidArgument, "project is required", "use list_projects to see available projects"), nil
	}

	if s.indexer == nil {
		return toolError(errReasonNotConfigured, "indexer is not configured", "add a project and run rebuild_index first"), nil
	}

	changes, err := s.indexer.DetectChanges(ctx, args.Project)
	if err != nil {
		return toolError(errReasonNotFound, err.Error(), "use list_projects to verify the project name"), nil
	}

	data, err := json.Marshal(changes)
	if err != nil {
		return toolError(errReasonSearchFailed, "marshal changes result", ""), nil
	}
	return mcp.NewToolResultText(string(data)), nil
}
