package handlers

import (
	"errors"
	"log/slog"
	"net/http"
	"strings"

	"github.com/alexedwards/scs/v2"

	"github.com/ifaisalabid1/notes-platform/internal/admin"
	"github.com/ifaisalabid1/notes-platform/internal/auth"
	"github.com/ifaisalabid1/notes-platform/internal/dashboard"
	"github.com/ifaisalabid1/notes-platform/internal/views"
)

type AdminAuthHandler struct {
	adminRepo      *admin.Repository
	dashboardRepo  *dashboard.Repository
	sessionManager *scs.SessionManager
	renderer       *views.Renderer
}

func NewAdminAuthHandler(
	adminRepo *admin.Repository,
	dashboardRepo *dashboard.Repository,
	sessionManager *scs.SessionManager,
	renderer *views.Renderer,
) *AdminAuthHandler {
	return &AdminAuthHandler{
		adminRepo:      adminRepo,
		dashboardRepo:  dashboardRepo,
		sessionManager: sessionManager,
		renderer:       renderer,
	}
}

type AdminDashboardPageData struct {
	Stats dashboard.Stats
}

func (h *AdminAuthHandler) ShowLogin(w http.ResponseWriter, r *http.Request) {
	if h.sessionManager.Exists(r.Context(), "admin_id") {
		http.Redirect(w, r, "/admin/dashboard", http.StatusSeeOther)
		return
	}

	h.renderer.Render(w, r, "admin_login.tmpl", views.TemplateData{
		Title: "Admin Login",
	})
}

func (h *AdminAuthHandler) Login(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		h.renderLoginError(w, r, "Invalid form submission.")
		return
	}

	email := strings.TrimSpace(r.PostForm.Get("email"))
	password := r.PostForm.Get("password")

	if email == "" || password == "" {
		h.renderLoginError(w, r, "Email and password are required.")
		return
	}

	foundAdmin, err := h.adminRepo.FindByEmail(r.Context(), email)
	if err != nil {
		if errors.Is(err, admin.ErrAdminNotFound) {
			h.renderLoginError(w, r, "Invalid email or password.")
			return
		}

		slog.Error("failed to find admin by email", "error", err)
		h.renderLoginError(w, r, "Something went wrong. Please try again.")
		return
	}

	matches, err := auth.VerifyPassword(password, foundAdmin.PasswordHash)
	if err != nil {
		slog.Error("failed to verify password", "error", err)
		h.renderLoginError(w, r, "Something went wrong. Please try again.")
		return
	}

	if !matches {
		h.renderLoginError(w, r, "Invalid email or password.")
		return
	}

	if err := h.sessionManager.RenewToken(r.Context()); err != nil {
		slog.Error("failed to renew session token", "error", err)
		h.renderLoginError(w, r, "Something went wrong. Please try again.")
		return
	}

	h.sessionManager.Put(r.Context(), "admin_id", foundAdmin.ID.String())
	h.sessionManager.Put(r.Context(), "admin_name", foundAdmin.Name)
	h.sessionManager.Put(r.Context(), "admin_email", foundAdmin.Email)
	h.sessionManager.Put(r.Context(), "admin_role", string(foundAdmin.Role))
	h.sessionManager.Put(r.Context(), "admin_is_owner", foundAdmin.IsOwner)

	http.Redirect(w, r, "/admin/dashboard", http.StatusSeeOther)
}

func (h *AdminAuthHandler) Logout(w http.ResponseWriter, r *http.Request) {
	if err := h.sessionManager.Destroy(r.Context()); err != nil {
		slog.Error("failed to destroy session", "error", err)
		http.Redirect(w, r, "/admin/dashboard", http.StatusSeeOther)
		return
	}

	http.Redirect(w, r, "/admin/login", http.StatusSeeOther)
}

func (h *AdminAuthHandler) Dashboard(w http.ResponseWriter, r *http.Request) {
	stats, err := h.dashboardRepo.Stats(r.Context())
	if err != nil {
		http.Error(w, "Failed to load dashboard", http.StatusInternalServerError)
		return
	}

	h.renderer.Render(w, r, "admin_dashboard.tmpl", views.TemplateData{
		Title: "Dashboard",
		Data: AdminDashboardPageData{
			Stats: stats,
		},
	})
}

func (h *AdminAuthHandler) renderLoginError(w http.ResponseWriter, r *http.Request, message string) {
	w.WriteHeader(http.StatusUnprocessableEntity)

	h.renderer.Render(w, r, "admin_login.tmpl", views.TemplateData{
		Title: "Admin Login",
		Error: message,
	})
}
