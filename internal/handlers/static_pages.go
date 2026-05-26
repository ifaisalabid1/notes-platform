package handlers

import (
	"net/http"

	"github.com/ifaisalabid1/notes-platform/internal/views"
)

type StaticPageHandler struct {
	renderer *views.Renderer
}

func NewStaticPageHandler(renderer *views.Renderer) *StaticPageHandler {
	return &StaticPageHandler{
		renderer: renderer,
	}
}

func (h *StaticPageHandler) About(w http.ResponseWriter, r *http.Request) {
	h.renderer.Render(w, r, "about.tmpl", views.TemplateData{
		Title:       "About",
		Description: "Learn about Notes Platform and how it helps students access classroom notes and study materials.",
	})
}

func (h *StaticPageHandler) Contact(w http.ResponseWriter, r *http.Request) {
	h.renderer.Render(w, r, "contact.tmpl", views.TemplateData{
		Title:       "Contact",
		Description: "Contact Notes Platform for questions, corrections, and support related to classroom notes.",
	})
}

func (h *StaticPageHandler) Privacy(w http.ResponseWriter, r *http.Request) {
	h.renderer.Render(w, r, "privacy.tmpl", views.TemplateData{
		Title:       "Privacy Policy",
		Description: "Read the Notes Platform privacy policy and learn how basic information, cookies, and admin account data are handled.",
	})
}

func (h *StaticPageHandler) Terms(w http.ResponseWriter, r *http.Request) {
	h.renderer.Render(w, r, "terms.tmpl", views.TemplateData{
		Title:       "Terms of Use",
		Description: "Read the Terms of Use for Notes Platform educational content and classroom materials.",
	})
}
