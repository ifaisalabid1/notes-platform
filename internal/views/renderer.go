package views

import (
	"fmt"
	"html/template"
	"io/fs"
	"net/http"
	"path"
	"strings"

	"github.com/alexedwards/scs/v2"
	"github.com/justinas/nosurf"
)

func templateFunctions() template.FuncMap {
	return template.FuncMap{
		"isActivePath": func(currentPath string, targetPath string) bool {
			if targetPath == "/" {
				return currentPath == "/"
			}

			return currentPath == targetPath || strings.HasPrefix(currentPath, targetPath+"/")
		},
	}
}

type Renderer struct {
	templates      map[string]*template.Template
	sessionManager *scs.SessionManager
	baseURL        string
}

type TemplateData struct {
	Title           string
	Description     string
	CanonicalURL    string
	OGType          string
	Styles          []string
	Scripts         []string
	CurrentPath     string
	CSRFToken       string
	IsAuthenticated bool
	Flash           string
	Error           string
	Data            any
}

func NewRenderer(sessionManager *scs.SessionManager, templateFS fs.FS, baseURL string) (*Renderer, error) {
	renderer := &Renderer{
		templates:      make(map[string]*template.Template),
		sessionManager: sessionManager,
		baseURL:        strings.TrimRight(baseURL, "/"),
	}

	pages, err := fs.Glob(templateFS, "templates/pages/*.tmpl")
	if err != nil {
		return nil, fmt.Errorf("glob page templates: %w", err)
	}

	for _, page := range pages {
		name := path.Base(page)

		tmpl, err := template.New(path.Base(page)).
			Funcs(templateFunctions()).
			ParseFS(
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

		tmpl, err := template.New(path.Base(partial)).
			Funcs(templateFunctions()).
			ParseFS(templateFS, partial)
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
		data.Title = "Rising Star"
	}

	if data.Description == "" {
		data.Description = "Browse classroom notes, subjects, units, chapters, and study materials shared by your teacher."
	}

	if data.OGType == "" {
		data.OGType = "website"
	}

	if data.CanonicalURL == "" && r.baseURL != "" {
		data.CanonicalURL = r.baseURL + req.URL.Path
	}

	data.CurrentPath = req.URL.Path
	data.CSRFToken = nosurf.Token(req)
	data.IsAuthenticated = r.sessionManager.Exists(req.Context(), "admin_id")
	data.Flash = r.sessionManager.PopString(req.Context(), "flash")

	return data
}
