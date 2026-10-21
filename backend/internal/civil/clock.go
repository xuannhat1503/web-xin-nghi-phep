package civil

import (
	"fmt"
	"strconv"
	"strings"
)

// Clock is a wall-clock time of day in minutes since midnight (0..1440). 1440 means end of day.
type Clock int

const EndOfDay Clock = 24 * 60

// ParseClock accepts "HH:MM" or MySQL TIME text "HH:MM:SS" (seconds must be zero).
func ParseClock(s string) (Clock, error) {
	parts := strings.Split(strings.TrimSpace(s), ":")
	if len(parts) < 2 || len(parts) > 3 || len(parts[0]) != 2 || len(parts[1]) != 2 {
		return 0, fmt.Errorf("invalid time %q (expected HH:mm)", s)
	}
	h, err1 := strconv.Atoi(parts[0])
	m, err2 := strconv.Atoi(parts[1])
	if err1 != nil || err2 != nil || h < 0 || h > 24 || m < 0 || m > 59 || (h == 24 && m != 0) {
		return 0, fmt.Errorf("invalid time %q", s)
	}
	if len(parts) == 3 && parts[2] != "00" {
		return 0, fmt.Errorf("invalid time %q (seconds not supported)", s)
	}
	return Clock(h*60 + m), nil
}

// String formats as HH:MM.
func (c Clock) String() string { return fmt.Sprintf("%02d:%02d", int(c)/60, int(c)%60) }

// SQL formats as MySQL TIME text.
func (c Clock) SQL() string { return c.String() + ":00" }
