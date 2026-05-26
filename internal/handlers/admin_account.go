package handlers

import (
	"errors"
	"log/slog"
	"net/http"

	"github.com/alexedwards/scs/v2"
	"github.com/google/uuid"

	"github.com/ifaisalabid1/notes-platform/internal/admin"
	"github.com/ifaisalabid1/notes-platform/internal/audit"
	"github.com/ifaisalabid1/notes-platform/internal/auth"
	"github.com/ifaisalabid1/notes-platform/internal/views"
)

type AdminAccountHandler struct {
	adminRepo      *admin.Repository
	sessionManager *scs.SessionManager
	auditRepo      *audit.Repository
	renderer       *views.Renderer
}

func NewAdminAccountHandler(
	adminRepo *admin.Repository,
	sessionManager *scs.SessionManager,
	auditRepo *audit.Repository,
	renderer *views.Renderer,
) *AdminAccountHandler {
	return &AdminAccountHandler{
		adminRepo:      adminRepo,
		sessionManager: sessionManager,
		auditRepo:      auditRepo,
		renderer:       renderer,
	}
}

func (h *AdminAccountHandler) ShowPassword(w http.ResponseWriter, r *http.Request) {
	h.renderer.Render(w, r, "admin_password.tmpl", views.TemplateData{
		Title: "Change Password",
	})
}

func (h *AdminAccountHandler) UpdatePassword(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		h.renderPasswordError(w, r, "Invalid form submission.")
		return
	}

	currentPassword := r.PostForm.Get("current_password")
	newPassword := r.PostForm.Get("new_password")
	confirmPassword := r.PostForm.Get("confirm_password")

	if currentPassword == "" || newPassword == "" || confirmPassword == "" {
		h.renderPasswordError(w, r, "All password fields are required.")
		return
	}

	if newPassword != confirmPassword {
		h.renderPasswordError(w, r, "New password and confirmation do not match.")
		return
	}

	adminIDValue := h.sessionManager.GetString(r.Context(), "admin_id")
	adminEmail := h.sessionManager.GetString(r.Context(), "admin_email")

	adminID, err := uuid.Parse(adminIDValue)
	if err != nil {
		http.Error(w, "Invalid session", http.StatusUnauthorized)
		return
	}

	foundAdmin, err := h.adminRepo.FindByEmail(r.Context(), adminEmail)
	if err != nil {
		if errors.Is(err, admin.ErrAdminNotFound) {
			http.Error(w, "Invalid session", http.StatusUnauthorized)
			return
		}

		slog.Error("failed to load admin for password change", "error", err)
		h.renderPasswordError(w, r, "Something went wrong. Please try again.")
		return
	}

	if foundAdmin.ID != adminID {
		http.Error(w, "Invalid session", http.StatusUnauthorized)
		return
	}

	matches, err := auth.VerifyPassword(currentPassword, foundAdmin.PasswordHash)
	if err != nil {
		slog.Error("failed to verify current password", "error", err)
		h.renderPasswordError(w, r, "Something went wrong. Please try again.")
		return
	}

	if !matches {
		h.renderPasswordError(w, r, "Current password is incorrect.")
		return
	}

	newPasswordHash, err := auth.HashPassword(newPassword)
	if err != nil {
		h.renderPasswordError(w, r, err.Error())
		return
	}

	if err := h.adminRepo.UpdatePassword(r.Context(), adminID, newPasswordHash); err != nil {
		if errors.Is(err, admin.ErrAdminNotFound) {
			http.Error(w, "Invalid session", http.StatusUnauthorized)
			return
		}

		slog.Error("failed to update admin password", "error", err)
		h.renderPasswordError(w, r, "Failed to update password.")
		return
	}

	if err := h.sessionManager.RenewToken(r.Context()); err != nil {
		slog.Error("failed to renew session after password change", "error", err)
		http.Error(w, "Password changed, but session renewal failed.", http.StatusInternalServerError)
		return
	}

	h.sessionManager.Put(r.Context(), "flash", "Password changed successfully.")

	http.Redirect(w, r, "/admin/dashboard", http.StatusSeeOther)
}

func (h *AdminAccountHandler) renderPasswordError(w http.ResponseWriter, r *http.Request, message string) {
	w.WriteHeader(http.StatusUnprocessableEntity)

	h.renderer.Render(w, r, "admin_password.tmpl", views.TemplateData{
		Title: "Change Password",
		Error: message,
	})
}
