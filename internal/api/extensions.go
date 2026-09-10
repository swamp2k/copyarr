package api

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/swamp2k/copyarr/internal/engine"
)

// EnhanceHandler layers newer dashboard/API capabilities over the original
// mux without making the core route table increasingly monolithic.
func EnhanceHandler(base http.Handler, eng *engine.Engine) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodGet && r.URL.Path == "/":
			w.Header().Set("Content-Type", "text/html; charset=utf-8")
			_, _ = w.Write([]byte(enhanceUI(uiHTML)))
			return

		case r.Method == http.MethodGet && r.URL.Path == "/api/settings":
			write(w, eng.SettingsExtended())
			return

		case r.Method == http.MethodPatch && r.URL.Path == "/api/settings":
			var req struct {
				LoggingEnabled   *bool `json:"logging_enabled"`
				LogRetentionDays *int  `json:"log_retention_days"`
			}
			if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
				http.Error(w, "invalid JSON", http.StatusBadRequest)
				return
			}
			if req.LoggingEnabled != nil {
				if err := eng.SetLoggingEnabled(*req.LoggingEnabled); err != nil {
					http.Error(w, err.Error(), http.StatusInternalServerError)
					return
				}
			}
			if req.LogRetentionDays != nil {
				if err := eng.SetLogRetentionDays(*req.LogRetentionDays); err != nil {
					http.Error(w, err.Error(), http.StatusBadRequest)
					return
				}
			}
			write(w, eng.SettingsExtended())
			return

		case r.Method == http.MethodGet && r.URL.Path == "/api/jobs":
			limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
			items, err := eng.ExecutionRows(limit)
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
			write(w, items)
			return
		}
		base.ServeHTTP(w, r)
	})
}
