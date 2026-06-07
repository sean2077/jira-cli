package jira

import "testing"

func TestNextStartAtAdvancesByActualReturnedCount(t *testing.T) {
	next, ok := NextStartAt(10, 50, 3)
	if !ok || next != 13 {
		t.Fatalf("NextStartAt = %d, %t; want 13, true", next, ok)
	}
}

func TestNextStartAtStopsAtTotalOrEmptyPage(t *testing.T) {
	tests := []struct {
		name      string
		startAt   int
		total     int
		itemCount int
	}{
		{name: "empty", startAt: 0, total: 10, itemCount: 0},
		{name: "at total", startAt: 8, total: 10, itemCount: 2},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if next, ok := NextStartAt(tt.startAt, tt.total, tt.itemCount); ok {
				t.Fatalf("NextStartAt = %d, true; want false", next)
			}
		})
	}
}
