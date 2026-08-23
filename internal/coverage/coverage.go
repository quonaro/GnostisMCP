package coverage

// Status describes the index state of a single path.
type Status struct {
	Path      string `json:"path"`
	Status    string `json:"status"` // indexed | stale | not_indexed | deleted
	FileHash  string `json:"file_hash,omitempty"`
	IndexHash string `json:"index_hash,omitempty"`
}

// Change describes a file whose on-disk state differs from the index.
type Change struct {
	Path   string `json:"path"`
	Status string `json:"status"` // modified | new | deleted
}
