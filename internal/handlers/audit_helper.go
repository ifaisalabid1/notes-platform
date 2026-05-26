package handlers

import (
	"log/slog"
	"net/http"

	"github.com/alexedwards/scs/v2"
	"github.com/google/uuid"

	"github.com/ifaisalabid1/notes-platform/internal/audit"
)

func adminIDFromSession(sessionManager *scs.SessionManager, r *http.Request) *uuid.UUID {
	adminIDValue := sessionManager.GetString(r.Context(), "admin_id")
	if adminIDValue == "" {
		return nil
	}

	adminID, err := uuid.Parse(adminIDValue)
	if err != nil {
		return nil
	}

	return &adminID
}

func writeAuditLog(
	r *http.Request,
	sessionManager *scs.SessionManager,
	auditRepo *audit.Repository,
	action string,
	entityType string,
	entityID *uuid.UUID,
	message string,
	metadata map[string]any,
) {
	if auditRepo == nil {
		return
	}

	if err := auditRepo.Create(r.Context(), audit.CreateLogParams{
		AdminID:    adminIDFromSession(sessionManager, r),
		Action:     action,
		EntityType: entityType,
		EntityID:   entityID,
		Message:    message,
		Metadata:   metadata,
		IPAddress:  clientIPForAudit(r),
		UserAgent:  r.UserAgent(),
	}); err != nil {
		slog.Error("failed to write audit log", "error", err)
	}
}

func clientIPForAudit(r *http.Request) string {
	if forwardedFor := r.Header.Get("X-Forwarded-For"); forwardedFor != "" {
		return forwardedFor
	}

	return r.RemoteAddr
}
