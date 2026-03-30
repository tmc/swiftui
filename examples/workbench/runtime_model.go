//go:build darwin
// +build darwin

package main

import "time"

func normalizeDay(t time.Time) time.Time {
	y, m, d := t.In(time.Local).Date()
	return time.Date(y, m, d, 0, 0, 0, 0, time.Local)
}

func plannerWindow(anchor time.Time) (time.Time, time.Time) {
	start := normalizeDay(anchor)
	return start, start.AddDate(0, 0, 6)
}

func plannerSelection(anchor time.Time, mask int) []time.Time {
	start := normalizeDay(anchor)
	if mask == 0 {
		return nil
	}
	dates := make([]time.Time, 0, 7)
	for i := 0; i < 7; i++ {
		if mask&(1<<i) == 0 {
			continue
		}
		dates = append(dates, start.AddDate(0, 0, i))
	}
	return dates
}
