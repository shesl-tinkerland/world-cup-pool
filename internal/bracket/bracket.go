// Package bracket holds FIFA's official 2026 best-third → Round-of-32
// allocation table (Annex C of the tournament regulations / Wikipedia
// transcription): for each of the 495 combinations of 8 qualifying
// third-placed groups, which group's third faces each group winner.
package bracket

import (
	_ "embed"
	"encoding/json"
	"sort"
	"strings"
)

//go:embed data/third_table.json
var rawTable []byte

// table[sortedQualifierLetters] = { winnerGroupLetter: thirdGroupLetter }
var table map[string]map[string]string

func init() {
	if err := json.Unmarshal(rawTable, &table); err != nil {
		panic("bracket: bad third_table.json: " + err.Error())
	}
}

// Key normalises a set of qualifying third-place group letters to the table
// key (sorted, upper-case, deduped).
func Key(groups []string) string {
	seen := map[string]bool{}
	out := make([]string, 0, len(groups))
	for _, g := range groups {
		g = strings.ToUpper(strings.TrimSpace(g))
		if g != "" && !seen[g] {
			seen[g] = true
			out = append(out, g)
		}
	}
	sort.Strings(out)
	return strings.Join(out, "")
}

// ThirdFor returns the group letter whose third-placed team faces the given
// group winner, for the given set of 8 qualifying third groups. ok is false
// if the combination isn't exactly 8 groups or isn't in the official table
// (callers should fall back to a deterministic matching).
func ThirdFor(qualifiers []string, winner string) (string, bool) {
	if len(qualifiers) != 8 {
		return "", false
	}
	m, ok := table[Key(qualifiers)]
	if !ok {
		return "", false
	}
	g, ok := m[strings.ToUpper(winner)]
	return g, ok
}

// Lookup returns the full winner→thirdGroup map for a combination.
func Lookup(qualifiers []string) (map[string]string, bool) {
	if len(qualifiers) != 8 {
		return nil, false
	}
	m, ok := table[Key(qualifiers)]
	return m, ok
}

// WinnerLetter returns the group-winner letter ("1X" -> "X") of a knockout
// match's two labels, i.e. the side a third-placed team is drawn against.
func WinnerLetter(homeLabel, awayLabel string) (string, bool) {
	for _, l := range []string{homeLabel, awayLabel} {
		if len(l) == 2 && l[0] == '1' && l[1] >= 'A' && l[1] <= 'L' {
			return string(l[1]), true
		}
	}
	return "", false
}

// Table exposes the whole official table (served to the frontend so its
// Forecast bracket uses the identical allocation).
func Table() map[string]map[string]string { return table }
