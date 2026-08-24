package mcp

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/mark3labs/mcp-go/mcp"
)

type grepArgs struct {
	Query   string  `json:"query"`
	Project string  `json:"project"`
	Path    string  `json:"path"`
	Regex   bool    `json:"regex"`
	TopK    float64 `json:"top_k"`
}

type grepMatch struct {
	Path    string `json:"path"`
	Line    int    `json:"line"`
	Content string `json:"content"`
}

func (s *Server) grep(ctx context.Context, request mcp.CallToolRequest, args grepArgs) (*mcp.CallToolResult, error) {
	slog.InfoContext(ctx, "mcp tool call", "tool", "grep", "query", args.Query, "project", args.Project, "path", args.Path)
	if args.Query == "" {
		return toolError(errReasonInvalidArgument, "query is required", "provide a non-empty search query"), nil
	}

	root, err := s.resolvePathOrAbsolute(args.Project, args.Path)
	if err != nil {
		return toolErrorFromResolvePath(err), nil
	}

	topK := int(args.TopK)
	if topK <= 0 {
		topK = 20
	}

	var re *regexp.Regexp
	if args.Regex {
		re, err = regexp.Compile(args.Query)
		if err != nil {
			return toolError(errReasonInvalidRegex, fmt.Sprintf("invalid regex: %v", err), "fix the regular expression"), nil
		}
	}

	var matches []grepMatch
	err = filepath.WalkDir(root, func(path string, d os.DirEntry, walkErr error) error {
		if walkErr != nil || d.IsDir() {
			return nil
		}
		if !isTextFile(path) {
			return nil
		}
		info, walkErr := d.Info()
		if walkErr != nil || info.Size() > 1<<20 {
			return nil
		}
		content, readErr := os.ReadFile(path)
		if readErr != nil {
			return nil
		}
		lines := strings.Split(string(content), "\n")
		for i, line := range lines {
			matched := (re != nil && re.MatchString(line)) || (re == nil && strings.Contains(line, args.Query))
			if matched {
				matches = append(matches, grepMatch{Path: path, Line: i + 1, Content: line})
				if len(matches) >= topK {
					return errGrepStop
				}
			}
		}
		return nil
	})
	if err != nil && err != errGrepStop {
		return nil, fmt.Errorf("grep walk: %w", err)
	}

	data, err := json.Marshal(matches)
	if err != nil {
		return toolError(errReasonSearchFailed, err.Error(), "internal error marshalling matches"), nil
	}
	return mcp.NewToolResultText(string(data)), nil
}

var errGrepStop = fmt.Errorf("grep stop")

func grepTool() mcp.Tool {
	return mcp.NewTool("grep",
		mcp.WithDescription("Search file contents by substring or regex. Requires project or path."),
		mcp.WithString("query", mcp.Required(), mcp.Description("Text or regex to search")),
		mcp.WithString("project", mcp.Description("Project name to restrict the search (required if path is omitted)")),
		mcp.WithString("path", mcp.Description("Relative path within the project (required if project is omitted)")),
		mcp.WithBoolean("regex", mcp.Description("Treat query as regex"), mcp.DefaultBool(false)),
		mcp.WithNumber("top_k", mcp.Description("Maximum number of matches"), mcp.DefaultNumber(20)),
	)
}
