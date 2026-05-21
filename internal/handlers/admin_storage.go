package handlers

import (
	"net/http"

	"github.com/ifaisalabid1/notes-platform/internal/storage"
)

type AdminStorageHandler struct {
	r2 *storage.R2Client
}

func NewAdminStorageHandler(r2 *storage.R2Client) *AdminStorageHandler {
	return &AdminStorageHandler{
		r2: r2,
	}
}

func (h *AdminStorageHandler) Ready(w http.ResponseWriter, r *http.Request) {
	if err := h.r2.Check(r.Context()); err != nil {
		writeJSON(w, http.StatusServiceUnavailable, map[string]string{
			"status": "unavailable",
			"error":  "r2 bucket not ready",
		})
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{
		"status":  "ok",
		"storage": "r2 connected",
	})
}
