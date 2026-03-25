package irag

import "testing"

func TestPublicProviderCode(t *testing.T) {
	tests := map[string]string{
		"nexure":    "n",
		"ryzumi":    "r",
		"kanata":    "k",
		"ytdlp":     "y",
		"chocomilk": "c",
		"unknown":   "",
	}

	for name, want := range tests {
		t.Run(name, func(t *testing.T) {
			if got := publicProviderCode(name); got != want {
				t.Fatalf("publicProviderCode(%q) = %q, want %q", name, got, want)
			}
		})
	}
}

func TestPublicProviderCodes(t *testing.T) {
	got := publicProviderCodes([]string{"nexure", "unknown", "ryzumi", "kanata"})
	want := []string{"n", "r", "k"}

	if len(got) != len(want) {
		t.Fatalf("publicProviderCodes length = %d, want %d (%v)", len(got), len(want), got)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("publicProviderCodes[%d] = %q, want %q (full=%v)", i, got[i], want[i], got)
		}
	}
}
