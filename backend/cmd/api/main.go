package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"github.com/Swek09/lct-fcp-2026/backend/internal/app"
	"github.com/Swek09/lct-fcp-2026/backend/migrations"
	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/pressly/goose/v3"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"
)

func main() {
	if err := run(); err != nil {
		slog.Error("server stopped", "error", err)
		os.Exit(1)
	}
}
func run() error {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	cfg := app.EnvConfig()
	if destination := os.Getenv("EXPORT_OPENAPI"); destination != "" {
		// Route registration builds schemas without connecting to storage.
		a := &app.App{Config: cfg}
		a.Router()
		var raw []byte
		var err error
		if filepath.Ext(destination) == ".json" {
			raw, err = json.MarshalIndent(a.API.OpenAPI(), "", "  ")
		} else {
			raw, err = a.API.OpenAPI().YAML()
		}
		if err != nil {
			return err
		}
		return os.WriteFile(destination, raw, 0644)
	}
	database, err := sql.Open("pgx", cfg.DatabaseURL)
	if err != nil {
		return err
	}
	goose.SetBaseFS(migrations.Files)
	if err = goose.SetDialect("postgres"); err != nil {
		return err
	}
	if err = goose.UpContext(ctx, database, "."); err != nil {
		return err
	}
	_ = database.Close()
	a, err := app.New(ctx, cfg)
	if err != nil {
		return err
	}
	defer a.Close()
	if cfg.Demo {
		if err = a.Seed(ctx); err != nil {
			return err
		}
	}
	handler := a.Router()
	server := &http.Server{Addr: ":8000", Handler: handler, ReadHeaderTimeout: 5 * time.Second, ReadTimeout: 15 * time.Second, WriteTimeout: 30 * time.Second, IdleTimeout: 60 * time.Second}
	go a.MailLoop(ctx)
	errCh := make(chan error, 1)
	go func() { slog.Info("API ready", "addr", server.Addr); errCh <- server.ListenAndServe() }()
	select {
	case err = <-errCh:
		if err != http.ErrServerClosed {
			return err
		}
	case <-ctx.Done():
	}
	shutdown, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	return server.Shutdown(shutdown)
}
