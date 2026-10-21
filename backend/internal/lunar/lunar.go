// Package lunar converts Vietnamese lunar dates to solar (Gregorian) dates.
//
// It implements Ho Ngoc Duc's astronomical algorithm ("Am lich Viet Nam"), based on
// "Astronomical Algorithms" by Jean Meeus, evaluated for UTC+7. The Vietnamese and Chinese
// calendars differ in some years (e.g. Tet 1985, 2007) precisely because of this time zone.
package lunar

import (
	"errors"
	"math"
	"time"

	"github.com/g-asiapacific/elm/backend/internal/civil"
)

// VietnamTZ is the time zone (hours) used by the Vietnamese lunar calendar.
const VietnamTZ = 7.0

var ErrInvalidDate = errors.New("invalid lunar date")

// ToSolar converts a lunar date (leap=true for the intercalary month) to its solar date.
func ToSolar(day, month, year int, leap bool) (civil.Date, error) {
	if day < 1 || day > 30 || month < 1 || month > 12 || year < 1900 || year > 2199 {
		return civil.Date{}, ErrInvalidDate
	}
	const tz = VietnamTZ
	var a11, b11 int
	if month < 11 {
		a11 = lunarMonth11(year-1, tz)
		b11 = lunarMonth11(year, tz)
	} else {
		a11 = lunarMonth11(year, tz)
		b11 = lunarMonth11(year+1, tz)
	}
	k := int(math.Floor(0.5 + (float64(a11)-2415021.076998695)/29.530588853))
	off := month - 11
	if off < 0 {
		off += 12
	}
	if b11-a11 > 365 { // leap year: 13 lunar months
		leapOff := leapMonthOffset(a11, tz)
		leapMonth := leapOff - 2
		if leapMonth < 0 {
			leapMonth += 12
		}
		if leap && month != leapMonth {
			return civil.Date{}, ErrInvalidDate
		}
		if leap || off >= leapOff {
			off++
		}
	} else if leap {
		return civil.Date{}, ErrInvalidDate
	}
	monthStart := newMoonDay(k+off, tz)
	nextMonthStart := newMoonDay(k+off+1, tz)
	if day > nextMonthStart-monthStart {
		return civil.Date{}, ErrInvalidDate // e.g. day 30 of a 29-day month
	}
	return jdToDate(monthStart + day - 1), nil
}

// LastDayOfYear returns the solar date of the last day of the given lunar year (New Year's Eve, "Giao thua").
func LastDayOfYear(year int) (civil.Date, error) {
	tet, err := ToSolar(1, 1, year+1, false)
	if err != nil {
		return civil.Date{}, err
	}
	return tet.AddDays(-1), nil
}

func jdFromDate(dd, mm, yy int) int {
	a := (14 - mm) / 12
	y := yy + 4800 - a
	m := mm + 12*a - 3
	jd := dd + (153*m+2)/5 + 365*y + y/4 - y/100 + y/400 - 32045
	if jd < 2299161 {
		jd = dd + (153*m+2)/5 + 365*y + y/4 - 32083
	}
	return jd
}

func jdToDate(jd int) civil.Date {
	var b, c int
	if jd > 2299160 {
		a := jd + 32044
		b = (4*a + 3) / 146097
		c = a - (b*146097)/4
	} else {
		b = 0
		c = jd + 32082
	}
	d := (4*c + 3) / 1461
	e := c - (1461*d)/4
	m := (5*e + 2) / 153
	day := e - (153*m+2)/5 + 1
	month := m + 3 - 12*(m/10)
	year := b*100 + d - 4800 + m/10
	return civil.D(year, time.Month(month), day)
}

// newMoon returns the Julian day (fractional, UTC) of the k-th new moon after 1900-01-01.
func newMoon(k int) float64 {
	kf := float64(k)
	T := kf / 1236.85
	T2 := T * T
	T3 := T2 * T
	dr := math.Pi / 180
	jd1 := 2415020.75933 + 29.53058868*kf + 0.0001178*T2 - 0.000000155*T3
	jd1 += 0.00033 * math.Sin((166.56+132.87*T-0.009173*T2)*dr)
	M := 359.2242 + 29.10535608*kf - 0.0000333*T2 - 0.00000347*T3
	Mpr := 306.0253 + 385.81691806*kf + 0.0107306*T2 + 0.00001236*T3
	F := 21.2964 + 390.67050646*kf - 0.0016528*T2 - 0.00000239*T3
	C1 := (0.1734-0.000393*T)*math.Sin(M*dr) + 0.0021*math.Sin(2*dr*M)
	C1 = C1 - 0.4068*math.Sin(Mpr*dr) + 0.0161*math.Sin(dr*2*Mpr)
	C1 = C1 - 0.0004*math.Sin(dr*3*Mpr)
	C1 = C1 + 0.0104*math.Sin(dr*2*F) - 0.0051*math.Sin(dr*(M+Mpr))
	C1 = C1 - 0.0074*math.Sin(dr*(M-Mpr)) + 0.0004*math.Sin(dr*(2*F+M))
	C1 = C1 - 0.0004*math.Sin(dr*(2*F-M)) - 0.0006*math.Sin(dr*(2*F+Mpr))
	C1 = C1 + 0.0010*math.Sin(dr*(2*F-Mpr)) + 0.0005*math.Sin(dr*(2*Mpr+M))
	var deltaT float64
	if T < -11 {
		deltaT = 0.001 + 0.000839*T + 0.0002261*T2 - 0.00000845*T3 - 0.000000081*T*T3
	} else {
		deltaT = -0.000278 + 0.000265*T + 0.000262*T2
	}
	return jd1 + C1 - deltaT
}

func newMoonDay(k int, tz float64) int {
	return int(math.Floor(newMoon(k) + 0.5 + tz/24))
}

// sunLongitude returns the sun's longitude (radians, 0..2π) at Julian day jdn.
func sunLongitude(jdn float64) float64 {
	T := (jdn - 2451545.0) / 36525
	T2 := T * T
	dr := math.Pi / 180
	M := 357.52910 + 35999.05030*T - 0.0001559*T2 - 0.00000048*T*T2
	L0 := 280.46645 + 36000.76983*T + 0.0003032*T2
	DL := (1.914600 - 0.004817*T - 0.000014*T2) * math.Sin(dr*M)
	DL += (0.019993-0.000101*T)*math.Sin(dr*2*M) + 0.000290*math.Sin(dr*3*M)
	L := (L0 + DL) * dr
	return L - 2*math.Pi*math.Floor(L/(2*math.Pi))
}

// sunLongitudeSector returns which of the 12 major solar terms (0..11) the day falls in.
func sunLongitudeSector(dayNumber int, tz float64) int {
	return int(math.Floor(sunLongitude(float64(dayNumber)-0.5-tz/24) / math.Pi * 6))
}

// lunarMonth11 returns the Julian day number of the start of lunar month 11 (containing the winter solstice) of year yy.
func lunarMonth11(yy int, tz float64) int {
	off := float64(jdFromDate(31, 12, yy)) - 2415021
	k := int(math.Floor(off / 29.530588853))
	nm := newMoonDay(k, tz)
	if sunLongitudeSector(nm, tz) >= 9 {
		nm = newMoonDay(k-1, tz)
	}
	return nm
}

// leapMonthOffset returns the offset (from month 11) of the leap month: the first month without a major solar term.
func leapMonthOffset(a11 int, tz float64) int {
	k := int(math.Floor((float64(a11)-2415021.076998695)/29.530588853 + 0.5))
	i := 1
	arc := sunLongitudeSector(newMoonDay(k+i, tz), tz)
	for {
		last := arc
		i++
		arc = sunLongitudeSector(newMoonDay(k+i, tz), tz)
		if arc == last || i >= 14 {
			break
		}
	}
	return i - 1
}
