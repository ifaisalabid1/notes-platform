package auth

import (
	"net/http"

	"github.com/alexedwards/scs/v2"
)

type Middleware struct {
	sessionManager *scs.SessionManager
}

func NewMiddleware(sessionManager *scs.SessionManager) *Middleware {
	return &Middleware{
		sessionManager: sessionManager,
	}
}

func (m *Middleware) RequireAdmin(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		adminID := m.sessionManager.GetString(r.Context(), "admin_id")
		if adminID == "" {
			http.Redirect(w, r, "/admin/login", http.StatusSeeOther)
			return
		}

		next.ServeHTTP(w, r)
	})
}

func (m *Middleware) RequireOwner(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		isOwner := m.sessionManager.GetBool(r.Context(), "admin_is_owner")
		if !isOwner {
			http.Error(w, "Forbidden", http.StatusForbidden)
			return
		}

		next.ServeHTTP(w, r)
	})
}
