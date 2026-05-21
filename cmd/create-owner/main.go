package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"log/slog"
	"os"
	"syscall"
	"time"

	"github.com/ifaisalabid1/notes-platform/internal/admin"
	"github.com/ifaisalabid1/notes-platform/internal/auth"
	"github.com/ifaisalabid1/notes-platform/internal/config"
	"github.com/ifaisalabid1/notes-platform/internal/database"

	"golang.org/x/term"
)

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	slog.SetDefault(logger)

	var name string
	var email string
	var password string

	flag.StringVar(&name, "name", "", "owner name")
	flag.StringVar(&email, "email", "", "owner email")
	flag.StringVar(&password, "password", "", "owner password")
	flag.Parse()

	if name == "" {
		slog.Error("name is required")
		os.Exit(1)
	}

	if email == "" {
		slog.Error("email is required")
		os.Exit(1)
	}

	if password == "" {
		fmt.Print("Enter owner password: ")

		passwordBytes, err := term.ReadPassword(int(syscall.Stdin))
		fmt.Println()

		if err != nil {
			slog.Error("failed to read password", "error", err)
			os.Exit(1)
		}

		password = string(passwordBytes)
	}

	cfg, err := config.Load()
	if err != nil {
		slog.Error("failed to load config", "error", err)
		os.Exit(1)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	db, err := database.New(ctx, cfg.DatabaseURL)
	if err != nil {
		slog.Error("failed to connect database", "error", err)
		os.Exit(1)
	}
	defer db.Close()

	passwordHash, err := auth.HashPassword(password)
	if err != nil {
		slog.Error("failed to hash password", "error", err)
		os.Exit(1)
	}

	adminRepo := admin.NewRepository(db.Pool)

	created, err := adminRepo.CreateOwner(ctx, admin.CreateOwnerParams{
		Name:         name,
		Email:        email,
		PasswordHash: passwordHash,
	})
	if err != nil {
		if errors.Is(err, admin.ErrOwnerAlreadyExists) {
			slog.Error("owner already exists")
			os.Exit(1)
		}

		slog.Error("failed to create owner", "error", err)
		os.Exit(1)
	}

	slog.Info(
		"owner created",
		"id", created.ID.String(),
		"name", created.Name,
		"email", created.Email,
		"role", created.Role,
	)
}
