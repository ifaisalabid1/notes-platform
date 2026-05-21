package server

import (
	"net/http"
	"time"

	"github.com/alexedwards/scs/v2"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/justinas/nosurf"

	"github.com/ifaisalabid1/notes-platform/internal/academic"
	"github.com/ifaisalabid1/notes-platform/internal/admin"
	"github.com/ifaisalabid1/notes-platform/internal/auth"
	"github.com/ifaisalabid1/notes-platform/internal/database"
	"github.com/ifaisalabid1/notes-platform/internal/handlers"
	"github.com/ifaisalabid1/notes-platform/internal/views"
)

type Dependencies struct {
	DB             *database.DB
	SessionManager *scs.SessionManager
	Renderer       *views.Renderer
}

func NewRouter(deps Dependencies) http.Handler {
	r := chi.NewRouter()

	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	r.Use(middleware.Recoverer)
	r.Use(middleware.Timeout(60 * time.Second))

	healthHandler := handlers.NewHealthHandler(deps.DB)

	adminRepo := admin.NewRepository(deps.DB.Pool)
	classRepo := academic.NewClassRepository(deps.DB.Pool)

	adminAuthHandler := handlers.NewAdminAuthHandler(
		adminRepo,
		deps.SessionManager,
		deps.Renderer,
	)

	adminManagementHandler := handlers.NewAdminManagementHandler(
		adminRepo,
		deps.SessionManager,
		deps.Renderer,
	)

	adminClassHandler := handlers.NewAdminClassHandler(
		classRepo,
		deps.Renderer,
	)

	authMiddleware := auth.NewMiddleware(deps.SessionManager)

	r.Get("/healthz", healthHandler.Check)
	r.Get("/readyz", healthHandler.Ready)

	r.Get("/admin/login", adminAuthHandler.ShowLogin)
	r.Post("/admin/login", adminAuthHandler.Login)

	r.Group(func(r chi.Router) {
		r.Use(authMiddleware.RequireAdmin)

		r.Get("/admin/dashboard", adminAuthHandler.Dashboard)
		r.Post("/admin/logout", adminAuthHandler.Logout)

		r.Get("/admin/classes", adminClassHandler.Index)
		r.Post("/admin/classes", adminClassHandler.Store)

		r.Group(func(r chi.Router) {
			r.Use(authMiddleware.RequireOwner)

			r.Get("/admin/users", adminManagementHandler.Index)
			r.Post("/admin/users", adminManagementHandler.Store)
		})
	})

	return deps.SessionManager.LoadAndSave(noSurf(r))
}

func noSurf(next http.Handler) http.Handler {
	csrfHandler := nosurf.New(next)

	csrfHandler.SetBaseCookie(http.Cookie{
		HttpOnly: true,
		Path:     "/",
		SameSite: http.SameSiteLaxMode,
		Secure:   false,
	})

	return csrfHandler
}
