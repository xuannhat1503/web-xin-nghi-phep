package holiday

import (
	"fmt"
	"slices"
	"strings"
	"time"

	"github.com/g-asiapacific/elm/backend/internal/civil"
	"github.com/g-asiapacific/elm/backend/internal/lunar"
)

// SeedDay is a proposed calendar entry. Seeds are always inserted as DRAFT; an Admin confirms or removes them.
type SeedDay struct {
	Key    string
	Date   civil.Date
	Name   string
	Kind   Kind
	Source string
	Note   string
}

const (
	srcLabourCode   = "Labour Code 2019, Art. 112"
	srcTet          = "Labour Code 2019, Art. 112 (5 days, dates chosen by the employer); lunar date computed (Ho Ngoc Duc algorithm, UTC+7)"
	srcHungKings    = "Labour Code 2019, Art. 112; 10th day of 3rd lunar month computed (Ho Ngoc Duc algorithm, UTC+7)"
	srcCompensatory = "Labour Code 2019, Art. 111(3): holiday falling on a weekly rest day is moved to the next working day"
	srcCommon       = "Common company practice"

	noteVerify       = "Seeded as DRAFT. Verify against the official / company announcement before confirming."
	noteTet          = "Proposed Tet schedule (New Year's Eve + days 1-4). Companies may choose a different 5-day split; adjust before confirming."
	noteNationalDay2 = "Proposed adjacent day. The Government announces the official day each year; adjust before confirming."
	noteOptional     = "Optional company holiday (common at foreign-invested companies). Delete if not applicable."
)

// VietnamSeed proposes the common Vietnamese holidays for a solar year: statutory holidays, a 5-day Tet,
// the adjacent National Day, compensatory days for statutory holidays on rest days, and Christmas as an
// optional company holiday. Nothing here is authoritative; every entry is DRAFT.
func VietnamSeed(year int, workdays Workdays) ([]SeedDay, error) {
	if workdays == 0 {
		return nil, fmt.Errorf("workdays mask must not be empty")
	}
	prefix := fmt.Sprintf("VN:%d:", year)
	key := func(code string) string { return prefix + code }

	tet, err := lunar.ToSolar(1, 1, year, false)
	if err != nil {
		return nil, fmt.Errorf("tet %d: %w", year, err)
	}
	hungKings, err := lunar.ToSolar(10, 3, year, false)
	if err != nil {
		return nil, fmt.Errorf("hung kings %d: %w", year, err)
	}

	days := []SeedDay{
		{key("NEW_YEAR"), civil.D(year, time.January, 1), "New Year's Day (Tết Dương lịch)", KindNational, srcLabourCode, noteVerify},
		{key("TET_EVE"), tet.AddDays(-1), "Lunar New Year's Eve (Giao thừa)", KindNational, srcTet, noteTet},
	}
	for i := range 4 {
		days = append(days, SeedDay{
			key(fmt.Sprintf("TET_%d", i+1)), tet.AddDays(i),
			fmt.Sprintf("Lunar New Year – Day %d (Tết Nguyên đán, mùng %d)", i+1, i+1),
			KindNational, srcTet, noteTet,
		})
	}
	days = append(days,
		SeedDay{key("HUNG_KINGS"), hungKings, "Hung Kings Commemoration Day (Giỗ Tổ Hùng Vương)", KindNational, srcHungKings, noteVerify},
		SeedDay{key("REUNIFICATION"), civil.D(year, time.April, 30), "Reunification Day (Ngày Giải phóng miền Nam)", KindNational, srcLabourCode, noteVerify},
		SeedDay{key("LABOUR_DAY"), civil.D(year, time.May, 1), "International Labour Day (Quốc tế Lao động)", KindNational, srcLabourCode, noteVerify},
		SeedDay{key("NATIONAL_DAY"), civil.D(year, time.September, 2), "National Day (Quốc khánh)", KindNational, srcLabourCode, noteVerify},
		SeedDay{key("NATIONAL_DAY_ADJ"), nationalDayAdjacent(year, workdays), "National Day – adjacent day (Quốc khánh, ngày liền kề)", KindNational, srcLabourCode, noteNationalDay2},
		SeedDay{key("CHRISTMAS"), civil.D(year, time.December, 25), "Christmas Day (Giáng sinh)", KindCompany, srcCommon, noteOptional},
	)

	days = append(days, compensatoryDays(days, workdays, prefix)...)
	slices.SortFunc(days, func(a, b SeedDay) int { return a.Date.Compare(b.Date) })
	return days, nil
}

// nationalDayAdjacent proposes Sep 1 or Sep 3, preferring the one that bridges to a weekend.
func nationalDayAdjacent(year int, workdays Workdays) civil.Date {
	sep1, sep2, sep3 := civil.D(year, time.September, 1), civil.D(year, time.September, 2), civil.D(year, time.September, 3)
	order := []civil.Date{sep1, sep3}
	if sep2.Weekday() == time.Thursday {
		order = []civil.Date{sep3, sep1}
	}
	for _, d := range order {
		if workdays.IsWorkday(d) {
			return d
		}
	}
	return sep1
}

// compensatoryDays moves each statutory holiday that falls on a rest day to the next free working day.
func compensatoryDays(days []SeedDay, workdays Workdays, keyPrefix string) []SeedDay {
	taken := make(map[civil.Date]bool, len(days))
	for _, d := range days {
		taken[d.Date] = true
	}
	sorted := slices.Clone(days)
	slices.SortFunc(sorted, func(a, b SeedDay) int { return a.Date.Compare(b.Date) })

	var out []SeedDay
	for _, h := range sorted {
		if h.Kind != KindNational || workdays.IsWorkday(h.Date) {
			continue
		}
		d := h.Date.AddDays(1)
		for !workdays.IsWorkday(d) || taken[d] {
			d = d.AddDays(1)
		}
		taken[d] = true
		out = append(out, SeedDay{
			Key:    keyPrefix + "COMP:" + strings.TrimPrefix(h.Key, keyPrefix),
			Date:   d,
			Name:   "Compensatory day for " + h.Name,
			Kind:   KindCompensatory,
			Source: srcCompensatory,
			Note:   fmt.Sprintf("Computed: %s falls on a %s. %s", h.Date, h.Date.Weekday(), noteVerify),
		})
	}
	return out
}
