package handlers

import (
	"net/http"

	"github.com/ifaisalabid1/notes-platform/internal/views"
)

type MaintenanceHandler struct {
	renderer *views.Renderer
}

func NewMaintenanceHandler(renderer *views.Renderer) *MaintenanceHandler {
	return &MaintenanceHandler{
		renderer: renderer,
	}
}

func (h *MaintenanceHandler) Show(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusServiceUnavailable)

	h.renderer.Render(w, r, "maintenance.tmpl", views.TemplateData{
		Title:       "Maintenance",
		Description: "Rising Star is temporarily unavailable while maintenance is in progress.",
	})
}
