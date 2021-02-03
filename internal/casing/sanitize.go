package casing

import (
	"strings"
	"unicode"
	"unicode/utf8"
)

// sanitize replaces a string with a version safe for using as a Go
// identifier. A Go identifier begins with a unicode letter or underscore and
// is followed by 0 or more unicode letters or digits, or underscores.
// Replaces illegal runes with underscores. Skips leading characters that aren't
// a letter or underscore.
func sanitize(s string) string {
	sb := &strings.Builder{}
	sb.Grow(len(s))
	var firstLetter rune
	secondCharIdx := -1
	// Find first legal starting char, a letter or an underscore.
	for idx, ch := range s {
		if unicode.IsLetter(ch) || ch == '_' {
			firstLetter = ch
			secondCharIdx = idx + utf8.RuneLen(ch)
			break
		}
	}
	if secondCharIdx == -1 {
		return ""
	}

	sb.WriteRune(firstLetter)
	prevUnderscore := firstLetter == '_'
	for _, ch := range s[secondCharIdx:] {
		switch {
		case unicode.IsLetter(ch) || unicode.IsDigit(ch):
			sb.WriteRune(ch)
			prevUnderscore = false
		default:
			if !prevUnderscore {
				sb.WriteRune('_')
			}
			prevUnderscore = true
		}
	}
	if sb.Len() == 0 {
		return ""
	}
	return sb.String()
}
