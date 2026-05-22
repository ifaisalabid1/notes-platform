package views

import (
	"fmt"
	"html/template"
	"net/http"
	"path/filepath"

	"github.com/alexedwards/scs/v2"
	"github.com/justinas/nosurf"
)

type Renderer struct {
	templates      map[string]*template.Template
	sessionManager *scs.SessionManager
}

type TemplateData struct {
	Title           string
	CSRFToken       string
	IsAuthenticated bool
	Flash           string
	Error           string
	Data            any
}

func NewRenderer(sessionManager *scs.SessionManager) (*Renderer, error) {
	renderer := &Renderer{
		templates:      make(map[string]*template.Template),
		sessionManager: sessionManager,
	}

	pages, err := filepath.Glob("web/templates/pages/*.tmpl")
	if err != nil {
		return nil, fmt.Errorf("glob page templates: %w", err)
	}

	for _, page := range pages {
		name := filepath.Base(page)

		files := []string{
			"web/templates/layouts/base.tmpl",
			page,
		}

		tmpl, err := template.ParseFiles(files...)
		if err != nil {
			return nil, fmt.Errorf("parse page template %s: %w", name, err)
		}

		renderer.templates[name] = tmpl
	}

	partials, err := filepath.Glob("web/templates/partials/*.tmpl")
	if err != nil {
		return nil, fmt.Errorf("glob partial templates: %w", err)
	}

	for _, partial := range partials {
		name := filepath.Base(partial)

		tmpl, err := template.ParseFiles(partial)
		if err != nil {
			return nil, fmt.Errorf("parse partial template %s: %w", name, err)
		}

		renderer.templates[name] = tmpl
	}

	return renderer, nil
}

func (r *Renderer) Render(w http.ResponseWriter, req *http.Request, name string, data TemplateData) {
	tmpl, ok := r.templates[name]
	if !ok {
		http.Error(w, "template not found", http.StatusInternalServerError)
		return
	}

	data.CSRFToken = nosurf.Token(req)
	data.IsAuthenticated = r.sessionManager.Exists(req.Context(), "admin_id")
	data.Flash = r.sessionManager.PopString(req.Context(), "flash")

	w.Header().Set("Content-Type", "text/html; charset=utf-8")

	if err := tmpl.ExecuteTemplate(w, "base", data); err != nil {
		http.Error(w, "render template", http.StatusInternalServerError)
	}
}

func (r *Renderer) RenderPartial(w http.ResponseWriter, req *http.Request, name string, data TemplateData) {
	tmpl, ok := r.templates[name]
	if !ok {
		http.Error(w, "template not found", http.StatusInternalServerError)
		return
	}

	data.CSRFToken = nosurf.Token(req)
	data.IsAuthenticated = r.sessionManager.Exists(req.Context(), "admin_id")
	data.Flash = r.sessionManager.PopString(req.Context(), "flash")

	w.Header().Set("Content-Type", "text/html; charset=utf-8")

	if err := tmpl.ExecuteTemplate(w, "content", data); err != nil {
		http.Error(w, "render partial", http.StatusInternalServerError)
	}
}
