package handlers

import (
	"log/slog"
	"net/http"

	"github.com/ifaisalabid1/notes-platform/internal/audit"
	"github.com/ifaisalabid1/notes-platform/internal/views"
)

type AdminAuditHandler struct {
	auditRepo *audit.Repository
	renderer  *views.Renderer
}

func NewAdminAuditHandler(auditRepo *audit.Repository, renderer *views.Renderer) *AdminAuditHandler {
	return &AdminAuditHandler{
		auditRepo: auditRepo,
		renderer:  renderer,
	}
}

type AdminAuditPageData struct {
	Logs       []audit.Log
	TotalCount int
}

func (h *AdminAuditHandler) Index(w http.ResponseWriter, r *http.Request) {
	logs, err := h.auditRepo.List(r.Context(), audit.ListParams{
		Limit: 50,
	})
	if err != nil {
		slog.Error("failed to list audit logs", "error", err)
		http.Error(w, "Failed to load audit logs", http.StatusInternalServerError)
		return
	}

	h.renderer.Render(w, r, "admin_audit.tmpl", views.TemplateData{
		Title: "Audit Log",
		Data: AdminAuditPageData{
			Logs:       logs.Logs,
			TotalCount: logs.TotalCount,
		},
	})
}
