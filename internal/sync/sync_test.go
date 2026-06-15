package sync

import (
	"testing"
	"time"

	"github.com/pocketbase/pocketbase/core"

	"github.com/oyvhov/world-cup-pool/internal/football"
)

func testMatchRecord(stage string) *core.Record {
	collection := core.NewBaseCollection("matches")
	collection.Fields.Add(&core.TextField{Name: "stage", Max: 16})
	collection.Fields.Add(&core.TextField{Name: "status", Max: 16})
	collection.Fields.Add(&core.TextField{Name: "homeTeam", Max: 32})
	collection.Fields.Add(&core.TextField{Name: "awayTeam", Max: 32})
	collection.Fields.Add(&core.TextField{Name: "penWinner", Max: 32})
	collection.Fields.Add(&core.TextField{Name: "advancer", Max: 32})
	collection.Fields.Add(&core.NumberField{Name: "ftHome", OnlyInt: true})
	collection.Fields.Add(&core.NumberField{Name: "ftAway", OnlyInt: true})
	collection.Fields.Add(&core.NumberField{Name: "etHome", OnlyInt: true})
	collection.Fields.Add(&core.NumberField{Name: "etAway", OnlyInt: true})
	collection.Fields.Add(&core.NumberField{Name: "penHome", OnlyInt: true})
	collection.Fields.Add(&core.NumberField{Name: "penAway", OnlyInt: true})
	collection.Fields.Add(&core.DateField{Name: "finalizedAt"})

	record := core.NewRecord(collection)
	record.Set("stage", stage)
	record.Set("homeTeam", "home")
	record.Set("awayTeam", "away")
	return record
}

func TestApplyResultStoresFinishedGroupResult(t *testing.T) {
	record := testMatchRecord("group")

	applyResult(record, "finished", pi(2), pi(1), nil, nil, nil, nil)

	if record.GetString("status") != "finished" {
		t.Fatalf("status = %q, want finished", record.GetString("status"))
	}
	if record.GetInt("ftHome") != 2 || record.GetInt("ftAway") != 1 {
		t.Fatalf("full-time score = %d-%d, want 2-1", record.GetInt("ftHome"), record.GetInt("ftAway"))
	}
	if record.GetInt("etHome") != 0 || record.GetInt("etAway") != 0 || record.GetInt("penHome") != 0 || record.GetInt("penAway") != 0 {
		t.Fatalf("extra-time/penalty defaults were not cleared")
	}
	if record.GetDateTime("finalizedAt").Time().IsZero() {
		t.Fatal("finalizedAt was not set")
	}
	if record.GetString("advancer") != "" {
		t.Fatalf("group advancer = %q, want empty", record.GetString("advancer"))
	}
}

func TestApplyResultDerivesKnockoutAdvancerFromPenalties(t *testing.T) {
	record := testMatchRecord("FINAL")

	applyResult(record, "finished", pi(1), pi(1), pi(2), pi(2), pi(4), pi(3))

	if record.GetString("advancer") != "home" {
		t.Fatalf("advancer = %q, want home", record.GetString("advancer"))
	}
	if record.GetString("penWinner") != "home" {
		t.Fatalf("penWinner = %q, want home", record.GetString("penWinner"))
	}
}

func TestApplyResultDoesNotDeriveAdvancerBeforeFinished(t *testing.T) {
	record := testMatchRecord("R32")

	applyResult(record, "live", pi(2), pi(0), nil, nil, nil, nil)

	if record.GetString("advancer") != "" {
		t.Fatalf("advancer = %q, want empty while live", record.GetString("advancer"))
	}
	if !record.GetDateTime("finalizedAt").Time().IsZero() {
		t.Fatal("finalizedAt was set for a live match")
	}
}

func TestApplyResultClearsFinishedKnockoutStateWhenMovedBackToLive(t *testing.T) {
	record := testMatchRecord("FINAL")

	applyResult(record, "finished", pi(1), pi(1), pi(2), pi(1), nil, nil)
	if record.GetString("advancer") != "home" {
		t.Fatalf("advancer = %q, want home", record.GetString("advancer"))
	}
	if record.GetDateTime("finalizedAt").Time().IsZero() {
		t.Fatal("finalizedAt was not set for finished match")
	}

	applyResult(record, "1H", pi(1), pi(0), nil, nil, nil, nil)

	if record.GetString("status") != "1H" {
		t.Fatalf("status = %q, want 1H", record.GetString("status"))
	}
	if !record.GetDateTime("finalizedAt").Time().IsZero() {
		t.Fatal("finalizedAt was not cleared when match moved back to live")
	}
	if record.GetString("advancer") != "" {
		t.Fatalf("advancer = %q, want empty after moving back to live", record.GetString("advancer"))
	}
	if record.GetString("penWinner") != "" {
		t.Fatalf("penWinner = %q, want empty after moving back to live", record.GetString("penWinner"))
	}
}

func TestResultAlreadyAppliedMatchesExistingFinishedResult(t *testing.T) {
	record := testMatchRecord("group")
	applyResult(record, "finished", pi(2), pi(1), nil, nil, nil, nil)

	if !resultAlreadyApplied(record, "finished", pi(2), pi(1), nil, nil, nil, nil) {
		t.Fatal("expected existing result to be treated as already applied")
	}
}

func TestResultAlreadyAppliedDetectsScoreChange(t *testing.T) {
	record := testMatchRecord("group")
	applyResult(record, "finished", pi(2), pi(1), nil, nil, nil, nil)

	if resultAlreadyApplied(record, "finished", pi(3), pi(1), nil, nil, nil, nil) {
		t.Fatal("changed score was treated as already applied")
	}
}

func TestResultAlreadyAppliedDetectsStaleDerivedKnockoutState(t *testing.T) {
	record := testMatchRecord("FINAL")
	applyResult(record, "finished", pi(1), pi(1), pi(2), pi(2), pi(4), pi(3))
	record.Set("advancer", "")
	record.Set("penWinner", "")

	if resultAlreadyApplied(record, "finished", pi(1), pi(1), pi(2), pi(2), pi(4), pi(3)) {
		t.Fatal("stale derived knockout state was treated as already applied")
	}
}

func TestEventProviderKeyUsesProviderIDsWhenPresent(t *testing.T) {
	base := football.Event{
		Elapsed:  90,
		Extra:    4,
		TeamID:   26,
		Team:     "Argentina",
		PlayerID: 278,
		Player:   "Lionel Messi",
		AssistID: 999,
		Assist:   "Angel Di Maria",
		Type:     "Goal",
		Detail:   "Penalty",
	}

	renamed := base
	renamed.Team = "Argentina National Team"
	renamed.Player = "L. Messi"
	renamed.Assist = "A. Di Maria"

	if eventProviderKey(42, base) != eventProviderKey(42, renamed) {
		t.Fatal("provider key changed despite stable provider ids")
	}

	differentPlayer := base
	differentPlayer.PlayerID = 279
	if eventProviderKey(42, base) == eventProviderKey(42, differentPlayer) {
		t.Fatal("provider key did not change for a different provider player id")
	}

	stoppage := base
	stoppage.Extra = 5
	if eventProviderKey(42, base) == eventProviderKey(42, stoppage) {
		t.Fatal("provider key did not include extra time")
	}
}

func TestKickoffStillNeedsSoonPollingAfterKickoffGraceWindow(t *testing.T) {
	now := time.Date(2026, 6, 11, 19, 27, 0, 0, time.UTC)
	kickoff := now.Add(-27 * time.Minute)

	if !kickoffStillNeedsSoonPolling(now, kickoff) {
		t.Fatal("expected scheduled match shortly after kickoff to keep soon polling")
	}

	if kickoffStillNeedsSoonPolling(now, now.Add(-(providerKickoffGrace + time.Minute))) {
		t.Fatal("stale scheduled match kept soon polling past the grace window")
	}

	if kickoffStillNeedsSoonPolling(now, now.Add(3*time.Hour)) {
		t.Fatal("distant future scheduled match should not keep soon polling")
	}
}
