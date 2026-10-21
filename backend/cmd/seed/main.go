// Command seed inserts controlled reference data.
//
//	go run ./cmd/seed holidays                  (current and next year, Asia/Ho_Chi_Minh)
//	go run ./cmd/seed holidays -years 2026,2027
//
// Holiday seeds are DRAFT, insert-only and idempotent; see internal/holiday.
package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"time"
	_ "time/tzdata" // Windows has no system zoneinfo

	"github.com/g-asiapacific/elm/backend/internal/config"
	"github.com/g-asiapacific/elm/backend/internal/database"
	"github.com/g-asiapacific/elm/backend/internal/holiday"
)

func main() {
	if err := run(os.Args[1:]); err != nil {
		fmt.Fprintln(os.Stderr, "seed:", err)
		os.Exit(1)
	}
}

func run(args []string) error {
	if len(args) == 0 || args[0] != "holidays" {
		return fmt.Errorf("usage: seed holidays [-years 2026,2027]")
	}
	fs := flag.NewFlagSet("holidays", flag.ContinueOnError)
	yearsFlag := fs.String("years", "", "comma-separated years (default: current and next year)")
	if err := fs.Parse(args[1:]); err != nil {
		return err
	}
	years, err := parseYears(*yearsFlag)
	if err != nil {
		return err
	}

	if _, err := config.LoadDotEnv(); err != nil {
		return fmt.Errorf("load .env: %w", err)
	}
	cfg, err := config.Load()
	if err != nil {
		return err
	}
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()

	db, err := database.Open(ctx, cfg.DB)
	if err != nil {
		return err
	}
	defer db.Close()

	seeder := holiday.Seeder{DB: db}
	for _, y := range years {
		rep, err := seeder.SeedYear(ctx, y)
		if err != nil {
			return fmt.Errorf("year %d: %w", y, err)
		}
		if rep.YearConfirmed {
			fmt.Printf("%d: skipped - calendar year already CONFIRMED by an Admin\n", y)
			continue
		}
		fmt.Printf("%d: inserted %d, skipped %d (all new entries are DRAFT)\n", y, len(rep.Inserted), len(rep.Skipped))
		for _, s := range rep.Skipped {
			fmt.Printf("  skip %-32s %s\n", s.Key, s.Reason)
		}
	}
	return nil
}

func parseYears(s string) ([]int, error) {
	if strings.TrimSpace(s) == "" {
		loc, err := time.LoadLocation("Asia/Ho_Chi_Minh")
		if err != nil {
			return nil, err
		}
		y := time.Now().In(loc).Year()
		return []int{y, y + 1}, nil
	}
	var years []int
	for _, part := range strings.Split(s, ",") {
		y, err := strconv.Atoi(strings.TrimSpace(part))
		if err != nil || y < 2000 || y > 2199 {
			return nil, fmt.Errorf("invalid year %q", part)
		}
		years = append(years, y)
	}
	return years, nil
}
