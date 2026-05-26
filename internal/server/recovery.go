package server

import (
	"log/slog"
	"net/http"
	"runtime/debug"

	"github.com/go-chi/chi/v5/middleware"
)

func recoverer(internalServerError http.HandlerFunc) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			defer func() {
				recoveredValue := recover()
				if recoveredValue == nil {
					return
				}

				requestID := middleware.GetReqID(r.Context())

				slog.Error(
					"panic recovered",
					"request_id", requestID,
					"method", r.Method,
					"path", r.URL.Path,
					"query", r.URL.RawQuery,
					"remote_ip", clientIPFromRequest(r),
					"user_agent", r.UserAgent(),
					"panic", recoveredValue,
					"stack", string(debug.Stack()),
				)

				internalServerError(w, r)
			}()

			next.ServeHTTP(w, r)
		})
	}
}
