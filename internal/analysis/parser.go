package analysis

import (
	"bufio"
	"bytes"
	"strings"
)

// ParseAdsTxt returns a map[seller_domain]count.
// Rules:
// - Ignore empty lines and full-line comments (# ...)
// - Strip inline comments starting with #
// - Split by comma, trim spaces; first token is the seller domain
// - Ignore lines that do not look like CSV of at least 1 token
// - Ignore directive-style lines that contain '=' before the first comma (e.g., contact=)
func ParseAdsTxt(b []byte) map[string]int {
	counts := make(map[string]int)
	s := bufio.NewScanner(bytes.NewReader(b))
	buf := make([]byte, 0, 1024*1024) // allow long lines up to 1MB
	s.Buffer(buf, 1024*1024)
	for s.Scan() {
		line := s.Text()
		line = strings.TrimSpace(line)
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
		// ignore directive-like lines (no comma but contains '=')
		if !strings.Contains(line, ",") && strings.Contains(line, "=") {
			continue
		}

		parts := strings.Split(line, ",")
		if len(parts) == 0 {
			continue
		}
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
		counts[p0]++
	}
	return counts
}
