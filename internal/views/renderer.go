package views

import (
	"fmt"
	"html/template"
	"io/fs"
	"net/http"
	"path"

	"github.com/alexedwards/scs/v2"
	"github.com/justinas/nosurf"
)

type Renderer struct {
	templates      map[string]*template.Template
	sessionManager *scs.SessionManager
}

type TemplateData struct {
	Title           string
	Description     string
	CanonicalURL    string
	OGType          string
	CSRFToken       string
	IsAuthenticated bool
	Flash           string
	Error           string
	Data            any
}

func NewRenderer(sessionManager *scs.SessionManager, templateFS fs.FS) (*Renderer, error) {
	renderer := &Renderer{
		templates:      make(map[string]*template.Template),
		sessionManager: sessionManager,
	}

	pages, err := fs.Glob(templateFS, "templates/pages/*.tmpl")
	if err != nil {
		return nil, fmt.Errorf("glob page templates: %w", err)
	}

	for _, page := range pages {
		name := path.Base(page)

		tmpl, err := template.ParseFS(
			templateFS,
			"templates/layouts/base.tmpl",
			page,
		)
		if err != nil {
			return nil, fmt.Errorf("parse page template %s: %w", name, err)
		}

		renderer.templates[name] = tmpl
	}

	partials, err := fs.Glob(templateFS, "templates/partials/*.tmpl")
	if err != nil {
		return nil, fmt.Errorf("glob partial templates: %w", err)
	}

	for _, partial := range partials {
		name := path.Base(partial)

		tmpl, err := template.ParseFS(templateFS, partial)
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

	data = r.prepareTemplateData(req, data)

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

	data = r.prepareTemplateData(req, data)

	w.Header().Set("Content-Type", "text/html; charset=utf-8")

	if err := tmpl.ExecuteTemplate(w, "content", data); err != nil {
		http.Error(w, "render partial", http.StatusInternalServerError)
	}
}

func (r *Renderer) prepareTemplateData(req *http.Request, data TemplateData) TemplateData {
	if data.Title == "" {
		data.Title = "Notes Platform"
	}

	if data.Description == "" {
		data.Description = "Browse classroom notes, subjects, units, chapters, and study materials shared by your teacher."
	}

	if data.OGType == "" {
		data.OGType = "website"
	}

	if data.CanonicalURL == "" {
		scheme := "https"
		if req.TLS == nil {
			scheme = "http"
		}

		host := req.Host
		if host != "" {
			data.CanonicalURL = scheme + "://" + host + req.URL.Path
		}
	}

	data.CSRFToken = nosurf.Token(req)
	data.IsAuthenticated = r.sessionManager.Exists(req.Context(), "admin_id")
	data.Flash = r.sessionManager.PopString(req.Context(), "flash")

	return data
}
