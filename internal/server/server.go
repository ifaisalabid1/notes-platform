package server

import (
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
	"github.com/ifaisalabid1/notes-platform/internal/storage"
	"github.com/ifaisalabid1/notes-platform/internal/views"
	"github.com/ifaisalabid1/notes-platform/internal/watermark"
)

type Dependencies struct {
	DB              *database.DB
	R2              *storage.R2Client
	PDFWatermarker  *watermark.PDFWatermarker
	FileProxySigner *fileproxy.Signer
	SessionManager  *scs.SessionManager
	Renderer        *views.Renderer
}

func NewRouter(deps Dependencies) http.Handler {
	r := chi.NewRouter()

	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	r.Use(middleware.Recoverer)
	r.Use(middleware.Timeout(60 * time.Second))

	fileServer := http.FileServer(http.Dir("web/static"))
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

	adminHTMXHandler := handlers.NewAdminHTMXHandler(
		publicRepo,
		deps.Renderer,
	)

	authMiddleware := auth.NewMiddleware(deps.SessionManager)

	r.Get("/healthz", healthHandler.Check)
	r.Get("/readyz", healthHandler.Ready)

	r.Get("/", publicHandler.Home)
	r.Get("/classes/{classSlug}", publicHandler.Semesters)
	r.Get("/classes/{classSlug}/semesters/{semesterSlug}", publicHandler.Subjects)
	r.Get("/classes/{classSlug}/semesters/{semesterSlug}/subjects/{subjectSlug}", publicHandler.Units)
	r.Get("/classes/{classSlug}/semesters/{semesterSlug}/subjects/{subjectSlug}/units/{unitSlug}", publicHandler.Chapters)
	r.Get("/classes/{classSlug}/semesters/{semesterSlug}/subjects/{subjectSlug}/units/{unitSlug}/chapters/{chapterSlug}", publicHandler.Notes)

	r.Get("/admin/login", adminAuthHandler.ShowLogin)
	r.Post("/admin/login", adminAuthHandler.Login)

	r.Group(func(r chi.Router) {
		r.Use(authMiddleware.RequireAdmin)

		r.Get("/admin/dashboard", adminAuthHandler.Dashboard)
		r.Post("/admin/logout", adminAuthHandler.Logout)

		r.Get("/admin/classes", adminClassHandler.Index)
		r.Post("/admin/classes", adminClassHandler.Store)
		r.Get("/admin/classes/{classID}/edit", adminClassHandler.Edit)
		r.Post("/admin/classes/{classID}/edit", adminClassHandler.Update)

		r.Get("/admin/semesters", adminSemesterHandler.Index)
		r.Post("/admin/semesters", adminSemesterHandler.Store)

		r.Get("/admin/subjects", adminSubjectHandler.Index)
		r.Post("/admin/subjects", adminSubjectHandler.Store)

		r.Get("/admin/units", adminUnitHandler.Index)
		r.Post("/admin/units", adminUnitHandler.Store)

		r.Get("/admin/chapters", adminChapterHandler.Index)
		r.Post("/admin/chapters", adminChapterHandler.Store)

		r.Get("/admin/notes", adminNoteHandler.Index)
		r.Post("/admin/notes", adminNoteHandler.Store)

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

	return deps.SessionManager.LoadAndSave(noSurf(r))
}

func noSurf(next http.Handler) http.Handler {
	csrfHandler := nosurf.New(next)

	csrfHandler.SetBaseCookie(http.Cookie{
		HttpOnly: true,
		Path:     "/",
		SameSite: http.SameSiteLaxMode,
		Secure:   false,
	})

	return csrfHandler
}
