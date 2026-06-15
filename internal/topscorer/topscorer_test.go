package topscorer

import (
	"testing"
	"time"
)

func TestRateLimiterBlocksWithinWindow(t *testing.T) {
	limiter := &rateLimiter{hits: map[string][]time.Time{}}

	if !limiter.allow("user-1", 2, time.Minute) {
		t.Fatal("first request was blocked")
	}
	if !limiter.allow("user-1", 2, time.Minute) {
		t.Fatal("second request was blocked")
	}
	if limiter.allow("user-1", 2, time.Minute) {
		t.Fatal("third request was not blocked")
	}
}

func TestRateLimiterExpiresOldHits(t *testing.T) {
	limiter := &rateLimiter{hits: map[string][]time.Time{
		"user-1": {time.Now().Add(-2 * time.Minute)},
	}}

	if !limiter.allow("user-1", 1, time.Minute) {
		t.Fatal("expired hit still counted against the rate limit")
	}
}

func TestOrderByGoalsKeepsScorersAheadOfZeroGoalPlayers(t *testing.T) {
	// "Stale" carries a low provider rank but no goals — the exact shape that
	// used to surface a 0-goal player at #1. It must sort to the bottom.
	players := []Player{
		{Name: "Stale", Goals: 0, Rank: 1},
		{Name: "Bravo", Goals: 3, Rank: 0},
		{Name: "Alpha", Goals: 3, Rank: 0},
		{Name: "Charlie", Goals: 5, Rank: 0},
	}

	orderByGoals(players)
	assignCompetitionRanks(players)

	wantOrder := []string{"Charlie", "Alpha", "Bravo", "Stale"}
	for i, want := range wantOrder {
		if players[i].Name != want {
			t.Fatalf("position %d: got %q, want %q (full order %+v)", i, players[i].Name, want, players)
		}
	}
	if players[3].Rank == 1 {
		t.Fatalf("a zero-goal player must never hold rank 1, got %+v", players[3])
	}
}

func TestAssignCompetitionRanksSharesAndSkipsForTies(t *testing.T) {
	// Already ordered by goals desc: 5, 3, 3, 1 -> ranks 1, 2, 2, 4.
	players := []Player{
		{Name: "A", Goals: 5},
		{Name: "B", Goals: 3},
		{Name: "C", Goals: 3},
		{Name: "D", Goals: 1},
	}

	assignCompetitionRanks(players)

	wantRanks := []int{1, 2, 2, 4}
	for i, want := range wantRanks {
		if players[i].Rank != want {
			t.Fatalf("player %q: got rank %d, want %d", players[i].Name, players[i].Rank, want)
		}
	}
}
