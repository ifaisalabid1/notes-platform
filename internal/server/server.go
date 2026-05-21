package server

import (
	"net/http"
	"time"

	"github.com/ifaisalabid1/notes-platform/internal/database"
	"github.com/ifaisalabid1/notes-platform/internal/handlers"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
)

type Dependencies struct {
	DB *database.DB
}

func NewRouter(deps Dependencies) http.Handler {
	r := chi.NewRouter()

	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	r.Use(middleware.Recoverer)
	r.Use(middleware.Timeout(60 * time.Second))

	healthHandler := handlers.NewHealthHandler(deps.DB)

	r.Get("/healthz", healthHandler.Check)
	r.Get("/readyz", healthHandler.Ready)

	return r
}
