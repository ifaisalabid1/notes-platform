package main

import (
	"context"
	"log/slog"
	"os"
	"time"

	"github.com/ifaisalabid1/notes-platform/internal/admin"
	"github.com/ifaisalabid1/notes-platform/internal/auth"
	"github.com/ifaisalabid1/notes-platform/internal/database"

	"github.com/joho/godotenv"
)

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	slog.SetDefault(logger)

	_ = godotenv.Load()

	databaseURL := os.Getenv("DATABASE_URL")
	ownerName := os.Getenv("OWNER_NAME")
	ownerEmail := os.Getenv("OWNER_EMAIL")
	ownerPassword := os.Getenv("OWNER_PASSWORD")

	if databaseURL == "" {
		slog.Error("DATABASE_URL is required")
		os.Exit(1)
	}

	if ownerName == "" {
		slog.Error("OWNER_NAME is required")
		os.Exit(1)
	}

	if ownerEmail == "" {
		slog.Error("OWNER_EMAIL is required")
		os.Exit(1)
	}

	if ownerPassword == "" {
		slog.Error("OWNER_PASSWORD is required")
		os.Exit(1)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	db, err := database.New(ctx, databaseURL)
	if err != nil {
		slog.Error("failed to connect database", "error", err)
		os.Exit(1)
	}
	defer db.Close()

	adminRepo := admin.NewRepository(db.Pool)

	exists, err := adminRepo.OwnerExists(ctx)
	if err != nil {
		slog.Error("failed to check owner", "error", err)
		os.Exit(1)
	}

	if exists {
		slog.Info("owner already exists, skipping bootstrap")
		return
	}

	passwordHash, err := auth.HashPassword(ownerPassword)
	if err != nil {
		slog.Error("failed to hash owner password", "error", err)
		os.Exit(1)
	}

	createdOwner, err := adminRepo.CreateOwner(ctx, admin.CreateOwnerParams{
		Name:         ownerName,
		Email:        ownerEmail,
		PasswordHash: passwordHash,
	})
	if err != nil {
		slog.Error("failed to create owner", "error", err)
		os.Exit(1)
	}

	slog.Info(
		"owner created successfully",
		"admin_id", createdOwner.ID.String(),
		"email", createdOwner.Email,
	)
}
