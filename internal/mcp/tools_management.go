package mcp

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"

	"github.com/mark3labs/mcp-go/mcp"

	"github.com/quonaro/gnostis/internal/jobs"
	"github.com/quonaro/gnostis/internal/memory"
	"github.com/quonaro/gnostis/internal/progress"
	"github.com/quonaro/gnostis/internal/stats"
)

type getIndexStatusArgs struct{}

type indexStatusResult struct {
	Projects       []string                 `json:"projects"`
	TotalChunks    int                      `json:"total_chunks"`
	Provider       string                   `json:"provider"`
	Model          string                   `json:"model"`
	Symbols        int                      `json:"symbols"`
	Progress       progress.State           `json:"progress"`
	ETA            string                   `json:"eta,omitempty"`
	ETASeconds     int64                    `json:"eta_seconds,omitempty"`
	ProjectStats   map[string]stats.Project `json:"project_stats"`
	MemoryStats    []memory.ProviderStat    `json:"memory_stats,omitempty"`
	MemoryProgress memory.ProgressState     `json:"memory_progress,omitempty"`
	Jobs           []jobs.Job               `json:"jobs,omitempty"`
}

func getIndexStatusTool() mcp.Tool {
	return mcp.NewTool("get_index_status",
		mcp.WithDescription("Return the current index status, project list, and progress"),
	)
}

func (s *Server) getIndexStatus(ctx context.Context, request mcp.CallToolRequest, args getIndexStatusArgs) (*mcp.CallToolResult, error) {
	slog.InfoContext(ctx, "mcp tool call", "tool", "get_index_status")
	if s.indexer == nil {
		return toolError(errReasonNotConfigured, "indexer is not configured", "check the Gnostis configuration"), nil
	}

	projects, chunks := s.indexer.Status()
	provider, model, symbols := s.indexer.Info()

	pstate, err := s.indexer.ProgressState()
	if err != nil {
		slog.ErrorContext(ctx, "get_index_status failed", "error", err)
		return toolError(errReasonSearchFailed, err.Error(), "try again later"), nil
	}

	pst, err := s.indexer.ProjectStats(ctx)
	if err != nil {
		slog.ErrorContext(ctx, "get_index_status failed", "error", err)
		return toolError(errReasonSearchFailed, err.Error(), "try again later"), nil
	}

	memStats := s.indexer.MemoryStats(ctx)
	memProgress := s.indexer.MemoryProgressState()
	jobList := s.indexer.Jobs()

	eta := pstate.ETA()
	result := indexStatusResult{
		Projects:       projects,
		TotalChunks:    chunks,
		Provider:       provider,
		Model:          model,
		Symbols:        symbols,
		Progress:       pstate,
		ProjectStats:   pst,
		MemoryStats:    memStats,
		MemoryProgress: memProgress,
		Jobs:           jobList,
	}
	if eta > 0 {
		result.ETA = eta.String()
		result.ETASeconds = int64(eta.Seconds())
	}

	data, err := json.Marshal(result)
	if err != nil {
		return toolError(errReasonSearchFailed, err.Error(), "internal error marshalling status"), nil
	}
	return mcp.NewToolResultText(string(data)), nil
}

type getIndexJobArgs struct {
	JobID string `json:"job_id"`
}

func getIndexJobTool() mcp.Tool {
	return mcp.NewTool("get_index_job",
		mcp.WithDescription("Return the status of a previously started rebuild job"),
		mcp.WithString("job_id", mcp.Required(), mcp.Description("Job ID returned by rebuild_project or rebuild_index")),
	)
}

func (s *Server) getIndexJob(ctx context.Context, request mcp.CallToolRequest, args getIndexJobArgs) (*mcp.CallToolResult, error) {
	slog.InfoContext(ctx, "mcp tool call", "tool", "get_index_job", "job_id", args.JobID)
	if s.indexer == nil {
		return toolError(errReasonNotConfigured, "indexer is not configured", "check the Gnostis configuration"), nil
	}
	if args.JobID == "" {
		return toolError(errReasonInvalidArgument, "job_id is required", "provide the job_id returned by rebuild_project/rebuild_index"), nil
	}

	pstate, err := s.indexer.ProgressState()
	if err != nil {
		slog.ErrorContext(ctx, "get_index_job failed", "error", err)
		return toolError(errReasonSearchFailed, err.Error(), "try again later"), nil
	}
	if pstate.JobID != args.JobID {
		return toolError(errReasonNotFound, fmt.Sprintf("job %q not found", args.JobID), "use get_index_status to see the current job"), nil
	}

	data, err := json.Marshal(pstate)
	if err != nil {
		return toolError(errReasonSearchFailed, err.Error(), "internal error marshalling job state"), nil
	}
	return mcp.NewToolResultText(string(data)), nil
}

type rebuildProjectArgs struct {
	Project string `json:"project"`
}

func rebuildProjectTool() mcp.Tool {
	return mcp.NewTool("rebuild_project",
		mcp.WithDescription("Rebuild the index for a single project"),
		mcp.WithString("project", mcp.Required(), mcp.Description("Project name")),
	)
}

func (s *Server) rebuildProject(ctx context.Context, request mcp.CallToolRequest, args rebuildProjectArgs) (*mcp.CallToolResult, error) {
	slog.InfoContext(ctx, "mcp tool call", "tool", "rebuild_project", "project", args.Project)
	if s.indexer == nil {
		return toolError(errReasonNotConfigured, "indexer is not configured", "check the Gnostis configuration"), nil
	}
	if args.Project == "" {
		return toolError(errReasonInvalidArgument, "project is required", "provide a project name from list_projects"), nil
	}

	jobID, err := s.indexer.StartRebuildProject(ctx, args.Project)
	if err != nil {
		slog.ErrorContext(ctx, "rebuild_project failed", "project", args.Project, "error", err)
		return toolError(errReasonSearchFailed, err.Error(), "try again later or check the project name"), nil
	}

	return mcp.NewToolResultText(fmt.Sprintf(`{"job_id":%q}`, jobID)), nil
}

type rebuildIndexArgs struct{}

func rebuildIndexTool() mcp.Tool {
	return mcp.NewTool("rebuild_index",
		mcp.WithDescription("Rebuild the entire index"),
	)
}

func (s *Server) rebuildIndex(ctx context.Context, request mcp.CallToolRequest, args rebuildIndexArgs) (*mcp.CallToolResult, error) {
	slog.InfoContext(ctx, "mcp tool call", "tool", "rebuild_index")
	if s.indexer == nil {
		return toolError(errReasonNotConfigured, "indexer is not configured", "check the Gnostis configuration"), nil
	}

	jobID, err := s.indexer.StartRebuildIndex(ctx)
	if err != nil {
		slog.ErrorContext(ctx, "rebuild_index failed", "error", err)
		return toolError(errReasonSearchFailed, err.Error(), "try again later"), nil
	}

	return mcp.NewToolResultText(fmt.Sprintf(`{"job_id":%q}`, jobID)), nil
}

type addProjectArgs struct {
	Path          string   `json:"path"`
	Name          string   `json:"name"`
	Extensions    []string `json:"extensions"`
	Include       []string `json:"include"`
	Exclude       []string `json:"exclude"`
	MaxFileSizeMB int      `json:"max_file_size_mb"`
}

func addProjectTool() mcp.Tool {
	return mcp.NewTool("add_project",
		mcp.WithDescription("Add a directory to the index. Does not start indexing — call rebuild_project separately."),
		mcp.WithString("path", mcp.Required(), mcp.Description("Absolute directory path")),
		mcp.WithString("name", mcp.Description("Project name (defaults to directory name)")),
		mcp.WithArray("extensions", mcp.Description("File extensions to index (e.g. .go, .py). Defaults to standard set if empty.")),
		mcp.WithArray("include", mcp.Description("Glob patterns to include (if set, only matching files are indexed)")),
		mcp.WithArray("exclude", mcp.Description("Glob patterns to exclude from indexing")),
		mcp.WithNumber("max_file_size_mb", mcp.Description("Skip files larger than this. Default: 5")),
	)
}

func (s *Server) addProject(ctx context.Context, request mcp.CallToolRequest, args addProjectArgs) (*mcp.CallToolResult, error) {
	slog.InfoContext(ctx, "mcp tool call", "tool", "add_project", "path", args.Path, "name", args.Name)
	if s.indexer == nil {
		return toolError(errReasonNotConfigured, "indexer is not configured", "check the Gnostis configuration"), nil
	}
	if args.Path == "" {
		return toolError(errReasonInvalidArgument, "path is required", "provide an absolute directory path"), nil
	}

	name, err := s.indexer.AddProject(ctx, args.Path, args.Name, args.Extensions, args.Include, args.Exclude, args.MaxFileSizeMB)
	if err != nil {
		slog.ErrorContext(ctx, "add_project failed", "path", args.Path, "error", err)
		return toolError(errReasonSearchFailed, err.Error(), "try again later or check the path"), nil
	}

	return mcp.NewToolResultText(fmt.Sprintf(`{"added":true,"name":%q,"hint":"use rebuild_project to index"}`, name)), nil
}

type editProjectArgs struct {
	Name          string   `json:"name"`
	Extensions    []string `json:"extensions"`
	Include       []string `json:"include"`
	Exclude       []string `json:"exclude"`
	MaxFileSizeMB int      `json:"max_file_size_mb"`
}

func editProjectTool() mcp.Tool {
	return mcp.NewTool("edit_project",
		mcp.WithDescription("Edit a project's indexing parameters (extensions, include/exclude patterns, max file size)"),
		mcp.WithString("name", mcp.Required(), mcp.Description("Project name")),
		mcp.WithArray("extensions", mcp.Description("File extensions to index (e.g. .go, .py). Defaults to standard set if empty.")),
		mcp.WithArray("include", mcp.Description("Glob patterns to include (if set, only matching files are indexed)")),
		mcp.WithArray("exclude", mcp.Description("Glob patterns to exclude from indexing")),
		mcp.WithNumber("max_file_size_mb", mcp.Description("Skip files larger than this. Default: 5")),
	)
}

func (s *Server) editProject(ctx context.Context, request mcp.CallToolRequest, args editProjectArgs) (*mcp.CallToolResult, error) {
	slog.InfoContext(ctx, "mcp tool call", "tool", "edit_project", "name", args.Name)
	if s.indexer == nil {
		return toolError(errReasonNotConfigured, "indexer is not configured", "check the Gnostis configuration"), nil
	}
	if args.Name == "" {
		return toolError(errReasonInvalidArgument, "name is required", "provide a project name from list_projects"), nil
	}

	if err := s.indexer.EditProject(ctx, args.Name, args.Extensions, args.Include, args.Exclude, args.MaxFileSizeMB); err != nil {
		slog.ErrorContext(ctx, "edit_project failed", "name", args.Name, "error", err)
		return toolError(errReasonSearchFailed, err.Error(), "try again later or check the project name"), nil
	}

	return mcp.NewToolResultText(fmt.Sprintf(`{"edited":true,"name":%q,"hint":"use rebuild_project to re-index with new settings"}`, args.Name)), nil
}

type removeProjectArgs struct {
	Name string `json:"name"`
}

func removeProjectTool() mcp.Tool {
	return mcp.NewTool("remove_project",
		mcp.WithDescription("Remove a project from the index and configuration"),
		mcp.WithString("name", mcp.Required(), mcp.Description("Project name")),
	)
}

func (s *Server) removeProject(ctx context.Context, request mcp.CallToolRequest, args removeProjectArgs) (*mcp.CallToolResult, error) {
	slog.InfoContext(ctx, "mcp tool call", "tool", "remove_project", "name", args.Name)
	if s.indexer == nil {
		return toolError(errReasonNotConfigured, "indexer is not configured", "check the Gnostis configuration"), nil
	}
	if args.Name == "" {
		return toolError(errReasonInvalidArgument, "name is required", "provide a project name from list_projects"), nil
	}

	if err := s.indexer.RemoveProject(ctx, args.Name); err != nil {
		slog.ErrorContext(ctx, "remove_project failed", "name", args.Name, "error", err)
		return toolError(errReasonSearchFailed, err.Error(), "try again later or check the project name"), nil
	}
	return mcp.NewToolResultText(`{"removed":true}`), nil
}
