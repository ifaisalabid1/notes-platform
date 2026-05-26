package server

import (
	"log/slog"
	"net/http"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5/middleware"
)

type responseRecorder struct {
	http.ResponseWriter
	statusCode int
	bytes      int
}

func newResponseRecorder(w http.ResponseWriter) *responseRecorder {
	return &responseRecorder{
		ResponseWriter: w,
		statusCode:     http.StatusOK,
	}
}

func (r *responseRecorder) WriteHeader(statusCode int) {
	r.statusCode = statusCode
	r.ResponseWriter.WriteHeader(statusCode)
}

func (r *responseRecorder) Write(data []byte) (int, error) {
	n, err := r.ResponseWriter.Write(data)
	r.bytes += n

	return n, err
}

func requestLogger(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		startedAt := time.Now()

		recorder := newResponseRecorder(w)

		next.ServeHTTP(recorder, r)

		duration := time.Since(startedAt)

		requestID := middleware.GetReqID(r.Context())

		slog.Info(
			"http_request",
			"request_id", requestID,
			"method", r.Method,
			"path", r.URL.Path,
			"query", r.URL.RawQuery,
			"status", recorder.statusCode,
			"status_class", statusClass(recorder.statusCode),
			"bytes", recorder.bytes,
			"duration_ms", float64(duration.Microseconds())/1000,
			"remote_ip", clientIPFromRequest(r),
			"user_agent", r.UserAgent(),
			"referer", r.Referer(),
		)
	})
}

func clientIPFromRequest(r *http.Request) string {
	if forwardedFor := r.Header.Get("X-Forwarded-For"); forwardedFor != "" {
		return forwardedFor
	}

	return r.RemoteAddr
}

func statusClass(statusCode int) string {
	switch {
	case statusCode >= 500:
		return "5xx"
	case statusCode >= 400:
		return "4xx"
	case statusCode >= 300:
		return "3xx"
	case statusCode >= 200:
		return "2xx"
	default:
		return strconv.Itoa(statusCode)
	}
}
