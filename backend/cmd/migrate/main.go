// Command migrate applies the embedded database migrations.
//
//	go run ./cmd/migrate up
//	go run ./cmd/migrate version
//	go run ./cmd/migrate down -steps 1 -confirm <DB_NAME>   (destructive; asks for the database name)
package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/g-asiapacific/elm/backend/internal/config"
	"github.com/g-asiapacific/elm/backend/internal/database"
)

func main() {
	if err := run(os.Args[1:]); err != nil {
		fmt.Fprintln(os.Stderr, "migrate:", err)
		os.Exit(1)
	}
}

func run(args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("usage: migrate up | version | down -steps N -confirm DB_NAME")
	}
	if _, err := config.LoadDotEnv(); err != nil {
		return fmt.Errorf("load .env: %w", err)
	}
	cfg, err := config.Load()
	if err != nil {
		return err
	}

	cmd, rest := args[0], args[1:]
	fs := flag.NewFlagSet(cmd, flag.ContinueOnError)
	steps := fs.Int("steps", 0, "number of migrations to roll back")
	confirm := fs.String("confirm", "", "must equal DB_NAME to allow a rollback")
	if err := fs.Parse(rest); err != nil {
		return err
	}
	if cmd == "down" && (*steps < 1 || *confirm != cfg.DB.Name) {
		return fmt.Errorf("down drops tables and data: pass -steps N and -confirm %s", cfg.DB.Name)
	}

	m, err := database.NewMigrator(cfg.DB)
	if err != nil {
		return err
	}
	defer m.Close()

	switch cmd {
	case "up":
		applied, err := m.Up()
		if err != nil {
			return err
		}
		if !applied {
			fmt.Println("no change: database is up to date")
		}
	case "down":
		if err := m.Down(*steps); err != nil {
			return err
		}
	case "version":
	default:
		return fmt.Errorf("unknown command %q", cmd)
	}

	v, dirty, err := m.Version()
	if err != nil {
		return err
	}
	latest, err := database.LatestMigration()
	if err != nil {
		return err
	}
	fmt.Printf("database %s: version %d (latest %d), dirty=%t\n", cfg.DB.Name, v, latest, dirty)
	return nil
}
