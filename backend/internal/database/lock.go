package database

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"
)

var ErrLockTimeout = errors.New("could not acquire lock")

// WithNamedLock runs fn in a transaction while holding a MySQL named lock (GET_LOCK). The lock is
// released only after the transaction commits or rolls back, so the next holder sees committed data.
// Use it for invariants that span many rows (e.g. "the manager graph has no cycles").
func WithNamedLock(ctx context.Context, db *sql.DB, name string, wait time.Duration, fn func(tx *sql.Tx) error) error {
	conn, err := db.Conn(ctx)
	if err != nil {
		return err
	}
	defer conn.Close()

	var got sql.NullInt64
	if err := conn.QueryRowContext(ctx, "SELECT GET_LOCK(?, ?)", name, int(wait.Seconds())).Scan(&got); err != nil {
		return fmt.Errorf("get lock: %w", err)
	}
	if !got.Valid || got.Int64 != 1 {
		return ErrLockTimeout
	}
	defer func() {
		// Background context: release even if the request was cancelled.
		_, _ = conn.ExecContext(context.Background(), "DO RELEASE_LOCK(?)", name)
	}()

	tx, err := conn.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin tx: %w", err)
	}
	defer func() {
		if p := recover(); p != nil {
			_ = tx.Rollback()
			panic(p)
		}
	}()
	if err := fn(tx); err != nil {
		_ = tx.Rollback()
		return err
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit tx: %w", err)
	}
	return nil
}
