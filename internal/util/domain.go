package util

import (
	"errors"
	"net"
	"net/url"
	"strings"
)

var ErrBadDomain = errors.New("invalid domain")

// NormalizeDomain tries to extract a bare host from inputs like
// "msn.com", "https://msn.com", "msn.com/path", "https://msn.com/ads.txt".
func NormalizeDomain(in string) (string, error) {
	s := strings.TrimSpace(strings.ToLower(in))
	if s == "" {
		return "", ErrBadDomain
	}

	// If it has a scheme, parse directly; otherwise, try prepending http://
	if strings.Contains(s, "://") {
		u, err := url.Parse(s)
		if err != nil || u.Host == "" {
			return "", ErrBadDomain
		}
		host := u.Host
		if h, _, err := net.SplitHostPort(host); err == nil {
			host = h
		}
		return strings.TrimSuffix(host, "."), nil
	}
	// no scheme: could still have path; add http:// for parsing
	u, err := url.Parse("http://" + s)
	if err != nil || u.Host == "" {
		return "", ErrBadDomain
	}
	host := u.Host
	if h, _, err := net.SplitHostPort(host); err == nil {
		host = h
	}
	return strings.TrimSuffix(host, "."), nil
}

