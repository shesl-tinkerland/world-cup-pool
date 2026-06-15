package account

import "testing"

func TestComputePlayerStatsEmpty(t *testing.T) {
	s := computePlayerStats(nil)
	if s.TipsScored != 0 || s.LongestStreak != 0 || s.CurrentStreak != 0 || s.HitRate.Total != 0 || s.HitRate.Pct != 0 {
		t.Fatalf("empty input must yield zeros, got %+v", s)
	}
}

func TestComputePlayerStatsStreakAndHitRate(t *testing.T) {
	// Chronological order by kickoff string; pure function sorts internally.
	in := []scoredMatch{
		{matchID: "a", kickoff: "2026-06-11T15:00:00Z", points: 3, exact: false, gdDev: 1},
		{matchID: "b", kickoff: "2026-06-12T15:00:00Z", points: 6, exact: true, gdDev: 0},
		{matchID: "c", kickoff: "2026-06-13T15:00:00Z", points: 0, exact: false, gdDev: 4}, // miss breaks streak
		{matchID: "d", kickoff: "2026-06-14T15:00:00Z", points: 1, exact: false, gdDev: 2},
		{matchID: "e", kickoff: "2026-06-15T15:00:00Z", points: 2, exact: false, gdDev: 1},
		{matchID: "f", kickoff: "2026-06-16T15:00:00Z", points: 6, exact: true, gdDev: 0},
	}
	s := computePlayerStats(in)
	if s.TipsScored != 6 {
		t.Fatalf("TipsScored = %d, want 6", s.TipsScored)
	}
	if s.HitRate.Count != 2 || s.HitRate.Total != 6 {
		t.Fatalf("HitRate = %+v, want 2/6", s.HitRate)
	}
	if want := 2.0 / 6.0; s.HitRate.Pct != want {
		t.Fatalf("HitRate.Pct = %v, want %v", s.HitRate.Pct, want)
	}
	if s.LongestStreak != 3 {
		t.Fatalf("LongestStreak = %d, want 3 (d,e,f)", s.LongestStreak)
	}
	if s.CurrentStreak != 3 {
		t.Fatalf("CurrentStreak = %d, want 3", s.CurrentStreak)
	}
}

func TestComputePlayerStatsAllZeros(t *testing.T) {
	in := []scoredMatch{
		{matchID: "a", kickoff: "2026-06-11T15:00:00Z", points: 0, gdDev: 3},
		{matchID: "b", kickoff: "2026-06-12T15:00:00Z", points: 0, gdDev: 5},
	}
	s := computePlayerStats(in)
	if s.LongestStreak != 0 || s.CurrentStreak != 0 {
		t.Fatalf("expected zero streaks, got long=%d current=%d", s.LongestStreak, s.CurrentStreak)
	}
	if s.HitRate.Count != 0 || s.HitRate.Total != 2 {
		t.Fatalf("HitRate = %+v, want 0/2", s.HitRate)
	}
}

func TestComputePlayerStatsBrokenStreakUsesLongest(t *testing.T) {
	in := []scoredMatch{
		{matchID: "a", kickoff: "2026-06-11T15:00:00Z", points: 3},
		{matchID: "b", kickoff: "2026-06-12T15:00:00Z", points: 3},
		{matchID: "c", kickoff: "2026-06-13T15:00:00Z", points: 3},
		{matchID: "d", kickoff: "2026-06-14T15:00:00Z", points: 3},
		{matchID: "e", kickoff: "2026-06-15T15:00:00Z", points: 0}, // current=0
	}
	s := computePlayerStats(in)
	if s.LongestStreak != 4 {
		t.Fatalf("LongestStreak = %d, want 4", s.LongestStreak)
	}
	if s.CurrentStreak != 0 {
		t.Fatalf("CurrentStreak = %d, want 0", s.CurrentStreak)
	}
}
