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
		Title: "About",
	})
}

func (h *StaticPageHandler) Contact(w http.ResponseWriter, r *http.Request) {
	h.renderer.Render(w, r, "contact.tmpl", views.TemplateData{
		Title: "Contact",
	})
}

func (h *StaticPageHandler) Privacy(w http.ResponseWriter, r *http.Request) {
	h.renderer.Render(w, r, "privacy.tmpl", views.TemplateData{
		Title: "Privacy Policy",
	})
}

func (h *StaticPageHandler) Terms(w http.ResponseWriter, r *http.Request) {
	h.renderer.Render(w, r, "terms.tmpl", views.TemplateData{
		Title: "Terms",
	})
}
