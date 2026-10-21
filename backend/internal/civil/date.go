// Package civil provides a time-zone-free calendar date.
package civil

import (
	"fmt"
	"time"
)

// Date is a calendar date without time or zone.
type Date struct {
	Year  int
	Month time.Month
	Day   int
}

// D is a terse constructor, normalising overflow (e.g. Feb 30 -> Mar 2).
func D(year int, month time.Month, day int) Date {
	return DateOf(time.Date(year, month, day, 0, 0, 0, 0, time.UTC))
}

// DateOf returns the date of t in t's location.
func DateOf(t time.Time) Date {
	y, m, d := t.Date()
	return Date{y, m, d}
}

// Parse parses YYYY-MM-DD strictly.
func Parse(s string) (Date, error) {
	t, err := time.Parse(time.DateOnly, s)
	if err != nil {
		return Date{}, fmt.Errorf("invalid date %q", s)
	}
	return DateOf(t), nil
}

// UTC returns midnight UTC of the date (the representation the MySQL driver uses for DATE).
func (d Date) UTC() time.Time { return time.Date(d.Year, d.Month, d.Day, 0, 0, 0, 0, time.UTC) }

func (d Date) AddDays(n int) Date           { return D(d.Year, d.Month, d.Day+n) }
func (d Date) Weekday() time.Weekday        { return d.UTC().Weekday() }
func (d Date) Before(o Date) bool           { return d.UTC().Before(o.UTC()) }
func (d Date) After(o Date) bool            { return d.UTC().After(o.UTC()) }
func (d Date) String() string               { return fmt.Sprintf("%04d-%02d-%02d", d.Year, int(d.Month), d.Day) }
func (d Date) IsZero() bool                 { return d == Date{} }
func (d Date) DaysUntil(o Date) int         { return int(o.UTC().Sub(d.UTC()).Hours() / 24) }
func (d Date) Compare(o Date) int           { return d.UTC().Compare(o.UTC()) }
func (d Date) MarshalText() ([]byte, error) { return []byte(d.String()), nil }

func (d *Date) UnmarshalText(b []byte) error {
	p, err := Parse(string(b))
	if err != nil {
		return err
	}
	*d = p
	return nil
}
