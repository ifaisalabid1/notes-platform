package main

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"time"

	"github.com/ifaisalabid1/notes-platform/migrations"

	"github.com/joho/godotenv"
	"github.com/pressly/goose/v3"

	_ "github.com/jackc/pgx/v5/stdlib"
)

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	slog.SetDefault(logger)

	_ = godotenv.Load()

	command := "up"
	if len(os.Args) > 1 {
		command = os.Args[1]
	}

	databaseURL := os.Getenv("DATABASE_URL")
	if databaseURL == "" {
		slog.Error("DATABASE_URL is required")
		os.Exit(1)
	}

	db, err := sql.Open("pgx", databaseURL)
	if err != nil {
		slog.Error("failed to open database", "error", err)
		os.Exit(1)
	}
	defer db.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := db.PingContext(ctx); err != nil {
		slog.Error("failed to ping database", "error", err)
		os.Exit(1)
	}

	goose.SetBaseFS(migrations.FS)

	if err := goose.SetDialect("postgres"); err != nil {
		slog.Error("failed to set goose dialect", "error", err)
		os.Exit(1)
	}

	if err := run(command, db); err != nil {
		slog.Error("migration failed", "command", command, "error", err)
		os.Exit(1)
	}

	slog.Info("migration completed", "command", command)
}

func run(command string, db *sql.DB) error {
	const migrationsDir = "."

	switch command {
	case "up":
		return goose.Up(db, migrationsDir)

	case "up-by-one":
		return goose.UpByOne(db, migrationsDir)

	case "down":
		return goose.Down(db, migrationsDir)

	case "status":
		return goose.Status(db, migrationsDir)

	case "version":
		version, err := goose.GetDBVersion(db)
		if err != nil {
			return err
		}

		fmt.Println(version)
		return nil

	default:
		return errors.New("unknown migration command: use up, up-by-one, down, status, or version")
	}
}
