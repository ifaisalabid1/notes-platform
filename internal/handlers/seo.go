package handlers

import (
	"encoding/xml"
	"fmt"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/ifaisalabid1/notes-platform/internal/academic"
)

type SEOHandler struct {
	publicRepo *academic.PublicRepository
	baseURL    string
}

func NewSEOHandler(publicRepo *academic.PublicRepository, baseURL string) *SEOHandler {
	return &SEOHandler{
		publicRepo: publicRepo,
		baseURL:    strings.TrimRight(baseURL, "/"),
	}
}

func (h *SEOHandler) Robots(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")

	_, _ = fmt.Fprintf(w, "User-agent: *\n")
	_, _ = fmt.Fprintf(w, "Allow: /\n")
	_, _ = fmt.Fprintf(w, "Disallow: /admin/\n")
	_, _ = fmt.Fprintf(w, "Disallow: /search\n")
	_, _ = fmt.Fprintf(w, "\n")
	_, _ = fmt.Fprintf(w, "Sitemap: %s/sitemap.xml\n", h.baseURL)
}

type sitemapURLSet struct {
	XMLName xml.Name     `xml:"urlset"`
	Xmlns   string       `xml:"xmlns,attr"`
	URLs    []sitemapURL `xml:"url"`
}

type sitemapURL struct {
	Loc        string `xml:"loc"`
	LastMod    string `xml:"lastmod,omitempty"`
	ChangeFreq string `xml:"changefreq,omitempty"`
	Priority   string `xml:"priority,omitempty"`
}

func (h *SEOHandler) Sitemap(w http.ResponseWriter, r *http.Request) {
	now := time.Now().UTC().Format("2006-01-02")

	urls := []sitemapURL{
		{
			Loc:        h.urlFor("/"),
			LastMod:    now,
			ChangeFreq: "daily",
			Priority:   "1.0",
		},
		{
			Loc:        h.urlFor("/about"),
			LastMod:    now,
			ChangeFreq: "monthly",
			Priority:   "0.4",
		},
		{
			Loc:        h.urlFor("/contact"),
			LastMod:    now,
			ChangeFreq: "monthly",
			Priority:   "0.4",
		},
		{
			Loc:        h.urlFor("/privacy"),
			LastMod:    now,
			ChangeFreq: "monthly",
			Priority:   "0.3",
		},
		{
			Loc:        h.urlFor("/terms"),
			LastMod:    now,
			ChangeFreq: "monthly",
			Priority:   "0.3",
		},
	}

	classes, err := h.publicRepo.PublishedClasses(r.Context())
	if err != nil {
		slog.Error("failed to load classes for sitemap", "error", err)
		http.Error(w, "Failed to generate sitemap", http.StatusInternalServerError)
		return
	}

	for _, classItem := range classes {
		classPath := fmt.Sprintf("/classes/%s", classItem.Slug)

		urls = append(urls, sitemapURL{
			Loc:        h.urlFor(classPath),
			LastMod:    lastModDate(classItem.UpdatedAt),
			ChangeFreq: "weekly",
			Priority:   "0.8",
		})

		_, semesters, err := h.publicRepo.PublishedSemestersByClassSlug(r.Context(), classItem.Slug)
		if err != nil {
			slog.Error("failed to load semesters for sitemap", "class_slug", classItem.Slug, "error", err)
			continue
		}

		for _, semesterItem := range semesters {
			semesterPath := fmt.Sprintf(
				"/classes/%s/semesters/%s",
				classItem.Slug,
				semesterItem.Slug,
			)

			urls = append(urls, sitemapURL{
				Loc:        h.urlFor(semesterPath),
				LastMod:    lastModDate(semesterItem.UpdatedAt),
				ChangeFreq: "weekly",
				Priority:   "0.7",
			})

			_, _, subjects, err := h.publicRepo.PublishedSubjects(
				r.Context(),
				classItem.Slug,
				semesterItem.Slug,
			)
			if err != nil {
				slog.Error(
					"failed to load subjects for sitemap",
					"class_slug", classItem.Slug,
					"semester_slug", semesterItem.Slug,
					"error", err,
				)
				continue
			}

			for _, subjectItem := range subjects {
				subjectPath := fmt.Sprintf(
					"/classes/%s/semesters/%s/subjects/%s",
					classItem.Slug,
					semesterItem.Slug,
					subjectItem.Slug,
				)

				urls = append(urls, sitemapURL{
					Loc:        h.urlFor(subjectPath),
					LastMod:    lastModDate(subjectItem.UpdatedAt),
					ChangeFreq: "weekly",
					Priority:   "0.7",
				})

				_, _, _, units, err := h.publicRepo.PublishedUnits(
					r.Context(),
					classItem.Slug,
					semesterItem.Slug,
					subjectItem.Slug,
				)
				if err != nil {
					slog.Error(
						"failed to load units for sitemap",
						"class_slug", classItem.Slug,
						"semester_slug", semesterItem.Slug,
						"subject_slug", subjectItem.Slug,
						"error", err,
					)
					continue
				}

				for _, unitItem := range units {
					unitPath := fmt.Sprintf(
						"/classes/%s/semesters/%s/subjects/%s/units/%s",
						classItem.Slug,
						semesterItem.Slug,
						subjectItem.Slug,
						unitItem.Slug,
					)

					urls = append(urls, sitemapURL{
						Loc:        h.urlFor(unitPath),
						LastMod:    lastModDate(unitItem.UpdatedAt),
						ChangeFreq: "weekly",
						Priority:   "0.6",
					})

					_, _, _, _, chapters, err := h.publicRepo.PublishedChapters(
						r.Context(),
						classItem.Slug,
						semesterItem.Slug,
						subjectItem.Slug,
						unitItem.Slug,
					)
					if err != nil {
						slog.Error(
							"failed to load chapters for sitemap",
							"class_slug", classItem.Slug,
							"semester_slug", semesterItem.Slug,
							"subject_slug", subjectItem.Slug,
							"unit_slug", unitItem.Slug,
							"error", err,
						)
						continue
					}

					for _, chapterItem := range chapters {
						chapterPath := fmt.Sprintf(
							"/classes/%s/semesters/%s/subjects/%s/units/%s/chapters/%s",
							classItem.Slug,
							semesterItem.Slug,
							subjectItem.Slug,
							unitItem.Slug,
							chapterItem.Slug,
						)

						urls = append(urls, sitemapURL{
							Loc:        h.urlFor(chapterPath),
							LastMod:    lastModDate(chapterItem.UpdatedAt),
							ChangeFreq: "weekly",
							Priority:   "0.6",
						})
					}
				}
			}
		}
	}

	payload := sitemapURLSet{
		Xmlns: "http://www.sitemaps.org/schemas/sitemap/0.9",
		URLs:  urls,
	}

	w.Header().Set("Content-Type", "application/xml; charset=utf-8")
	w.WriteHeader(http.StatusOK)

	_, _ = w.Write([]byte(xml.Header))

	if err := xml.NewEncoder(w).Encode(payload); err != nil {
		slog.Error("failed to encode sitemap", "error", err)
	}
}

func (h *SEOHandler) urlFor(path string) string {
	if h.baseURL == "" {
		return path
	}

	if !strings.HasPrefix(path, "/") {
		path = "/" + path
	}

	return h.baseURL + path
}

func lastModDate(t time.Time) string {
	if t.IsZero() {
		return time.Now().UTC().Format("2006-01-02")
	}

	return t.UTC().Format("2006-01-02")
}
