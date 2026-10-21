// Package holiday manages the company working calendar: holidays, compensatory days, makeup workdays
// and year confirmation.
package holiday

import (
	"time"

	"github.com/g-asiapacific/elm/backend/internal/civil"
)

type Kind string

const (
	KindNational     Kind = "NATIONAL_HOLIDAY"
	KindCompany      Kind = "COMPANY_HOLIDAY"
	KindCompensatory Kind = "COMPENSATORY_HOLIDAY"
	KindMakeup       Kind = "MAKEUP_WORKDAY"
)

// Workdays is a weekday bitmask: bit 0 = Monday ... bit 6 = Sunday (same as policy_versions.workdays_mask).
type Workdays uint8

const MondayToFriday Workdays = 0b0011111

func (w Workdays) Has(d time.Weekday) bool {
	return w&(1<<((int(d)+6)%7)) != 0
}

func (w Workdays) IsWorkday(d civil.Date) bool { return w.Has(d.Weekday()) }
