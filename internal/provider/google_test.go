package provider

import "testing"

func TestSanitizeModelPath(t *testing.T) {
	ok := []string{"gemini-2.0-flash", "models/gemini-1.5-pro", "a", "meta.llama-3_70b", "x~y"}
	for _, m := range ok {
		if got := sanitizeModelPath(m); got != m {
			t.Errorf("sanitizeModelPath(%q) = %q, want unchanged", m, got)
		}
	}
	bad := []string{
		"", " ", "../etc/passwd", "a/../b", "gemini?key=1", "gemini#frag",
		"gemini%2F", "with space", "a/with space", ".",
	}
	for _, m := range bad {
		if got := sanitizeModelPath(m); got != "unknown-model" {
			t.Errorf("sanitizeModelPath(%q) = %q, want rejected placeholder", m, got)
		}
	}
}
