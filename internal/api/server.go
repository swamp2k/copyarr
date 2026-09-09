package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"

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
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(v)
}
