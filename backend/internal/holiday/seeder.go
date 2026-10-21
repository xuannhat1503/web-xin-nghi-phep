package holiday

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/g-asiapacific/elm/backend/internal/database"
	"github.com/g-asiapacific/elm/backend/internal/store"
)

// SeedReport summarises one year's seed run.
type SeedReport struct {
	Year          int
	YearConfirmed bool // the whole year was skipped because an Admin already confirmed it
	Inserted      []string
	Skipped       []SkippedSeed
}

type SkippedSeed struct {
	Key    string
	Reason string
}

// Seeder applies VietnamSeed to the database. It is insert-only and idempotent: it never updates or
// re-creates a row an Admin edited or deleted, never touches a CONFIRMED year, and never places a
// second rule on a date that already has one.
type Seeder struct {
	DB *sql.DB
}

func (s Seeder) SeedYear(ctx context.Context, year int) (SeedReport, error) {
	rep := SeedReport{Year: year}
	q := store.New(s.DB)

	mask, err := q.GetCurrentWorkdaysMask(ctx)
	if err != nil {
		return rep, fmt.Errorf("read workdays policy: %w", err)
	}
	days, err := VietnamSeed(year, Workdays(mask))
	if err != nil {
		return rep, err
	}

	err = database.InTx(ctx, s.DB, func(tx *sql.Tx) error {
		qtx := q.WithTx(tx)
		if err := qtx.EnsureCalendarYear(ctx, uint16(year)); err != nil {
			return fmt.Errorf("ensure calendar year: %w", err)
		}
		y, err := qtx.LockCalendarYear(ctx, uint16(year))
		if err != nil {
			return fmt.Errorf("lock calendar year: %w", err)
		}
		if y.Status == store.CalendarYearsStatusCONFIRMED {
			rep.YearConfirmed = true
			return nil
		}

		for _, d := range days {
			reason, err := s.insertOne(ctx, qtx, year, d)
			if err != nil {
				return fmt.Errorf("seed %s: %w", d.Key, err)
			}
			if reason != "" {
				rep.Skipped = append(rep.Skipped, SkippedSeed{d.Key, reason})
			} else {
				rep.Inserted = append(rep.Inserted, d.Key)
			}
		}
		return nil
	})
	return rep, err
}

// insertOne returns a non-empty skip reason when the seed must not be applied.
func (s Seeder) insertOne(ctx context.Context, q *store.Queries, year int, d SeedDay) (string, error) {
	existing, err := q.GetCalendarDayBySeedKey(ctx, sql.NullString{String: d.Key, Valid: true})
	switch {
	case err == nil && existing.DeletedAt.Valid:
		return "deleted by admin", nil
	case err == nil && existing.AdminModified:
		return "modified by admin", nil
	case err == nil:
		return "already seeded", nil
	case !errors.Is(err, sql.ErrNoRows):
		return "", err
	}

	if other, err := q.GetActiveCalendarDayByDate(ctx, sql.NullTime{Time: d.Date.UTC(), Valid: true}); err == nil {
		return fmt.Sprintf("date already has %q (id %d)", other.Name, other.ID), nil
	} else if !errors.Is(err, sql.ErrNoRows) {
		return "", err
	}

	id, err := q.InsertSeedCalendarDay(ctx, store.InsertSeedCalendarDayParams{
		DayDate: d.Date.UTC(),
		Name:    d.Name,
		Kind:    store.CalendarDaysKind(d.Kind),
		Source:  d.Source,
		Note:    d.Note,
		SeedKey: sql.NullString{String: d.Key, Valid: true},
	})
	if err != nil {
		return "", err
	}
	after, err := json.Marshal(map[string]any{
		"date": d.Date.String(), "name": d.Name, "kind": d.Kind, "status": "DRAFT", "seed_key": d.Key,
	})
	if err != nil {
		return "", err
	}
	return "", q.InsertCalendarDayEvent(ctx, store.InsertCalendarDayEventParams{
		Year:          uint16(year),
		CalendarDayID: sql.NullInt64{Int64: id, Valid: true},
		Action:        "SEEDED",
		AfterData:     after,
	})
}
