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
	urls := []sitemapURL{
		{
			Loc:        h.baseURL + "/",
			LastMod:    time.Now().UTC().Format("2006-01-02"),
			ChangeFreq: "daily",
			Priority:   "1.0",
		},
	}

	classes, err := h.publicRepo.PublishedClasses(r.Context())
	if err != nil {
		slog.Error("failed to load classes for sitemap", "error", err)
		http.Error(w, "Failed to generate sitemap", http.StatusInternalServerError)
		return
	}

	for _, classItem := range classes {
		classURL := fmt.Sprintf("%s/classes/%s", h.baseURL, classItem.Slug)

		urls = append(urls, sitemapURL{
			Loc:        classURL,
			LastMod:    classItem.UpdatedAt.UTC().Format("2006-01-02"),
			ChangeFreq: "weekly",
			Priority:   "0.8",
		})

		_, semesters, err := h.publicRepo.PublishedSemestersByClassSlug(r.Context(), classItem.Slug)
		if err != nil {
			slog.Error("failed to load semesters for sitemap", "class_slug", classItem.Slug, "error", err)
			continue
		}

		for _, semesterItem := range semesters {
			semesterURL := fmt.Sprintf(
				"%s/classes/%s/semesters/%s",
				h.baseURL,
				classItem.Slug,
				semesterItem.Slug,
			)

			urls = append(urls, sitemapURL{
				Loc:        semesterURL,
				LastMod:    semesterItem.UpdatedAt.UTC().Format("2006-01-02"),
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
				subjectURL := fmt.Sprintf(
					"%s/classes/%s/semesters/%s/subjects/%s",
					h.baseURL,
					classItem.Slug,
					semesterItem.Slug,
					subjectItem.Slug,
				)

				urls = append(urls, sitemapURL{
					Loc:        subjectURL,
					LastMod:    subjectItem.UpdatedAt.UTC().Format("2006-01-02"),
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
					unitURL := fmt.Sprintf(
						"%s/classes/%s/semesters/%s/subjects/%s/units/%s",
						h.baseURL,
						classItem.Slug,
						semesterItem.Slug,
						subjectItem.Slug,
						unitItem.Slug,
					)

					urls = append(urls, sitemapURL{
						Loc:        unitURL,
						LastMod:    unitItem.UpdatedAt.UTC().Format("2006-01-02"),
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
						chapterURL := fmt.Sprintf(
							"%s/classes/%s/semesters/%s/subjects/%s/units/%s/chapters/%s",
							h.baseURL,
							classItem.Slug,
							semesterItem.Slug,
							subjectItem.Slug,
							unitItem.Slug,
							chapterItem.Slug,
						)

						urls = append(urls, sitemapURL{
							Loc:        chapterURL,
							LastMod:    chapterItem.UpdatedAt.UTC().Format("2006-01-02"),
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
