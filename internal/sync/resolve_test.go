package sync

import (
	"testing"

	"github.com/pocketbase/pocketbase/core"
)

func knockoutRecord(num int, stage, homeLabel, awayLabel string) *core.Record {
	record := testMatchRecord(stage)
	record.Set("num", num)
	record.Set("status", "scheduled")
	record.Set("homeTeam", "")
	record.Set("awayTeam", "")
	record.Set("homeLabel", homeLabel)
	record.Set("awayLabel", awayLabel)
	return record
}

func groupRecord(letter string) *core.Record {
	record := testMatchRecord("group")
	record.Set("groupLetter", letter)
	return record
}

func TestBestThirdAssignmentsWaitsForEveryGroupToComplete(t *testing.T) {
	matches := []*core.Record{
		knockoutRecord(74, "R32", "1E", "3A/B/C/D/F"),
	}
	for _, letter := range []string{"A", "B", "C", "D", "E", "F", "G", "H", "I", "J", "K", "L"} {
		matches = append(matches, groupRecord(letter))
	}
	thirds := []standing{
		{group: "A", team: "third-a"},
		{group: "B", team: "third-b"},
		{group: "C", team: "third-c"},
		{group: "D", team: "third-d"},
		{group: "E", team: "third-e"},
		{group: "F", team: "third-f"},
		{group: "G", team: "third-g"},
		{group: "H", team: "third-h"},
	}
	thirdTeams := map[string]string{
		"A": "third-a",
		"B": "third-b",
		"C": "third-c",
		"D": "third-d",
		"E": "third-e",
		"F": "third-f",
		"G": "third-g",
		"H": "third-h",
	}

	got := bestThirdAssignments(matches, thirdTeams, thirds, 8)

	if len(got) != 0 {
		t.Fatalf("best-third assignments = %v, want none before all groups complete", got)
	}
}

func TestThirdAssignmentsUsesOfficialTableWhenGroupsAreComplete(t *testing.T) {
	matches := []*core.Record{
		knockoutRecord(74, "R32", "1E", "3A/B/C/D/F"),
	}
	quals := []string{"A", "B", "C", "D", "E", "F", "G", "H"}
	thirdTeams := map[string]string{
		"A": "third-a",
		"B": "third-b",
		"C": "third-c",
		"D": "third-d",
		"E": "third-e",
		"F": "third-f",
		"G": "third-g",
		"H": "third-h",
	}

	got := thirdAssignments(matches, quals, thirdTeams)

	if got[74] != "third-c" {
		t.Fatalf("R32 third assignment = %q, want third-c", got[74])
	}
}

func TestThirdAssignmentsWaitsForEightQualifiers(t *testing.T) {
	matches := []*core.Record{
		knockoutRecord(74, "R32", "1E", "3A/B/C/D/F"),
	}
	thirdTeams := map[string]string{
		"A": "third-a",
		"B": "third-b",
		"C": "third-c",
	}

	got := thirdAssignments(matches, []string{"A", "B", "C"}, thirdTeams)

	if len(got) != 0 {
		t.Fatalf("third assignments = %v, want none before all best thirds are known", got)
	}
}

func TestReconcileParticipantCorrectsScheduledStaleTeam(t *testing.T) {
	match := knockoutRecord(85, "R32", "1B", "3E/F/G/I/J")
	match.Set("awayTeam", "bosnia")

	got := reconcileParticipant(match, "awayTeam", "3E/F/G/I/J", "third-j")

	if !got.changed {
		t.Fatal("expected stale scheduled participant to be corrected")
	}
	if !got.invalidatesTips {
		t.Fatal("expected stale participant correction to invalidate tips")
	}
	if got := match.GetString("awayTeam"); got != "third-j" {
		t.Fatalf("awayTeam = %q, want third-j", got)
	}
}

func TestReconcileParticipantClearsScheduledStaleUnresolvedTeam(t *testing.T) {
	match := knockoutRecord(85, "R32", "1B", "3E/F/G/I/J")
	match.Set("awayTeam", "bosnia")

	got := reconcileParticipant(match, "awayTeam", "3E/F/G/I/J", "")

	if !got.changed {
		t.Fatal("expected stale scheduled participant to be cleared")
	}
	if !got.invalidatesTips {
		t.Fatal("expected stale participant clear to invalidate tips")
	}
	if got := match.GetString("awayTeam"); got != "" {
		t.Fatalf("awayTeam = %q, want empty", got)
	}
}

func TestReconcileParticipantDoesNotRewriteLiveTeam(t *testing.T) {
	match := knockoutRecord(85, "R32", "1B", "3E/F/G/I/J")
	match.Set("status", "live")
	match.Set("awayTeam", "bosnia")

	if got := reconcileParticipant(match, "awayTeam", "3E/F/G/I/J", "third-j"); got.changed {
		t.Fatal("did not expect live participant to be rewritten")
	}
	if got := match.GetString("awayTeam"); got != "bosnia" {
		t.Fatalf("awayTeam = %q, want bosnia", got)
	}
}

func TestReconcileParticipantFillingEmptySlotDoesNotInvalidateTips(t *testing.T) {
	match := knockoutRecord(85, "R32", "1B", "3E/F/G/I/J")

	got := reconcileParticipant(match, "awayTeam", "3E/F/G/I/J", "third-j")

	if !got.changed {
		t.Fatal("expected empty participant to be filled")
	}
	if got.invalidatesTips {
		t.Fatal("did not expect first-time participant fill to invalidate tips")
	}
}
