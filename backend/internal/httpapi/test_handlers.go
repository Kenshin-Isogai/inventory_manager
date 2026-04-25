package httpapi

import "net/http"

func (h Handlers) ResetTestData(w http.ResponseWriter, r *http.Request) {
	if h.cfg.App.Env != "test" && h.cfg.App.Mode != "test" {
		http.NotFound(w, r)
		return
	}
	if h.cfg.Auth.Mode == "enforced" && !h.requireActiveRole(w, r, "admin") {
		return
	}
	if err := h.phaseOne.ResetTestData(r.Context()); err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}
