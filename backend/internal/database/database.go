// Package database owns the MySQL connection pool, migrations and transaction helper.
package database

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"net"
	"strconv"
	"strings"
	"time"

	"github.com/go-sql-driver/mysql"

	"github.com/g-asiapacific/elm/backend/internal/config"
)

// sessionSQLMode is forced on every connection: WampServer ships with an empty global sql_mode.
const sessionSQLMode = "'STRICT_ALL_TABLES,NO_ZERO_IN_DATE,NO_ZERO_DATE,ERROR_FOR_DIVISION_BY_ZERO,ONLY_FULL_GROUP_BY,NO_ENGINE_SUBSTITUTION'"

// DSN builds a driver DSN. Timestamps are read/written as UTC and the session time zone is pinned to +00:00.
func DSN(c config.DB, multiStatements bool) string {
	mc := mysql.NewConfig()
	mc.User = c.User
	mc.Passwd = c.Password
	mc.Net = "tcp"
	mc.Addr = net.JoinHostPort(c.Host, strconv.Itoa(c.Port))
	mc.DBName = c.Name
	mc.ParseTime = true
	mc.Loc = time.UTC
	mc.Collation = "utf8mb4_unicode_ci"
	mc.Timeout = 5 * time.Second
	mc.ReadTimeout = 30 * time.Second
	mc.WriteTimeout = 30 * time.Second
	mc.MultiStatements = multiStatements
	mc.Params = map[string]string{
		"time_zone": "'+00:00'",
		"sql_mode":  sessionSQLMode,
	}
	return mc.FormatDSN()
}

// Open creates the pool and verifies the server is MySQL 8+ (not the MariaDB instance WampServer also runs).
func Open(ctx context.Context, c config.DB) (*sql.DB, error) {
	db, err := sql.Open("mysql", DSN(c, false))
	if err != nil {
		return nil, fmt.Errorf("open database: %w", err)
	}
	db.SetMaxOpenConns(c.MaxOpenConns)
	db.SetMaxIdleConns(c.MaxIdleConns)
	db.SetConnMaxLifetime(c.ConnMaxLifetime)
	db.SetConnMaxIdleTime(5 * time.Minute)

	if err := CheckServer(ctx, db); err != nil {
		db.Close()
		return nil, err
	}
	return db, nil
}

// CheckServer pings and validates server version and session settings.
func CheckServer(ctx context.Context, db *sql.DB) error {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	var version, tz string
	if err := db.QueryRowContext(ctx, "SELECT VERSION(), @@session.time_zone").Scan(&version, &tz); err != nil {
		return fmt.Errorf("connect to database: %w", err)
	}
	if err := checkVersion(version); err != nil {
		return err
	}
	if tz != "+00:00" {
		return fmt.Errorf("database session time zone is %q, expected +00:00", tz)
	}
	return nil
}

func checkVersion(v string) error {
	if strings.Contains(strings.ToLower(v), "mariadb") {
		return fmt.Errorf("connected to MariaDB (%s); point DB_PORT at the MySQL 8 server", v)
	}
	major, _, _ := strings.Cut(v, ".")
	if n, err := strconv.Atoi(major); err != nil || n < 8 {
		return fmt.Errorf("unsupported MySQL version %q; MySQL 8.0+ is required", v)
	}
	return nil
}

// InTx runs fn in a transaction, committing on success and rolling back on error or panic.
// A transaction chosen as an InnoDB deadlock victim (error 1213) is retried from the start, up to 3 attempts;
// fn must therefore not have side effects outside the transaction.
func InTx(ctx context.Context, db *sql.DB, fn func(tx *sql.Tx) error) error {
	var err error
	for attempt := 1; attempt <= 3; attempt++ {
		if err = inTxOnce(ctx, db, fn); !isDeadlock(err) {
			return err
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(time.Duration(attempt*20) * time.Millisecond):
		}
	}
	return err
}

func isDeadlock(err error) bool {
	var me *mysql.MySQLError
	return errors.As(err, &me) && me.Number == 1213
}

func inTxOnce(ctx context.Context, db *sql.DB, fn func(tx *sql.Tx) error) (err error) {
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin tx: %w", err)
	}
	defer func() {
		if p := recover(); p != nil {
			_ = tx.Rollback()
			panic(p)
		}
		if err != nil {
			_ = tx.Rollback()
		}
	}()
	if err = fn(tx); err != nil {
		return err
	}
	if err = tx.Commit(); err != nil {
		return fmt.Errorf("commit tx: %w", err)
	}
	return nil
}

// IsDuplicateKey reports whether err is a MySQL unique-key violation (1062).
func IsDuplicateKey(err error) bool {
	var me *mysql.MySQLError
	return errors.As(err, &me) && me.Number == 1062
}
