package web

import (
	"log/slog"
	"net/http"
)

type rebuildProjectRequest struct {
	Name string `json:"name"`
}

type addProjectRequest struct {
	Path          string   `json:"path"`
	Name          string   `json:"name"`
	Extensions    []string `json:"extensions"`
	Include       []string `json:"include"`
	Exclude       []string `json:"exclude"`
	MaxFileSizeMB int      `json:"max_file_size_mb"`
}

type editProjectRequest struct {
	Name          string   `json:"name"`
	Extensions    []string `json:"extensions"`
	Include       []string `json:"include"`
	Exclude       []string `json:"exclude"`
	MaxFileSizeMB int      `json:"max_file_size_mb"`
}

type removeProjectRequest struct {
	Name string `json:"name"`
}

type reindexRequest struct {
	Paths []string `json:"paths"`
}

type jobResponse struct {
	JobID string `json:"job_id"`
}

type nameResponse struct {
	Name string `json:"name"`
}

func (s *Server) handleRebuildProject(w http.ResponseWriter, r *http.Request) {
	var req rebuildProjectRequest
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	if req.Name == "" {
		writeError(w, http.StatusBadRequest, "name is required")
		return
	}

	jobID, err := s.app.StartRebuildProject(r.Context(), req.Name)
	if err != nil {
		slog.ErrorContext(r.Context(), "rebuild project", "name", req.Name, "error", err)
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusAccepted, jobResponse{JobID: jobID})
}

func (s *Server) handleRebuildIndex(w http.ResponseWriter, r *http.Request) {
	jobID, err := s.app.StartRebuildIndex(r.Context())
	if err != nil {
		slog.ErrorContext(r.Context(), "rebuild index", "error", err)
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusAccepted, jobResponse{JobID: jobID})
}

func (s *Server) handleAddProject(w http.ResponseWriter, r *http.Request) {
	var req addProjectRequest
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	if req.Path == "" {
		writeError(w, http.StatusBadRequest, "path is required")
		return
	}

	name, err := s.app.AddProject(r.Context(), req.Path, req.Name, req.Extensions, req.Include, req.Exclude, req.MaxFileSizeMB)
	if err != nil {
		slog.ErrorContext(r.Context(), "add project", "path", req.Path, "error", err)
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusCreated, nameResponse{Name: name})
}

func (s *Server) handleEditProject(w http.ResponseWriter, r *http.Request) {
	var req editProjectRequest
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	if req.Name == "" {
		writeError(w, http.StatusBadRequest, "name is required")
		return
	}

	if err := s.app.EditProject(r.Context(), req.Name, req.Extensions, req.Include, req.Exclude, req.MaxFileSizeMB); err != nil {
		slog.ErrorContext(r.Context(), "edit project", "name", req.Name, "error", err)
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "edited"})
}

func (s *Server) handleRemoveProject(w http.ResponseWriter, r *http.Request) {
	var req removeProjectRequest
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	if req.Name == "" {
		writeError(w, http.StatusBadRequest, "name is required")
		return
	}

	if err := s.app.RemoveProject(r.Context(), req.Name); err != nil {
		slog.ErrorContext(r.Context(), "remove project", "name", req.Name, "error", err)
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "removed"})
}

func (s *Server) handleReindex(w http.ResponseWriter, r *http.Request) {
	var req reindexRequest
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	if len(req.Paths) == 0 {
		writeError(w, http.StatusBadRequest, "paths is required")
		return
	}

	if err := s.app.ReindexFiles(r.Context(), req.Paths); err != nil {
		slog.ErrorContext(r.Context(), "reindex files", "paths", req.Paths, "error", err)
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "reindexed"})
}
