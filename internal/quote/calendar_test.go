package quote

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestFOMCDatesCurrent is a tripwire: the maintained FOMC schedule must always
// cover the current and next calendar year. When it stops (i.e. a year rolls
// over and nobody added the new dates), this fails in CI instead of silently
// dropping FOMC events from the live calendar. Fix it by adding the next year's
// dates from federalreserve.gov/monetarypolicy/fomccalendars.htm.
func TestFOMCDatesCurrent(t *testing.T) {
	loc, err := time.LoadLocation("America/New_York")
	require.NoError(t, err)

	year := time.Now().Year()
	for _, y := range []int{year, year + 1} {
		dates := fomcDates(y, loc)
		assert.NotEmptyf(t, dates, "FOMC schedule missing for %d — add it from the Fed calendar", y)
		assert.Lenf(t, dates, 8, "expected 8 FOMC meetings in %d", y)
	}
}

// UpcomingEconEvents must never come back empty: even outside the maintained
// FOMC window the algorithmic events (NFP/CPI/PCE) keep the calendar populated.
func TestUpcomingEconEventsNeverEmpty(t *testing.T) {
	// A date far past the maintained FOMC schedule still yields events.
	future := time.Date(2099, time.June, 1, 12, 0, 0, 0, time.UTC)
	events := UpcomingEconEvents(future)
	assert.NotEmpty(t, events, "calendar should fall back to derived events")
}
