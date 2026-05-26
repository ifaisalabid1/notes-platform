package server

import (
	"net/http"
	"strings"
)

func maintenanceMode(enabled bool, maintenanceHandler http.HandlerFunc) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if !enabled {
				next.ServeHTTP(w, r)
				return
			}

			path := r.URL.Path

			if isMaintenanceAllowedPath(path) {
				next.ServeHTTP(w, r)
				return
			}

			maintenanceHandler(w, r)
		})
	}
}

func isMaintenanceAllowedPath(path string) bool {
	if path == "/healthz" || path == "/readyz" {
		return true
	}

	if strings.HasPrefix(path, "/admin") {
		return true
	}

	if strings.HasPrefix(path, "/static/") {
		return true
	}

	return false
}
