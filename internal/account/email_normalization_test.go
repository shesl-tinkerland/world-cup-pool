package account

import "testing"

func TestNormalizeEmailAddress(t *testing.T) {
	tests := map[string]string{
		" Example@Mail.COM ": "example@mail.com",
		"example@mail.com":   "example@mail.com",
		"":                   "",
		"   ":                "",
	}

	for input, want := range tests {
		if got := normalizeEmailAddress(input); got != want {
			t.Fatalf("normalizeEmailAddress(%q) = %q, want %q", input, got, want)
		}
	}
}
