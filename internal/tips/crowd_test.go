package tips

import "testing"

func TestPctSplitSumsTo100(t *testing.T) {
	cases := []struct {
		name          string
		h, d, a       int
		wantH, wantD, wantA int
	}{
		{"empty", 0, 0, 0, 0, 0, 0},
		{"equal thirds give rounding to largest", 1, 1, 1, 34, 33, 33}, // ties → home wins remainder
		{"clear majority home", 7, 2, 1, 70, 20, 10},
		{"draw majority", 1, 5, 1, 14, 72, 14},
		{"away majority", 1, 0, 2, 33, 0, 67},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			total := tc.h + tc.d + tc.a
			gh, gd, ga := pctSplit(tc.h, tc.d, tc.a, total)
			if total > 0 && gh+gd+ga != 100 {
				t.Fatalf("percentages must sum to 100, got %d+%d+%d=%d", gh, gd, ga, gh+gd+ga)
			}
			if gh != tc.wantH || gd != tc.wantD || ga != tc.wantA {
				t.Fatalf("pctSplit(%d,%d,%d) = (%d,%d,%d) want (%d,%d,%d)",
					tc.h, tc.d, tc.a, gh, gd, ga, tc.wantH, tc.wantD, tc.wantA)
			}
		})
	}
}
