package handler

import "testing"

func TestEscapeLikeNeutralizesWildcards(t *testing.T) {
	cases := map[string]string{
		"plain":      "plain",
		"100%":       `100\%`,
		"a_b":        `a\_b`,
		`back\slash`: `back\\slash`,
		"%_":         `\%\_`,
	}
	for in, want := range cases {
		if got := escapeLike(in); got != want {
			t.Errorf("escapeLike(%q) = %q, want %q", in, got, want)
		}
	}
	if p := likePattern("50%off"); p != `%50\%off%` {
		t.Errorf("likePattern wildcard bound = %q", p)
	}
}

func TestValidModelPath(t *testing.T) {
	valid := []string{"gpt-4o", "google/gemini-2.0-flash", "a"}
	for _, m := range valid {
		if !validModelPath(m) {
			t.Errorf("validModelPath(%q) should accept", m)
		}
	}
	invalid := []string{"", "..", "a/..", "a/../../b", "m?x", "m#f", "m%20"}
	for _, m := range invalid {
		if validModelPath(m) {
			t.Errorf("validModelPath(%q) should reject", m)
		}
	}
}
