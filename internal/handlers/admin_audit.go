package handlers

import (
	"log/slog"
	"net/http"
	"net/url"
	"strconv"
	"strings"

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
	Actions    []string
	Pagination AdminAuditPagination
}

type AdminAuditPagination struct {
	Search      string
	Action      string
	Page        int
	PerPage     int
	TotalCount  int
	TotalPages  int
	HasPrevious bool
	HasNext     bool
	PreviousURL string
	NextURL     string
}

func (h *AdminAuditHandler) Index(w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query()

	search := strings.TrimSpace(query.Get("q"))
	action := strings.TrimSpace(query.Get("action"))
	page := parsePositiveInt(query.Get("page"), 1)
	perPage := parsePositiveInt(query.Get("per_page"), 50)

	if perPage > 100 {
		perPage = 100
	}

	offset := (page - 1) * perPage

	logs, err := h.auditRepo.List(r.Context(), audit.ListParams{
		Search: search,
		Action: action,
		Limit:  perPage,
		Offset: offset,
	})
	if err != nil {
		slog.Error("failed to list audit logs", "error", err)
		http.Error(w, "Failed to load audit logs", http.StatusInternalServerError)
		return
	}

	totalPages := 0
	if logs.TotalCount > 0 {
		totalPages = (logs.TotalCount + perPage - 1) / perPage
	}

	if totalPages > 0 && page > totalPages {
		page = totalPages
		offset = (page - 1) * perPage

		logs, err = h.auditRepo.List(r.Context(), audit.ListParams{
			Search: search,
			Action: action,
			Limit:  perPage,
			Offset: offset,
		})
		if err != nil {
			slog.Error("failed to list audit logs after page correction", "error", err)
			http.Error(w, "Failed to load audit logs", http.StatusInternalServerError)
			return
		}
	}

	actions, err := h.auditRepo.DistinctActions(r.Context())
	if err != nil {
		slog.Error("failed to list audit actions", "error", err)
		http.Error(w, "Failed to load audit actions", http.StatusInternalServerError)
		return
	}

	hasPrevious := page > 1
	hasNext := totalPages > 0 && page < totalPages

	previousURL := ""
	if hasPrevious {
		previousURL = buildAdminAuditURL(search, action, page-1, perPage)
	}

	nextURL := ""
	if hasNext {
		nextURL = buildAdminAuditURL(search, action, page+1, perPage)
	}

	h.renderer.Render(w, r, "admin_audit.tmpl", views.TemplateData{
		Title: "Audit Log",
		Data: AdminAuditPageData{
			Logs:    logs.Logs,
			Actions: actions,
			Pagination: AdminAuditPagination{
				Search:      search,
				Action:      action,
				Page:        page,
				PerPage:     perPage,
				TotalCount:  logs.TotalCount,
				TotalPages:  totalPages,
				HasPrevious: hasPrevious,
				HasNext:     hasNext,
				PreviousURL: previousURL,
				NextURL:     nextURL,
			},
		},
	})
}

func buildAdminAuditURL(search string, action string, page int, perPage int) string {
	values := url.Values{}

	if strings.TrimSpace(search) != "" {
		values.Set("q", strings.TrimSpace(search))
	}

	if strings.TrimSpace(action) != "" {
		values.Set("action", strings.TrimSpace(action))
	}

	values.Set("page", strconv.Itoa(page))
	values.Set("per_page", strconv.Itoa(perPage))

	return "/admin/audit?" + values.Encode()
}
