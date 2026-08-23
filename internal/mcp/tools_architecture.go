package mcp

import (
	"context"
	"encoding/json"

	"github.com/mark3labs/mcp-go/mcp"
)

func getArchitectureTool() mcp.Tool {
	return mcp.NewTool("get_architecture",
		mcp.WithDescription("One-call structural overview of a project: languages, packages, entry points, hotspots, symbol counts, recently changed files"),
		mcp.WithString("project", mcp.Required(), mcp.Description("Project name from list_projects")),
	)
}

type getArchitectureArgs struct {
	Project string `json:"project"`
}

func (s *Server) getArchitecture(ctx context.Context, _ mcp.CallToolRequest, args getArchitectureArgs) (*mcp.CallToolResult, error) {
	if args.Project == "" {
		return toolError(errReasonInvalidArgument, "project is required", "use list_projects to see available projects"), nil
	}

	if s.indexer == nil {
		return toolError(errReasonNotConfigured, "indexer is not configured", "add a project and run rebuild_index first"), nil
	}

	arch, err := s.indexer.Architecture(ctx, args.Project)
	if err != nil {
		return toolError(errReasonNotFound, err.Error(), "use list_projects to verify the project name"), nil
	}

	data, err := json.Marshal(arch)
	if err != nil {
		return toolError(errReasonSearchFailed, "marshal architecture result", ""), nil
	}
	return mcp.NewToolResultText(string(data)), nil
}
