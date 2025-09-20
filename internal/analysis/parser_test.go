package analysis

import "testing"

// TestParseAdsTxt_Basics checks that parser:
// - ignores comments and blank lines,
// - strips inline comments,
// - counts first CSV token (seller domain),
// - ignores malformed domains.
// PASS: counts match expected for google/appnexus/rubicon; "notadomain" skipped.
// FAIL: any mismatch in counts or inclusion of invalid domain.
func TestParseAdsTxt_Basics(t *testing.T) {
	in := []byte(`
# comment
\n
google.com, pub-123, DIRECT # inline
appnexus.com, 123, RESELLER , f5ab1-
contact=ads@example.com
rubiconproject.com, abc, DIRECT
google.com, pub-456, RESELLER
badline without commas
notadomain, x, DIRECT
`)
	m := ParseAdsTxt(in)
	if m["google.com"] != 2 {
		t.Fatalf("google count=%d", m["google.com"])
	}
	if m["appnexus.com"] != 1 {
		t.Fatalf("appnexus count=%d", m["appnexus.com"])
	}
	if m["rubiconproject.com"] != 1 {
		t.Fatalf("rubicon count=%d", m["rubiconproject.com"])
	}
	if _, ok := m["notadomain"]; ok {
		t.Fatalf("should skip invalid domain without dot")
	}
}

// TestParseAdsTxt_InlineCommentsAndDirectives ensures directive-style lines
// (e.g., contact=) are skipped and valid lines still counted.
// PASS: only google.com and appnexus.com counted once each.
// FAIL: directives included or counts incorrect.
func TestParseAdsTxt_InlineCommentsAndDirectives(t *testing.T) {
	in := []byte(`
# full comment
contact=ads@example.com
seller_id=123
google.com, pub-1, DIRECT # comment
# another
appnexus.com, 1, RESELLER, abc
`)
	m := ParseAdsTxt(in)
	if len(m) != 2 {
		t.Fatalf("len=%d want 2", len(m))
	}
	if m["google.com"] != 1 || m["appnexus.com"] != 1 {
		t.Fatalf("bad counts: %#v", m)
	}
}
