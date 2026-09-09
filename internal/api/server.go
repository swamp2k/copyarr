package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"github.com/swamp2k/copyarr/internal/config"
	"github.com/swamp2k/copyarr/internal/db"
	"github.com/swamp2k/copyarr/internal/engine"
)

type Server struct {
	eng *engine.Engine
}

func New(e *engine.Engine) *Server { return &Server{eng: e} }

func (s *Server) Handler() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/" {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		_, _ = w.Write([]byte(uiHTML))
	})
	mux.HandleFunc("GET /health", func(w http.ResponseWriter, r *http.Request) {
		write(w, map[string]any{"ok": true, "service": "copyarr"})
	})
	mux.HandleFunc("GET /api/status", func(w http.ResponseWriter, r *http.Request) {
		status, err := s.eng.Status()
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		write(w, status)
	})
	mux.HandleFunc("GET /api/rules", func(w http.ResponseWriter, r *http.Request) {
		write(w, s.eng.Rules())
	})
	mux.HandleFunc("GET /api/job-definitions", func(w http.ResponseWriter, r *http.Request) {
		write(w, s.eng.Rules())
	})
	mux.HandleFunc("POST /api/job-definitions", func(w http.ResponseWriter, r *http.Request) {
		var def config.Rule
		if err := json.NewDecoder(r.Body).Decode(&def); err != nil {
			http.Error(w, "invalid JSON", http.StatusBadRequest)
			return
		}
		if err := s.eng.SaveJobDefinition(def); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		write(w, map[string]any{"ok": true, "id": def.ID})
	})
	mux.HandleFunc("PUT /api/job-definitions/{id}", func(w http.ResponseWriter, r *http.Request) {
		var def config.Rule
		if err := json.NewDecoder(r.Body).Decode(&def); err != nil {
			http.Error(w, "invalid JSON", http.StatusBadRequest)
			return
		}
		def.ID = r.PathValue("id")
		if err := s.eng.SaveJobDefinition(def); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		write(w, map[string]any{"ok": true, "id": def.ID})
	})
	mux.HandleFunc("DELETE /api/job-definitions/{id}", func(w http.ResponseWriter, r *http.Request) {
		if err := s.eng.DeleteJobDefinition(r.PathValue("id")); err != nil {
			http.Error(w, err.Error(), http.StatusConflict)
			return
		}
		write(w, map[string]any{"ok": true})
	})
	mux.HandleFunc("POST /api/job-definitions/{id}/reset", func(w http.ResponseWriter, r *http.Request) {
		if err := s.eng.ResetJobDefinition(r.PathValue("id")); err != nil {
			http.Error(w, err.Error(), http.StatusConflict)
			return
		}
		write(w, map[string]any{"ok": true, "id": r.PathValue("id")})
	})
	mux.HandleFunc("PATCH /api/rules/{id}", func(w http.ResponseWriter, r *http.Request) {
		id := r.PathValue("id")
		current, ok := s.eng.RetryPolicy(id)
		if !ok {
			http.Error(w, "rule not found", http.StatusNotFound)
			return
		}
		var req struct {
			RetryCount       *int `json:"retry_count"`
			RetryWaitSeconds *int `json:"retry_wait_seconds"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "invalid JSON", http.StatusBadRequest)
			return
		}
		count := current.RetryCount
		wait := current.RetryWaitSeconds
		if req.RetryCount != nil {
			count = *req.RetryCount
		}
		if req.RetryWaitSeconds != nil {
			wait = *req.RetryWaitSeconds
		}
		if err := s.eng.UpdateRetryPolicy(id, count, wait); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		policy, _ := s.eng.RetryPolicy(id)
		write(w, policy)
	})
	mux.HandleFunc("GET /api/remotes", func(w http.ResponseWriter, r *http.Request) {
		items, err := s.eng.Remotes(r.Context())
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		write(w, items)
	})
	mux.HandleFunc("GET /api/remotes/providers", func(w http.ResponseWriter, r *http.Request) {
		raw, err := s.eng.RemoteProviders(r.Context())
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write(raw)
	})
	mux.HandleFunc("GET /api/remotes/{name}", func(w http.ResponseWriter, r *http.Request) {
		detail, err := s.eng.RemoteConfig(r.Context(), r.PathValue("name"))
		if err != nil {
			http.Error(w, err.Error(), http.StatusNotFound)
			return
		}
		write(w, detail)
	})
	mux.HandleFunc("POST /api/remotes/test", func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			Name       string            `json:"name"`
			Type       string            `json:"type"`
			Parameters map[string]string `json:"parameters"`
			Path       string            `json:"path"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "invalid JSON", http.StatusBadRequest)
			return
		}
		// A type means "check this draft before saving it"; a bare name means
		// "check the remote that is already in the config".
		if req.Type != "" {
			result, err := s.eng.TestRemoteConfig(r.Context(), req.Type, req.Parameters, req.Path)
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
			write(w, result)
			return
		}
		if req.Name == "" {
			http.Error(w, "name or type is required", http.StatusBadRequest)
			return
		}
		write(w, s.eng.TestRemote(r.Context(), req.Name, req.Path))
	})
	mux.HandleFunc("POST /api/remotes", func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			Name       string            `json:"name"`
			Type       string            `json:"type"`
			Parameters map[string]string `json:"parameters"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "invalid JSON", http.StatusBadRequest)
			return
		}
		if req.Name == "" || req.Type == "" {
			http.Error(w, "name and type are required", http.StatusBadRequest)
			return
		}
		q, err := s.eng.CreateRemote(r.Context(), req.Name, req.Type, req.Parameters)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		write(w, map[string]any{"ok": q == nil, "question": q})
	})
	mux.HandleFunc("PATCH /api/remotes/{name}", func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			Parameters map[string]string `json:"parameters"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "invalid JSON", http.StatusBadRequest)
			return
		}
		q, err := s.eng.UpdateRemote(r.Context(), r.PathValue("name"), req.Parameters)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		write(w, map[string]any{"ok": q == nil, "question": q})
	})
	mux.HandleFunc("GET /api/remotes/{name}/usage", func(w http.ResponseWriter, r *http.Request) {
		write(w, s.eng.RemoteUsage(r.PathValue("name")))
	})
	mux.HandleFunc("DELETE /api/remotes/{name}", func(w http.ResponseWriter, r *http.Request) {
		name := r.PathValue("name")
		// Deleting a remote that jobs still point at breaks them silently at
		// the next scan, so it takes a deliberate second request to go through.
		if usage := s.eng.RemoteUsage(name); len(usage) > 0 && r.URL.Query().Get("confirm") != "true" {
			writeStatus(w, http.StatusConflict, map[string]any{
				"error":   fmt.Sprintf("remote %q is used by %d job definition(s)", name, len(usage)),
				"used_by": usage,
				"confirm": "repeat this request with ?confirm=true to delete it anyway",
				"blocked": true,
			})
			return
		}
		if err := s.eng.DeleteRemote(r.Context(), name); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		write(w, map[string]any{"ok": true})
	})
	mux.HandleFunc("GET /api/jobs/{id}", func(w http.ResponseWriter, r *http.Request) {
		id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
		if err != nil {
			http.Error(w, "invalid job id", http.StatusBadRequest)
			return
		}
		detail, err := s.eng.JobDetail(id)
		if err != nil {
			if db.IsNoRows(err) {
				http.Error(w, "job not found", http.StatusNotFound)
			} else {
				http.Error(w, err.Error(), http.StatusInternalServerError)
			}
			return
		}
		write(w, detail)
	})
	mux.HandleFunc("GET /api/logs", func(w http.ResponseWriter, r *http.Request) {
		limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
		var jobID *int64
		if raw := r.URL.Query().Get("job_id"); raw != "" {
			id, err := strconv.ParseInt(raw, 10, 64)
			if err != nil {
				http.Error(w, "invalid job_id", http.StatusBadRequest)
				return
			}
			jobID = &id
		}
		items, err := s.eng.Logs(limit, jobID)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		write(w, items)
	})
	mux.HandleFunc("DELETE /api/logs", func(w http.ResponseWriter, r *http.Request) {
		if err := s.eng.ClearLogs(); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		write(w, map[string]any{"ok": true})
	})
	mux.HandleFunc("GET /api/settings", func(w http.ResponseWriter, r *http.Request) {
		write(w, s.eng.Settings())
	})
	mux.HandleFunc("PATCH /api/settings", func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			LoggingEnabled *bool `json:"logging_enabled"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "invalid JSON", http.StatusBadRequest)
			return
		}
		if req.LoggingEnabled != nil {
			if err := s.eng.SetLoggingEnabled(*req.LoggingEnabled); err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
		}
		write(w, s.eng.Settings())
	})
	mux.HandleFunc("GET /api/jobs", func(w http.ResponseWriter, r *http.Request) {
		limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
		items, err := s.eng.Jobs(limit)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		write(w, items)
	})
	mux.HandleFunc("GET /api/objects", func(w http.ResponseWriter, r *http.Request) {
		limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
		items, err := s.eng.DB().List(limit)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		write(w, items)
	})
	mux.HandleFunc("POST /api/jobs/", func(w http.ResponseWriter, r *http.Request) {
		parts := strings.Split(strings.Trim(r.URL.Path, "/"), "/")
		if len(parts) != 4 || parts[0] != "api" || parts[1] != "jobs" {
			http.NotFound(w, r)
			return
		}
		id, err := strconv.ParseInt(parts[2], 10, 64)
		if err != nil {
			http.Error(w, "invalid job id", http.StatusBadRequest)
			return
		}
		action := parts[3]
		switch action {
		case "retry", "resume", "pause", "cancel":
			err = s.eng.ControlJob(id, action)
		default:
			http.NotFound(w, r)
			return
		}
		if err != nil {
			http.Error(w, fmt.Sprintf("%s failed: %v", action, err), http.StatusConflict)
			return
		}
		write(w, map[string]any{"ok": true, "job_id": id, "action": action})
	})
	mux.HandleFunc("POST /api/scan", func(w http.ResponseWriter, r *http.Request) {
		s.eng.TriggerScan()
		write(w, map[string]any{"queued": true})
	})
	return mux
}

func write(w http.ResponseWriter, v any) {
	writeStatus(w, http.StatusOK, v)
}

func writeStatus(w http.ResponseWriter, code int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	_ = json.NewEncoder(w).Encode(v)
}
