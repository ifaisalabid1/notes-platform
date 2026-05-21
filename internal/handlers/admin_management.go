package handlers

import (
	"errors"
	"log/slog"
	"net/http"
	"strings"

	"github.com/alexedwards/scs/v2"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgconn"

	"github.com/ifaisalabid1/notes-platform/internal/admin"
	"github.com/ifaisalabid1/notes-platform/internal/auth"
	"github.com/ifaisalabid1/notes-platform/internal/views"
)

type AdminManagementHandler struct {
	adminRepo      *admin.Repository
	sessionManager *scs.SessionManager
	renderer       *views.Renderer
}

func NewAdminManagementHandler(
	adminRepo *admin.Repository,
	sessionManager *scs.SessionManager,
	renderer *views.Renderer,
) *AdminManagementHandler {
	return &AdminManagementHandler{
		adminRepo:      adminRepo,
		sessionManager: sessionManager,
		renderer:       renderer,
	}
}

type AdminListPageData struct {
	Admins []admin.Admin
}

func (h *AdminManagementHandler) Index(w http.ResponseWriter, r *http.Request) {
	admins, err := h.adminRepo.List(r.Context())
	if err != nil {
		slog.Error("failed to list admins", "error", err)
		http.Error(w, "Failed to load admins", http.StatusInternalServerError)
		return
	}

	h.renderer.Render(w, r, "admin_users.tmpl", views.TemplateData{
		Title: "Admins",
		Data: AdminListPageData{
			Admins: admins,
		},
	})
}

func (h *AdminManagementHandler) Store(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		h.renderIndexWithError(w, r, "Invalid form submission.")
		return
	}

	name := strings.TrimSpace(r.PostForm.Get("name"))
	email := strings.TrimSpace(r.PostForm.Get("email"))
	password := r.PostForm.Get("password")

	if name == "" || email == "" || password == "" {
		h.renderIndexWithError(w, r, "Name, email, and password are required.")
		return
	}

	passwordHash, err := auth.HashPassword(password)
	if err != nil {
		h.renderIndexWithError(w, r, err.Error())
		return
	}

	createdByString := h.sessionManager.GetString(r.Context(), "admin_id")

	createdBy, err := uuid.Parse(createdByString)
	if err != nil {
		slog.Error("failed to parse logged-in admin id", "error", err)
		http.Error(w, "Invalid session", http.StatusUnauthorized)
		return
	}

	_, err = h.adminRepo.CreateAdmin(r.Context(), admin.CreateAdminParams{
		Name:         name,
		Email:        email,
		PasswordHash: passwordHash,
		CreatedBy:    createdBy,
	})
	if err != nil {
		if isUniqueViolation(err) {
			h.renderIndexWithError(w, r, "An admin with this email already exists.")
			return
		}

		slog.Error("failed to create admin", "error", err)
		h.renderIndexWithError(w, r, "Failed to create admin.")
		return
	}

	h.sessionManager.Put(r.Context(), "flash", "Admin created successfully.")

	http.Redirect(w, r, "/admin/users", http.StatusSeeOther)
}

func (h *AdminManagementHandler) renderIndexWithError(w http.ResponseWriter, r *http.Request, message string) {
	admins, err := h.adminRepo.List(r.Context())
	if err != nil {
		slog.Error("failed to list admins after form error", "error", err)
		http.Error(w, "Failed to load admins", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusUnprocessableEntity)

	h.renderer.Render(w, r, "admin_users.tmpl", views.TemplateData{
		Title: "Admins",
		Error: message,
		Data: AdminListPageData{
			Admins: admins,
		},
	})
}

func isUniqueViolation(err error) bool {
	var pgErr *pgconn.PgError

	if errors.As(err, &pgErr) {
		return pgErr.Code == "23505"
	}

	return false
}
