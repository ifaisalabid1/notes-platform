package main

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/alexedwards/scs/pgxstore"
	"github.com/alexedwards/scs/v2"
	"golang.org/x/time/rate"

	"github.com/ifaisalabid1/notes-platform/internal/config"
	"github.com/ifaisalabid1/notes-platform/internal/database"
	"github.com/ifaisalabid1/notes-platform/internal/fileproxy"
	"github.com/ifaisalabid1/notes-platform/internal/ratelimit"
	"github.com/ifaisalabid1/notes-platform/internal/server"
	"github.com/ifaisalabid1/notes-platform/internal/storage"
	"github.com/ifaisalabid1/notes-platform/internal/views"
	"github.com/ifaisalabid1/notes-platform/internal/watermark"
	webassets "github.com/ifaisalabid1/notes-platform/web"
)

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	slog.SetDefault(logger)

	cfg, err := config.Load()
	if err != nil {
		slog.Error("failed to load config", "error", err)
		os.Exit(1)
	}

	ctx := context.Background()

	db, err := database.New(ctx, cfg.DatabaseURL)
	if err != nil {
		slog.Error("failed to connect database", "error", err)
		os.Exit(1)
	}
	defer db.Close()

	r2Client, err := storage.NewR2Client(ctx, storage.R2Config{
		AccountID:       cfg.R2AccountID,
		AccessKeyID:     cfg.R2AccessKeyID,
		SecretAccessKey: cfg.R2SecretAccessKey,
		BucketName:      cfg.R2BucketName,
		Endpoint:        cfg.R2Endpoint(),
	})
	if err != nil {
		slog.Error("failed to create r2 client", "error", err)
		os.Exit(1)
	}

	pdfWatermarker, err := watermark.NewPDFWatermarker(cfg.PDFWatermarkText())
	if err != nil {
		slog.Error("failed to create pdf watermarker", "error", err)
		os.Exit(1)
	}

	fileProxySigner, err := fileproxy.NewSigner(fileproxy.Config{
		BaseURL: cfg.FileProxyBaseURL,
		Secret:  cfg.FileProxySecret,
		TTL:     time.Duration(cfg.FileProxyURLTTLSeconds) * time.Second,
	})
	if err != nil {
		slog.Error("failed to create file proxy signer", "error", err)
		os.Exit(1)
	}

	sessionManager := scs.New()
	sessionManager.Store = pgxstore.New(db.Pool)
	sessionManager.Lifetime = 24 * time.Hour
	sessionManager.Cookie.Name = "notes_platform_session"
	sessionManager.Cookie.HttpOnly = true
	sessionManager.Cookie.Persist = true
	sessionManager.Cookie.SameSite = http.SameSiteLaxMode
	sessionManager.Cookie.Secure = cfg.IsProduction()

	renderer, err := views.NewRenderer(sessionManager, webassets.FS, cfg.AppBaseURL)
	if err != nil {
		slog.Error("failed to create template renderer", "error", err)
		os.Exit(1)
	}

	loginRateLimiter := ratelimit.NewIPLimiter(
		rate.Every(10*time.Second),
		5,
		30*time.Minute,
	)

	router := server.NewRouter(server.Dependencies{
		DB:               db,
		R2:               r2Client,
		PDFWatermarker:   pdfWatermarker,
		FileProxySigner:  fileProxySigner,
		SessionManager:   sessionManager,
		Renderer:         renderer,
		EmbeddedFS:       webassets.FS,
		LoginRateLimiter: loginRateLimiter,
		AppBaseURL:       cfg.AppBaseURL,
	})

	httpServer := &http.Server{
		Addr:         cfg.Addr(),
		Handler:      router,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 30 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	serverErrors := make(chan error, 1)

	go func() {
		slog.Info("starting server", "addr", cfg.Addr(), "env", cfg.AppEnv)

		if err := httpServer.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			serverErrors <- err
		}
	}()

	shutdown := make(chan os.Signal, 1)
	signal.Notify(shutdown, os.Interrupt, syscall.SIGTERM)

	select {
	case err := <-serverErrors:
		slog.Error("server error", "error", err)
		os.Exit(1)

	case sig := <-shutdown:
		slog.Info("shutdown started", "signal", sig.String())

		shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		if err := httpServer.Shutdown(shutdownCtx); err != nil {
			slog.Error("graceful shutdown failed", "error", err)

			if closeErr := httpServer.Close(); closeErr != nil {
				slog.Error("forced shutdown failed", "error", closeErr)
			}

			os.Exit(1)
		}

		slog.Info("shutdown complete")
	}
}
