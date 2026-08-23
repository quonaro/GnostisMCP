package graph

import (
	"fmt"

	"github.com/quonaro/gnostis/internal/db"
)

// Save writes the graph to SQLite.
func (g *Graph) Save(database *db.DB) error {
	g.mu.RLock()
	defer g.mu.RUnlock()

	tx, err := database.Writer().Begin()
	if err != nil {
		return fmt.Errorf("begin tx: %w", err)
	}
	if _, err := tx.Exec(`DELETE FROM graph_nodes`); err != nil {
		_ = tx.Rollback()
		return fmt.Errorf("delete graph nodes: %w", err)
	}
	if _, err := tx.Exec(`DELETE FROM graph_edges`); err != nil {
		_ = tx.Rollback()
		return fmt.Errorf("delete graph edges: %w", err)
	}
	nodeStmt, err := tx.Prepare(`INSERT INTO graph_nodes (id, path, symbol, kind, language, start_line, end_line) VALUES (?, ?, ?, ?, ?, ?, ?)`)
	if err != nil {
		_ = tx.Rollback()
		return fmt.Errorf("prepare node insert: %w", err)
	}
	defer func() { _ = nodeStmt.Close() }()
	for _, n := range g.nodes {
		if _, err := nodeStmt.Exec(n.ID, n.Path, n.Symbol, n.Kind, n.Language, n.StartLine, n.EndLine); err != nil {
			_ = tx.Rollback()
			return fmt.Errorf("insert node %s: %w", n.ID, err)
		}
	}
	edgeStmt, err := tx.Prepare(`INSERT INTO graph_edges (from_id, "to", line) VALUES (?, ?, ?)`)
	if err != nil {
		_ = tx.Rollback()
		return fmt.Errorf("prepare edge insert: %w", err)
	}
	defer func() { _ = edgeStmt.Close() }()
	for _, e := range g.edges {
		if _, err := edgeStmt.Exec(e.From, e.To, e.Line); err != nil {
			_ = tx.Rollback()
			return fmt.Errorf("insert edge: %w", err)
		}
	}
	return tx.Commit()
}

// Load reads the graph from SQLite.
func (g *Graph) Load(database *db.DB) error {
	rows, err := database.Reader().Query(`SELECT id, path, symbol, kind, language, start_line, end_line FROM graph_nodes`)
	if err != nil {
		return fmt.Errorf("query graph nodes: %w", err)
	}
	defer func() { _ = rows.Close() }()

	g.mu.Lock()
	defer g.mu.Unlock()

	g.nodes = make(map[string]Node)
	g.files = make(map[string]bool)
	for rows.Next() {
		var n Node
		if err := rows.Scan(&n.ID, &n.Path, &n.Symbol, &n.Kind, &n.Language, &n.StartLine, &n.EndLine); err != nil {
			return fmt.Errorf("scan node row: %w", err)
		}
		g.nodes[n.ID] = n
		g.files[n.Path] = true
	}
	if err := rows.Err(); err != nil {
		return fmt.Errorf("iterate nodes: %w", err)
	}

	edgeRows, err := database.Reader().Query(`SELECT from_id, "to", line FROM graph_edges`)
	if err != nil {
		return fmt.Errorf("query graph edges: %w", err)
	}
	defer func() { _ = edgeRows.Close() }()
	for edgeRows.Next() {
		var e Edge
		if err := edgeRows.Scan(&e.From, &e.To, &e.Line); err != nil {
			return fmt.Errorf("scan edge row: %w", err)
		}
		g.edges = append(g.edges, e)
	}
	return edgeRows.Err()
}
