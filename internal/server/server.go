package server

import (
	"io/fs"
	"net/http"
	"time"

	"github.com/alexedwards/scs/v2"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/justinas/nosurf"

	"github.com/ifaisalabid1/notes-platform/internal/academic"
	"github.com/ifaisalabid1/notes-platform/internal/admin"
	"github.com/ifaisalabid1/notes-platform/internal/auth"
	"github.com/ifaisalabid1/notes-platform/internal/database"
	"github.com/ifaisalabid1/notes-platform/internal/fileproxy"
	"github.com/ifaisalabid1/notes-platform/internal/handlers"
	"github.com/ifaisalabid1/notes-platform/internal/ratelimit"
	"github.com/ifaisalabid1/notes-platform/internal/storage"
	"github.com/ifaisalabid1/notes-platform/internal/views"
	"github.com/ifaisalabid1/notes-platform/internal/watermark"
)

type Dependencies struct {
	DB               *database.DB
	R2               *storage.R2Client
	PDFWatermarker   *watermark.PDFWatermarker
	FileProxySigner  *fileproxy.Signer
	SessionManager   *scs.SessionManager
	Renderer         *views.Renderer
	EmbeddedFS       fs.FS
	LoginRateLimiter *ratelimit.IPLimiter
	AppBaseURL       string
}

func NewRouter(deps Dependencies) http.Handler {
	r := chi.NewRouter()

	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	r.Use(middleware.Recoverer)
	r.Use(securityHeaders)
	r.Use(middleware.Timeout(60 * time.Second))

	staticFS, err := fs.Sub(deps.EmbeddedFS, "static")
	if err != nil {
		panic("failed to create static fs: " + err.Error())
	}

	fileServer := http.FileServerFS(staticFS)
	r.Handle("/static/*", http.StripPrefix("/static/", fileServer))

	healthHandler := handlers.NewHealthHandler(deps.DB)

	adminRepo := admin.NewRepository(deps.DB.Pool)
	classRepo := academic.NewClassRepository(deps.DB.Pool)
	semesterRepo := academic.NewSemesterRepository(deps.DB.Pool)
	subjectRepo := academic.NewSubjectRepository(deps.DB.Pool)
	unitRepo := academic.NewUnitRepository(deps.DB.Pool)
	chapterRepo := academic.NewChapterRepository(deps.DB.Pool)
	noteRepo := academic.NewNoteRepository(deps.DB.Pool)

	publicRepo := academic.NewPublicRepository(deps.DB.Pool)

	adminAuthHandler := handlers.NewAdminAuthHandler(
		adminRepo,
		deps.SessionManager,
		deps.Renderer,
	)

	adminAccountHandler := handlers.NewAdminAccountHandler(
		adminRepo,
		deps.SessionManager,
		deps.Renderer,
	)

	adminManagementHandler := handlers.NewAdminManagementHandler(
		adminRepo,
		deps.SessionManager,
		deps.Renderer,
	)

	adminClassHandler := handlers.NewAdminClassHandler(
		classRepo,
		deps.Renderer,
	)

	adminSemesterHandler := handlers.NewAdminSemesterHandler(
		classRepo,
		semesterRepo,
		deps.Renderer,
	)

	adminSubjectHandler := handlers.NewAdminSubjectHandler(
		semesterRepo,
		subjectRepo,
		deps.Renderer,
	)

	adminUnitHandler := handlers.NewAdminUnitHandler(
		subjectRepo,
		unitRepo,
		deps.Renderer,
	)

	adminChapterHandler := handlers.NewAdminChapterHandler(
		unitRepo,
		chapterRepo,
		deps.Renderer,
	)

	adminNoteHandler := handlers.NewAdminNoteHandler(
		classRepo,
		chapterRepo,
		noteRepo,
		deps.R2,
		deps.PDFWatermarker,
		deps.FileProxySigner,
		deps.SessionManager,
		deps.Renderer,
	)

	adminStorageHandler := handlers.NewAdminStorageHandler(deps.R2)

	publicHandler := handlers.NewPublicHandler(
		publicRepo,
		deps.FileProxySigner,
		deps.Renderer,
	)

	seoHandler := handlers.NewSEOHandler(
		publicRepo,
		deps.AppBaseURL,
	)

	staticPageHandler := handlers.NewStaticPageHandler(deps.Renderer)

	adminHTMXHandler := handlers.NewAdminHTMXHandler(
		publicRepo,
		deps.Renderer,
	)

	errorHandler := handlers.NewErrorHandler(deps.Renderer)

	authMiddleware := auth.NewMiddleware(
		deps.SessionManager,
		errorHandler.Forbidden,
	)

	r.Get("/healthz", healthHandler.Check)
	r.Get("/readyz", healthHandler.Ready)

	r.Get("/", publicHandler.Home)
	r.Get("/robots.txt", seoHandler.Robots)
	r.Get("/sitemap.xml", seoHandler.Sitemap)

	r.Get("/about", staticPageHandler.About)
	r.Get("/contact", staticPageHandler.Contact)
	r.Get("/privacy", staticPageHandler.Privacy)
	r.Get("/terms", staticPageHandler.Terms)

	r.Get("/classes/{classSlug}", publicHandler.Semesters)
	r.Get("/classes/{classSlug}/semesters/{semesterSlug}", publicHandler.Subjects)
	r.Get("/classes/{classSlug}/semesters/{semesterSlug}/subjects/{subjectSlug}", publicHandler.Units)
	r.Get("/classes/{classSlug}/semesters/{semesterSlug}/subjects/{subjectSlug}/units/{unitSlug}", publicHandler.Chapters)
	r.Get("/classes/{classSlug}/semesters/{semesterSlug}/subjects/{subjectSlug}/units/{unitSlug}/chapters/{chapterSlug}", publicHandler.Notes)

	r.Get("/admin/login", adminAuthHandler.ShowLogin)
	r.With(ratelimit.Middleware(
		deps.LoginRateLimiter,
		errorHandler.TooManyRequests,
	)).Post("/admin/login", adminAuthHandler.Login)

	r.Group(func(r chi.Router) {
		r.Use(authMiddleware.RequireAdmin)

		r.Get("/admin/dashboard", adminAuthHandler.Dashboard)
		r.Post("/admin/logout", adminAuthHandler.Logout)
		r.Get("/admin/account/password", adminAccountHandler.ShowPassword)
		r.Post("/admin/account/password", adminAccountHandler.UpdatePassword)

		r.Get("/admin/classes", adminClassHandler.Index)
		r.Post("/admin/classes", adminClassHandler.Store)
		r.Get("/admin/classes/{classID}/edit", adminClassHandler.Edit)
		r.Post("/admin/classes/{classID}/edit", adminClassHandler.Update)

		r.Get("/admin/semesters", adminSemesterHandler.Index)
		r.Post("/admin/semesters", adminSemesterHandler.Store)
		r.Get("/admin/semesters/{semesterID}/edit", adminSemesterHandler.Edit)
		r.Post("/admin/semesters/{semesterID}/edit", adminSemesterHandler.Update)

		r.Get("/admin/subjects", adminSubjectHandler.Index)
		r.Post("/admin/subjects", adminSubjectHandler.Store)
		r.Get("/admin/subjects/{subjectID}/edit", adminSubjectHandler.Edit)
		r.Post("/admin/subjects/{subjectID}/edit", adminSubjectHandler.Update)

		r.Get("/admin/units", adminUnitHandler.Index)
		r.Post("/admin/units", adminUnitHandler.Store)
		r.Get("/admin/units/{unitID}/edit", adminUnitHandler.Edit)
		r.Post("/admin/units/{unitID}/edit", adminUnitHandler.Update)

		r.Get("/admin/chapters", adminChapterHandler.Index)
		r.Post("/admin/chapters", adminChapterHandler.Store)
		r.Get("/admin/chapters/{chapterID}/edit", adminChapterHandler.Edit)
		r.Post("/admin/chapters/{chapterID}/edit", adminChapterHandler.Update)

		r.Get("/admin/notes", adminNoteHandler.Index)
		r.Post("/admin/notes", adminNoteHandler.Store)
		r.Get("/admin/notes/{noteID}/edit", adminNoteHandler.Edit)
		r.Post("/admin/notes/{noteID}/edit", adminNoteHandler.Update)
		r.Post("/admin/notes/{noteID}/archive", adminNoteHandler.Archive)
		r.Post("/admin/notes/{noteID}/unarchive", adminNoteHandler.Unarchive)

		r.Get("/admin/htmx/semesters", adminHTMXHandler.SemestersByClass)
		r.Get("/admin/htmx/subjects", adminHTMXHandler.SubjectsBySemester)
		r.Get("/admin/htmx/units", adminHTMXHandler.UnitsBySubject)
		r.Get("/admin/htmx/chapters", adminHTMXHandler.ChaptersByUnit)

		r.Get("/admin/storage/readyz", adminStorageHandler.Ready)

		r.Group(func(r chi.Router) {
			r.Use(authMiddleware.RequireOwner)

			r.Get("/admin/users", adminManagementHandler.Index)
			r.Post("/admin/users", adminManagementHandler.Store)
		})
	})

	r.NotFound(errorHandler.NotFound)

	return deps.SessionManager.LoadAndSave(noSurf(r))
}

func noSurf(next http.Handler) http.Handler {
	csrfHandler := nosurf.New(next)

	csrfHandler.SetBaseCookie(http.Cookie{
		HttpOnly: true,
		Path:     "/",
		SameSite: http.SameSiteLaxMode,
		Secure:   true,
	})

	return csrfHandler
}
