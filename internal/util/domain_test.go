package util

import "testing"

// TestNormalizeDomain validates normalization from various URL-like inputs
// to a bare host and errors on invalid inputs.
// PASS: expected host outputs; errors for empty/bad.
// FAIL: wrong host or missing error.
func TestNormalizeDomain(t *testing.T) {
	cases := []struct{ in, out string }{
		{"msn.com", "msn.com"},
		{"MSN.COM", "msn.com"},
		{"https://msn.com/ads.txt", "msn.com"},
		{"http://msn.com:80/path", "msn.com"},
		{" sub.domain.com ", "sub.domain.com"},
	}
	for _, c := range cases {
		got, err := NormalizeDomain(c.in)
		if err != nil {
			t.Fatalf("err for %q: %v", c.in, err)
		}
		if got != c.out {
			t.Fatalf("%q -> %q want %q", c.in, got, c.out)
		}
	}
	if _, err := NormalizeDomain(""); err == nil {
		t.Fatalf("want error for empty input")
	}
	if _, err := NormalizeDomain("http://"); err == nil {
		t.Fatalf("want error for bad url")
	}
}
