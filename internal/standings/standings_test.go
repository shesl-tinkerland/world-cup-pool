package standings

import (
	"reflect"
	"testing"
)

// case 1: A and B finish equal on overall points/gd/gf; A beat B head-to-head,
// so A ranks first. C also has 6 pts but a worse goal difference, so it is
// third on the overall criteria — and note C actually beat A head-to-head,
// which must NOT lift it above A (head-to-head only applies among teams tied on
// the overall criteria).
func TestHeadToHeadBreaksOverallTie(t *testing.T) {
	ms := []Match{
		{"A", "A", "B", 2, 1},
		{"A", "A", "C", 0, 1},
		{"A", "A", "D", 2, 0},
		{"A", "B", "C", 2, 0},
		{"A", "B", "D", 1, 0},
		{"A", "C", "D", 1, 0},
	}
	order, _, ambiguous := GroupTables(ms)
	if got, want := order["A"], []string{"A", "B", "C", "D"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("order = %v, want %v", got, want)
	}
	if len(ambiguous) != 0 {
		t.Fatalf("ambiguous = %v, want none", ambiguous)
	}
}

// case 3: P and Q are identical on every numeric criterion, including their
// head-to-head (a draw). The group must be reported ambiguous and fall back to
// a deterministic team-id order (P before Q).
func TestUnbreakableTieIsAmbiguousAndDeterministic(t *testing.T) {
	ms := []Match{
		{"B", "P", "Q", 0, 0},
		{"B", "P", "R", 2, 0},
		{"B", "P", "S", 2, 0},
		{"B", "Q", "R", 2, 0},
		{"B", "Q", "S", 2, 0},
		{"B", "R", "S", 1, 0},
	}
	order, _, ambiguous := GroupTables(ms)
	if got, want := order["B"], []string{"P", "Q", "R", "S"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("order = %v, want %v", got, want)
	}
	found := false
	for _, g := range ambiguous {
		if g == "B" {
			found = true
		}
	}
	if !found {
		t.Fatalf("ambiguous = %v, want it to contain group B", ambiguous)
	}
}

// Incomplete groups (a team has played fewer than 3 matches) are excluded from
// the table entirely.
func TestIncompleteGroupExcluded(t *testing.T) {
	ms := []Match{
		{"C", "W", "X", 1, 0},
		{"C", "Y", "Z", 1, 0},
	}
	order, thirds, _ := GroupTables(ms)
	if _, ok := order["C"]; ok {
		t.Fatalf("incomplete group C should be excluded, got %v", order["C"])
	}
	if len(thirds) != 0 {
		t.Fatalf("thirds = %v, want none from an incomplete group", thirds)
	}
}

// The third-placed teams are ranked globally on the overall criteria across
// groups (they never meet, so no head-to-head applies).
func TestThirdsRankedGloballyByOverall(t *testing.T) {
	// Two complete groups. Group A's third has a better goal difference than
	// group B's third, so it ranks first among thirds.
	ms := []Match{
		// Group A: third place (C) finishes on -1.
		{"A", "A", "B", 1, 0}, {"A", "A", "C", 1, 0}, {"A", "A", "D", 1, 0},
		{"A", "B", "C", 1, 0}, {"A", "B", "D", 1, 0}, {"A", "C", "D", 1, 0},
		// Group B: third place (G) finishes on a worse record.
		{"B", "E", "F", 3, 0}, {"B", "E", "G", 3, 0}, {"B", "E", "H", 3, 0},
		{"B", "F", "G", 3, 0}, {"B", "F", "H", 3, 0}, {"B", "G", "H", 3, 0},
	}
	_, thirds, _ := GroupTables(ms)
	if len(thirds) != 2 {
		t.Fatalf("got %d thirds, want 2", len(thirds))
	}
	// C (group A third, gd -1) should outrank G (group B third, gd -3).
	if thirds[0].TeamID != "C" || thirds[1].TeamID != "G" {
		t.Fatalf("thirds order = [%s, %s], want [C, G]", thirds[0].TeamID, thirds[1].TeamID)
	}
}

func TestThirdsCutAmbiguous(t *testing.T) {
	// 9 thirds where the 8th and 9th are equal on every numeric criterion: who
	// advances would be decided by fair play / lots.
	rows := make([]Row, 9)
	for i := range rows {
		rows[i] = Row{Pts: 9 - i, GD: 1, GF: 1}
	}
	rows[7] = Row{Pts: 1, GD: 0, GF: 1}
	rows[8] = Row{Pts: 1, GD: 0, GF: 1}
	if !thirdsCutAmbiguous(rows) {
		t.Fatal("expected the 8th/9th third tie to be flagged ambiguous")
	}
	rows[8] = Row{Pts: 0, GD: -1, GF: 0}
	if thirdsCutAmbiguous(rows) {
		t.Fatal("a separated 8th/9th cut must not be flagged ambiguous")
	}
}
