package analysis

import (
	"bufio"
	"bytes"
	"strings"
)

// ParseAdsTxt returns a map[seller_domain]count.
//
// Rules implemented:
// - Ignore empty lines and full-line comments (# ...)
// - Strip inline comments starting with #
// - Ignore directive-style lines where '=' appears before the first comma
//   (e.g., "contact=...", "subdomain=..."), or no comma but has '='
// - Split by comma (SplitN=2), trim spaces; first token is the seller domain
// - Naive domain sanity: non-empty, lowercased, no spaces, has a dot,
//   no '@', and not starting/ending with '.' or '-'
func ParseAdsTxt(b []byte) map[string]int {
	counts := make(map[string]int)
	s := bufio.NewScanner(bytes.NewReader(b))
	buf := make([]byte, 0, 1024*1024) // allow long lines up to 1MB
	s.Buffer(buf, 1024*1024)
	for s.Scan() {
		line := strings.TrimSpace(s.Text())
		if line == "" {
			continue
		}
		if strings.HasPrefix(line, "#") {
			continue
		}

		// strip inline comments
		if i := strings.Index(line, "#"); i >= 0 {
			line = strings.TrimSpace(line[:i])
		}
		if line == "" {
			continue
		}

		// Skip if there's an '=' BEFORE the first comma,
		// or if there's no comma at all but line contains '='.
		comma := strings.IndexByte(line, ',')
		eq := strings.IndexByte(line, '=')
		if eq >= 0 && (comma == -1 || eq < comma) {
			continue
		}

		// We only need the first field (seller domain),
		// so split at most once.
		parts := strings.SplitN(line, ",", 2)
		p0 := strings.ToLower(strings.TrimSpace(parts[0]))
		if p0 == "" {
			continue
		}
		if strings.Contains(p0, " ") { // malformed domain
			continue
		}
		// naive domain shape check (contains a dot)
		if !strings.Contains(p0, ".") {
			continue
		}

		// domains shouldn't contain '@' and shouldn't start/end with '.' or '-'
		if strings.Contains(p0, "@") {
			continue
		}
		if p0[0] == '.' || p0[len(p0)-1] == '.' || p0[0] == '-' || p0[len(p0)-1] == '-' {
			continue
		}
		counts[p0]++
	}
	return counts
}
