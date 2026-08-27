package auth

import "strings"

// NormalizeEmail is the canonical representation used for account and invitation matching.
func NormalizeEmail(email string) string {
	return strings.ToLower(strings.TrimSpace(email))
}
