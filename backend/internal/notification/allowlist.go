package notification

import (
	"errors"
	"strings"
)

// BlockedDomain can never receive mail from this test system, whatever an Admin configures.
const BlockedDomain = "@g-asiapac.com.vn"

var ErrNotAllowed = errors.New("recipient not in the test allowlist")

func normEmail(s string) string { return strings.ToLower(strings.TrimSpace(s)) }

// CheckTestAllowlist is the final gate before any real send: exactly the configured TO, only configured
// CC addresses, no BCC, never the company domain.
func CheckTestAllowlist(to string, cc, bcc []string, allowedTo string, allowedCC []string) error {
	if allowedTo == "" || normEmail(to) != normEmail(allowedTo) || len(bcc) > 0 {
		return ErrNotAllowed
	}
	ok := map[string]bool{}
	for _, a := range allowedCC {
		ok[normEmail(a)] = true
	}
	for _, c := range cc {
		if !ok[normEmail(c)] {
			return ErrNotAllowed
		}
	}
	for _, a := range append([]string{to}, cc...) {
		if strings.HasSuffix(normEmail(a), BlockedDomain) {
			return ErrNotAllowed
		}
	}
	return nil
}
