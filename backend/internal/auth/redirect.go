package auth

import (
	"net/url"
	"strings"
)

// SafeReturnTo accepts only same-site absolute paths (prevents open redirects); anything else becomes "/".
func SafeReturnTo(raw string) string {
	if raw == "" || len(raw) > 300 || !strings.HasPrefix(raw, "/") {
		return "/"
	}
	// "//evil.com" and "/\evil.com" are protocol-relative in browsers.
	if strings.HasPrefix(raw, "//") || strings.HasPrefix(raw, "/\\") || strings.ContainsAny(raw, "\r\n\t\x00") {
		return "/"
	}
	u, err := url.Parse(raw)
	if err != nil || u.Scheme != "" || u.Host != "" || u.User != nil {
		return "/"
	}
	// Never bounce back into the auth flow itself.
	if strings.HasPrefix(u.Path, "/auth/") || u.Path == "/login" {
		return "/"
	}
	return raw
}
