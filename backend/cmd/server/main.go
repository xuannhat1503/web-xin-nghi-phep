// Command server runs the HTTP API (and, from later phases, the background workers).
package main

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"
	_ "time/tzdata" // Windows has no system zoneinfo

	"github.com/g-asiapacific/elm/backend/internal/analytics"
	"github.com/g-asiapacific/elm/backend/internal/applog"
	"github.com/g-asiapacific/elm/backend/internal/auth"
	"github.com/g-asiapacific/elm/backend/internal/config"
	"github.com/g-asiapacific/elm/backend/internal/database"
	"github.com/g-asiapacific/elm/backend/internal/employee"
	"github.com/g-asiapacific/elm/backend/internal/holiday"
	"github.com/g-asiapacific/elm/backend/internal/httpapi"
	"github.com/g-asiapacific/elm/backend/internal/leave"
	"github.com/g-asiapacific/elm/backend/internal/notification"
	"github.com/g-asiapacific/elm/backend/internal/review"
	"github.com/g-asiapacific/elm/backend/internal/settings"
	"github.com/g-asiapacific/elm/backend/internal/store"
	"github.com/g-asiapacific/elm/backend/internal/system"
	"github.com/g-asiapacific/elm/backend/internal/worker"
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, "server:", err)
		os.Exit(1)
	}
}

func run() error {
	envFile, err := config.LoadDotEnv()
	if err != nil {
		return fmt.Errorf("load .env: %w", err)
	}
	cfg, err := config.Load()
	if err != nil {
		return err
	}
	if err := cfg.ValidateServer(); err != nil {
		return err
	}
	log := applog.New(cfg.Env, cfg.LogLevel)
	slog.SetDefault(log)

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	db, err := database.Open(ctx, cfg.DB)
	if err != nil {
		return err
	}
	defer db.Close()

	latest, err := database.LatestMigration()
	if err != nil {
		return err
	}
	if v, dirty, err := database.SchemaVersion(ctx, db); err != nil || dirty || v != latest {
		// Start anyway so /health/ready can report it, but make the problem obvious.
		log.Warn("database schema is not up to date; run: go run ./cmd/migrate up", "version", v, "latest", latest, "dirty", dirty)
	}

	authSvc := auth.NewService(db, auth.NewOIDCProvider(cfg.Google), cfg, log)

	health := &httpapi.Health{Logger: log, Checks: []httpapi.Check{
		{Name: "database", Fn: db.PingContext},
		{Name: "migrations", Fn: migrationsCheck(db, latest)},
	}}
	settingsSvc := settings.NewService(db)
	settingsSvc.SetSMTP(cfg.SMTP.Configured(), cfg.SMTP.From)
	settingsSvc.SetLive(cfg.EmailLive, cfg.Access)
	handler := httpapi.NewRouter(httpapi.Deps{
		Config: cfg, Logger: log, Health: health, Auth: authSvc,
		Employees: employee.NewHandlers(employee.NewService(db), log),
		Calendar:  holiday.NewHandlers(holiday.NewAdmin(db), holiday.Seeder{DB: db}, log),
		Settings:  settings.NewHandlers(settingsSvc, log),
		Leave:     leave.NewHandlers(leave.NewService(db, settingsSvc, cfg.AppOrigin.String()), log),
		Review:    review.NewHandlers(review.NewService(db), settingsSvc, log),
		Notify:    notification.NewCenter(db, log),
		Outbox:    notification.NewAdmin(db, log),
		Analytics: analytics.NewHandlers(db, settingsSvc, log),
		System:    system.NewHandlers(db, log),
	})

	outbox := notification.Outbox{DB: db, Origin: cfg.AppOrigin.String(), SMTP: cfg.SMTP, Live: cfg.EmailLive, Access: cfg.Access}
	workerCtx, stopWorkers := context.WithCancel(context.Background())
	waitWorkers := worker.Start(workerCtx, db, log,
		worker.Job{Name: "session_cleanup", Interval: 10 * time.Minute, Run: authSvc.Cleanup},
		worker.Job{Name: "email_outbox", Interval: 30 * time.Second, Run: outbox.Process},
		worker.Job{Name: "leader_reminders", Interval: 5 * time.Minute, Run: notification.Reminders{DB: db}.Run},
		worker.Job{Name: "email_log_retention", Interval: 6 * time.Hour, Run: outbox.Cleanup},
		worker.Job{Name: "idempotency_cleanup", Interval: time.Hour, Run: func(ctx context.Context) error {
			_, err := store.New(db).DeleteExpiredIdempotencyKeys(ctx, time.Now().UTC())
			return err
		}},
	)

	srv := &http.Server{
		Addr:              cfg.HTTPAddr,
		Handler:           handler,
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       15 * time.Second,
		WriteTimeout:      30 * time.Second,
		IdleTimeout:       60 * time.Second,
		MaxHeaderBytes:    32 << 10,
	}

	errCh := make(chan error, 1)
	go func() {
		log.Info("http server listening", "addr", cfg.HTTPAddr, "env", cfg.Env, "env_file", envFile)
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			errCh <- err
		}
		close(errCh)
	}()

	select {
	case err := <-errCh:
		stopWorkers()
		waitWorkers()
		return err
	case <-ctx.Done():
	}

	log.Info("shutting down")
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	shutdownErr := srv.Shutdown(shutdownCtx)
	// Stop workers after HTTP so in-flight requests finish first; both complete before the DB pool closes.
	stopWorkers()
	waitWorkers()
	if shutdownErr != nil {
		return fmt.Errorf("graceful shutdown: %w", shutdownErr)
	}
	log.Info("stopped")
	return nil
}

func migrationsCheck(db *sql.DB, latest uint) func(context.Context) error {
	return func(ctx context.Context) error {
		v, dirty, err := database.SchemaVersion(ctx, db)
		switch {
		case err != nil:
			return err
		case dirty:
			return fmt.Errorf("schema version %d is dirty", v)
		case v != latest:
			return fmt.Errorf("schema version %d, expected %d", v, latest)
		}
		return nil
	}
}
