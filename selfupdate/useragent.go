package selfupdate

import (
	"runtime"
	"strings"
	"unicode"
	"unicode/utf8"
)

// UserAgent returns "product/version (goos/goarch)" for
// GitHubOptions.UserAgent
// (docs/decisions/0004-MADR-evolve-selfupdate-api-and-tui-support.md §5).
// Control characters and invalid UTF-8 are dropped from product and version,
// each run of space, "/", "(" or ")" becomes one "-", and an empty result is
// "unknown". The result always passes NewGitHubSource's User-Agent check.
func UserAgent(product, version string) string {
	return uaToken(product) + "/" + uaToken(version) + " (" + runtime.GOOS + "/" + runtime.GOARCH + ")"
}

// uaUnknown stands in for an empty User-Agent token.
const uaUnknown = "unknown"

// uaToken sanitises one User-Agent token.
func uaToken(s string) string {
	var b strings.Builder
	sep := false
	for s != "" {
		r, size := utf8.DecodeRuneInString(s)
		s = s[size:]
		switch {
		case r == utf8.RuneError && size == 1, unicode.IsControl(r):
			continue
		case unicode.IsSpace(r) || r == '/' || r == '(' || r == ')':
			sep = true
			continue
		}
		if sep && b.Len() > 0 {
			b.WriteByte('-')
		}
		sep = false
		b.WriteRune(r)
	}
	out := strings.Trim(b.String(), "-")
	if out == "" {
		return uaUnknown
	}
	return out
}
