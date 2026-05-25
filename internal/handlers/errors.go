package handlers

import (
	"log/slog"
	"net/http"

	"github.com/ifaisalabid1/notes-platform/internal/views"
)

type ErrorHandler struct {
	renderer *views.Renderer
}

type ErrorPageData struct {
	StatusCode int
	Heading    string
	Message    string
	ActionText string
	ActionURL  string
}

func NewErrorHandler(renderer *views.Renderer) *ErrorHandler {
	return &ErrorHandler{
		renderer: renderer,
	}
}

func (h *ErrorHandler) NotFound(w http.ResponseWriter, r *http.Request) {
	h.Render(w, r, http.StatusNotFound, ErrorPageData{
		StatusCode: http.StatusNotFound,
		Heading:    "Page not found",
		Message:    "The page you are looking for does not exist or may have been moved.",
		ActionText: "Go to homepage",
		ActionURL:  "/",
	})
}

func (h *ErrorHandler) Forbidden(w http.ResponseWriter, r *http.Request) {
	h.Render(w, r, http.StatusForbidden, ErrorPageData{
		StatusCode: http.StatusForbidden,
		Heading:    "Access denied",
		Message:    "You do not have permission to access this page.",
		ActionText: "Go to dashboard",
		ActionURL:  "/admin/dashboard",
	})
}

func (h *ErrorHandler) TooManyRequests(w http.ResponseWriter, r *http.Request) {
	h.Render(w, r, http.StatusTooManyRequests, ErrorPageData{
		StatusCode: http.StatusTooManyRequests,
		Heading:    "Too many requests",
		Message:    "Please wait a moment before trying again.",
		ActionText: "Back to login",
		ActionURL:  "/admin/login",
	})
}

func (h *ErrorHandler) InternalServerError(w http.ResponseWriter, r *http.Request) {
	h.Render(w, r, http.StatusInternalServerError, ErrorPageData{
		StatusCode: http.StatusInternalServerError,
		Heading:    "Something went wrong",
		Message:    "The server could not complete your request. Please try again later.",
		ActionText: "Go to homepage",
		ActionURL:  "/",
	})
}

func (h *ErrorHandler) Render(w http.ResponseWriter, r *http.Request, status int, data ErrorPageData) {
	w.WriteHeader(status)

	h.renderer.Render(w, r, "error.tmpl", views.TemplateData{
		Title: data.Heading,
		Data:  data,
	})
}

func RenderInternalServerError(renderer *views.Renderer, w http.ResponseWriter, r *http.Request, message string, err error) {
	if err != nil {
		slog.Error(message, "error", err)
	}

	NewErrorHandler(renderer).InternalServerError(w, r)
}
