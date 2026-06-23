package quote

import (
	"sort"
	"time"
)

// fomc2026 lists FOMC rate-decision announcement dates for 2026.
// Each is a Wednesday at 14:00 ET. Source: Federal Reserve.
var fomc2026 = []struct {
	month time.Month
	day   int
}{
	{time.January, 28},
	{time.March, 18},
	{time.April, 29},
	{time.June, 10},
	{time.July, 29},
	{time.September, 16},
	{time.October, 28},
	{time.December, 9},
}

// UpcomingEconEvents returns the next 5 highest-impact US economic events
// after now, ordered by date. Events are computed from known release schedules
// (FOMC hardcoded, NFP/CPI/PCE derived algorithmically).
func UpcomingEconEvents(now time.Time) []EconEvent {
	loc, _ := time.LoadLocation("America/New_York")
	if loc == nil {
		loc = time.UTC
	}
	now = now.In(loc)
	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, loc)
	cutoff := today.AddDate(0, 3, 0) // scan 3 months ahead

	type candidate struct {
		t time.Time
		e EconEvent
	}
	var cands []candidate

	add := func(t time.Time, name, impact string) {
		if !t.Before(today) && t.Before(cutoff) {
			cands = append(cands, candidate{
				t: t,
				e: EconEvent{Name: name, Country: "US", Impact: impact},
			})
		}
	}

	for year := now.Year(); year <= now.Year()+1; year++ {
		// FOMC Rate Decisions
		for _, fd := range fomcDates(year, loc) {
			add(fd, "FOMC Rate Decision", "high")
		}

		for m := time.January; m <= time.December; m++ {
			// Non-Farm Payrolls: first Friday of each month at 08:30 ET
			nfp := firstWeekdayOf(year, m, time.Friday, loc)
			add(time.Date(year, m, nfp, 8, 30, 0, 0, loc), "Non-Farm Payrolls", "high")

			// CPI Release: first Wednesday on or after the 10th at 08:30 ET
			cpi := firstWeekdayOnOrAfter(year, m, 10, time.Wednesday, loc)
			add(time.Date(year, m, cpi, 8, 30, 0, 0, loc), "CPI Release", "high")

			// PCE Price Index: last Friday of each month at 08:30 ET
			pce := lastWeekdayOf(year, m, time.Friday, loc)
			add(time.Date(year, m, pce, 8, 30, 0, 0, loc), "PCE Price Index", "high")
		}
	}

	sort.Slice(cands, func(i, j int) bool { return cands[i].t.Before(cands[j].t) })

	seen := map[string]bool{}
	events := make([]EconEvent, 0, 5)
	for _, c := range cands {
		key := c.t.Format("2006-01-02") + "|" + c.e.Name
		if seen[key] {
			continue
		}
		seen[key] = true
		ev := c.e
		ev.When = c.t.Format("Mon Jan 02 15:04") + " ET"
		events = append(events, ev)
		if len(events) >= 5 {
			break
		}
	}
	return events
}

// fomcDates returns FOMC announcement times for the given year.
func fomcDates(year int, loc *time.Location) []time.Time {
	switch year {
	case 2026:
		dates := make([]time.Time, len(fomc2026))
		for i, fd := range fomc2026 {
			dates[i] = time.Date(2026, fd.month, fd.day, 14, 0, 0, 0, loc)
		}
		return dates
	default:
		return nil
	}
}

// firstWeekdayOf returns the day-of-month of the first occurrence of wd
// in the given year/month.
func firstWeekdayOf(year int, month time.Month, wd time.Weekday, loc *time.Location) int {
	t := time.Date(year, month, 1, 0, 0, 0, 0, loc)
	diff := (int(wd) - int(t.Weekday()) + 7) % 7
	return 1 + diff
}

// firstWeekdayOnOrAfter returns the day-of-month for the first occurrence of
// wd on or after startDay within the given year/month.
func firstWeekdayOnOrAfter(year int, month time.Month, startDay int, wd time.Weekday, loc *time.Location) int {
	t := time.Date(year, month, startDay, 0, 0, 0, 0, loc)
	diff := (int(wd) - int(t.Weekday()) + 7) % 7
	return startDay + diff
}

// lastWeekdayOf returns the day-of-month of the last occurrence of wd
// in the given year/month.
func lastWeekdayOf(year int, month time.Month, wd time.Weekday, loc *time.Location) int {
	last := time.Date(year, month+1, 0, 0, 0, 0, 0, loc) // day 0 = last day of month
	diff := (int(last.Weekday()) - int(wd) + 7) % 7
	return last.Day() - diff
}
