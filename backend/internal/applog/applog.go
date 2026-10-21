// Package applog builds the process-wide structured logger.
package applog

import (
	"log/slog"
	"os"

	"github.com/g-asiapacific/elm/backend/internal/config"
)

// New returns a JSON logger in production and a human-readable one otherwise.
func New(env config.Env, level slog.Level) *slog.Logger {
	opts := &slog.HandlerOptions{Level: level}
	if env == config.EnvProduction {
		return slog.New(slog.NewJSONHandler(os.Stdout, opts))
	}
	return slog.New(slog.NewTextHandler(os.Stdout, opts))
}
