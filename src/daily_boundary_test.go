package main

import (
	"testing"
	"time"
)

func TestDailyTodayDoesNotCountFuturePortionOfCachedEntries(t *testing.T) {
	now := time.Date(2026, 10, 19, 10, 0, 0, 0, time.UTC) // Monday.
	for _, tc := range []struct {
		name     string
		entry    Entry
		wantDay  int64
		wantWeek int64
	}{
		{"completed entry ending in the future", Entry{Start: now.Add(-time.Hour), Duration: 7200}, 3600, 3600},
		{"entry entirely in the future", Entry{Start: now.Add(time.Hour), Duration: 3600}, 0, 0},
		{"finished entry", Entry{Start: now.Add(-2 * time.Hour), Duration: 3600}, 3600, 3600},
		{"running entry", Entry{Start: now.Add(-time.Hour), Duration: -1}, 3600, 3600},
		{"cross-midnight entry ending in the future", Entry{Start: now.Add(-12 * time.Hour), Duration: 24 * 3600}, 10 * 3600, 10 * 3600},
	} {
		t.Run(tc.name, func(t *testing.T) {
			days, week := daily(Snapshot{Entries: []Entry{tc.entry}}, now)
			if days[6] != tc.wantDay || week != tc.wantWeek {
				t.Fatalf("today=%d week=%d; want today=%d week=%d", days[6], week, tc.wantDay, tc.wantWeek)
			}
		})
	}
}

func TestDailyPastDaysKeepLocalDSTBoundaries(t *testing.T) {
	location, err := time.LoadLocation("Europe/Oslo")
	if err != nil {
		t.Fatal(err)
	}
	for _, tc := range []struct {
		month time.Month
		day   int
		hours int64
	}{
		{time.March, 29, 23},
		{time.October, 25, 25},
	} {
		start := time.Date(2026, tc.month, tc.day, 0, 0, 0, 0, location)
		now := start.AddDate(0, 0, 1).Add(10 * time.Hour)
		days, week := daily(Snapshot{Entries: []Entry{{Start: start, Duration: tc.hours * 3600}}}, now)
		if days[5] != tc.hours*3600 || days[6] != 0 || week != 0 {
			t.Fatalf("%v: yesterday=%d today=%d week=%d", start, days[5], days[6], week)
		}
	}
}
