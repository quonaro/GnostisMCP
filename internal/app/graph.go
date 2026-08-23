package app

import (
	"log/slog"

	"github.com/quonaro/gnostis/internal/graph"
)

// CallGraph returns the application's call graph.
func (a *App) CallGraph() *graph.Graph {
	return a.callGraph
}

// saveCallGraph persists the call graph to disk.
func (a *App) saveCallGraph() {
	if a.callGraph == nil {
		return
	}
	if err := a.callGraph.Save(a.cfg.DataDir + "/call_graph.json"); err != nil {
		slog.Error("save call graph", "error", err)
	}
}

// saveSimhashIndex persists the simhash index to disk.
func (a *App) saveSimhashIndex() {
	if a.simhashIndex == nil {
		return
	}
	if err := a.simhashIndex.Save(); err != nil {
		slog.Error("save simhash index", "error", err)
	}
}
