package tips

import (
	"testing"
	"time"
)

func TestIsLockedAtKickoffBoundary(t *testing.T) {
	kickoff := time.Date(2026, time.June, 11, 19, 0, 0, 0, time.UTC)

	tests := []struct {
		name string
		now  time.Time
		want bool
	}{
		{name: "before kickoff remains editable", now: kickoff.Add(-time.Nanosecond), want: false},
		{name: "exact kickoff locks", now: kickoff, want: true},
		{name: "after kickoff stays locked", now: kickoff.Add(time.Second), want: true},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := isLocked(tc.now, kickoff); got != tc.want {
				t.Fatalf("isLocked() = %v, want %v", got, tc.want)
			}
		})
	}
}
