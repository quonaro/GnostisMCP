package mcp

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"sort"
	"strings"

	"github.com/mark3labs/mcp-go/mcp"

	"github.com/quonaro/gnostis/internal/search"
)

const (
	searchSourceCode = "code"
	searchSourceDocs = "docs"
	searchSourceAll  = "all"
)

type unifiedSearchArgs struct {
	Query          string  `json:"query"`
	Source         string  `json:"source"`
	Project        string  `json:"project"`
	Path           string  `json:"path"`
	Language       string  `json:"language"`
	TopK           float64 `json:"top_k"`
	IncludeContent bool    `json:"include_content"`
}

type unifiedSearchResult struct {
	Source    string  `json:"source"`
	ProjectID string  `json:"project,omitempty"`
	Path      string  `json:"path"`
	Language  string  `json:"language,omitempty"`
	Symbol    string  `json:"symbol,omitempty"`
	Signature string  `json:"signature,omitempty"`
	StartLine int     `json:"start_line"`
	EndLine   int     `json:"end_line"`
	Score     float32 `json:"score"`
	Content   string  `json:"content,omitempty"`
}

func unifiedSearchTool() mcp.Tool {
	return mcp.NewTool("search",
		mcp.WithDescription("Semantic search across indexed code and documentation"),
		mcp.WithString("query", mcp.Required(), mcp.Description("Natural language search query")),
		mcp.WithString("source", mcp.Description("Where to search: code, docs, or all"), mcp.DefaultString(searchSourceAll)),
		mcp.WithString("project", mcp.Description("Project name to restrict the search")),
		mcp.WithString("path", mcp.Description("Absolute or project-relative path prefix")),
		mcp.WithString("language", mcp.Description("Language filter, e.g. go, python, markdown")),
		mcp.WithNumber("top_k", mcp.Description("Number of results"), mcp.DefaultNumber(10)),
		mcp.WithBoolean("include_content", mcp.Description("Include full chunk text"), mcp.DefaultBool(true)),
	)
}

func (s *Server) unifiedSearch(ctx context.Context, request mcp.CallToolRequest, args unifiedSearchArgs) (*mcp.CallToolResult, error) {
	slog.InfoContext(ctx, "mcp tool call", "tool", "search", "query", args.Query, "source", args.Source, "project", args.Project, "path", args.Path, "language", args.Language)
	if strings.TrimSpace(args.Query) == "" {
		return toolError(errReasonInvalidArgument, "query is required", "provide a non-empty search query"), nil
	}

	source := strings.ToLower(strings.TrimSpace(args.Source))
	if source == "" {
		source = searchSourceAll
	}
	validSources := map[string]bool{searchSourceCode: true, searchSourceDocs: true, searchSourceAll: true}
	if !validSources[source] {
		return toolError(errReasonInvalidArgument, fmt.Sprintf("invalid source %q", args.Source), "use one of: code, docs, all"), nil
	}

	topK := int(args.TopK)
	if topK <= 0 {
		topK = 10
	}

	var all []unifiedSearchResult

	if source == searchSourceCode || source == searchSourceAll {
		results, toolErr := s.searchCode(ctx, args, topK)
		if toolErr != nil {
			return toolErr, nil
		}
		all = append(all, results...)
	}

	if source == searchSourceDocs || source == searchSourceAll {
		results, toolErr := s.searchDocs(ctx, args, topK)
		if toolErr != nil {
			return toolErr, nil
		}
		all = append(all, results...)
	}

	deduped := deduplicateUnifiedResults(all)
	sort.Slice(deduped, func(i, j int) bool { return deduped[i].Score > deduped[j].Score })
	if len(deduped) > topK {
		deduped = deduped[:topK]
	}

	data, err := json.Marshal(deduped)
	if err != nil {
		return toolError(errReasonSearchFailed, err.Error(), "internal error marshalling search results"), nil
	}
	return mcp.NewToolResultText(string(data)), nil
}

func (s *Server) searchCode(ctx context.Context, args unifiedSearchArgs, topK int) ([]unifiedSearchResult, *mcp.CallToolResult) {
	filters := map[string]string{}
	if args.Project != "" {
		filters["project_id"] = args.Project
	}
	if args.Language != "" {
		filters["language"] = strings.ToLower(args.Language)
	}
	if args.Path != "" {
		prefix, err := s.resolvePathOrAbsolute(args.Project, args.Path)
		if err != nil {
			return nil, toolErrorFromResolvePath(err)
		}
		filters["path"] = prefix
	}

	results, err := s.engine.Search(ctx, args.Query, filters, topK)
	if err != nil {
		slog.ErrorContext(ctx, "search code failed", "query", args.Query, "error", err)
		return nil, toolError(errReasonSearchFailed, err.Error(), "try again later or check the index status")
	}

	items := make([]unifiedSearchResult, len(results))
	for i, r := range results {
		items[i] = searchResultToUnified(r, searchSourceCode, args.IncludeContent)
	}
	return items, nil
}

func (s *Server) searchDocs(ctx context.Context, args unifiedSearchArgs, topK int) ([]unifiedSearchResult, *mcp.CallToolResult) {
	filters := map[string]string{"language": "markdown"}
	if args.Project != "" {
		filters["project_id"] = args.Project
	}
	if args.Path != "" {
		prefix, err := s.resolvePathOrAbsolute(args.Project, args.Path)
		if err != nil {
			return nil, toolErrorFromResolvePath(err)
		}
		filters["path"] = prefix
	}

	results, err := s.engine.Search(ctx, args.Query, filters, topK)
	if err != nil {
		slog.ErrorContext(ctx, "search docs failed", "query", args.Query, "error", err)
		return nil, toolError(errReasonSearchFailed, err.Error(), "try again later or check the index status")
	}

	items := make([]unifiedSearchResult, len(results))
	for i, r := range results {
		items[i] = searchResultToUnified(r, searchSourceDocs, args.IncludeContent)
	}
	return items, nil
}

func searchResultToUnified(r search.Result, source string, includeContent bool) unifiedSearchResult {
	res := unifiedSearchResult{
		Source:    source,
		ProjectID: r.ProjectID,
		Path:      r.Path,
		Language:  r.Language,
		Symbol:    r.Symbol,
		Signature: r.Signature,
		StartLine: r.StartLine,
		EndLine:   r.EndLine,
		Score:     r.Score,
	}
	if includeContent {
		res.Content = r.Content
	}
	return res
}

func deduplicateUnifiedResults(results []unifiedSearchResult) []unifiedSearchResult {
	seen := make(map[string]bool)
	out := make([]unifiedSearchResult, 0, len(results))
	for _, r := range results {
		key := fmt.Sprintf("%s:%d:%d", r.Path, r.StartLine, r.EndLine)
		if seen[key] {
			continue
		}
		seen[key] = true
		out = append(out, r)
	}
	return out
}
