package casing

import (
	"strings"
	"unicode"
	"unicode/utf8"
)

// Caser converts strings from camel_case to UpperCamelCase.
type Caser struct {
	acronyms map[string]string
}

func NewCaser() Caser {
	return Caser{
		acronyms: map[string]string{},
	}
}

// AddAcronyms adds each acronym that's specially handled in conversion
// routines.
func (cs Caser) AddAcronyms(acros map[string]string) {
	for a, b := range acros {
		cs.acronyms[a] = b
	}
}

// AddAcronym adds an acronym that's specially handled in conversion routines.
func (cs Caser) AddAcronym(str, acronym string) {
	cs.acronyms[str] = acronym
}

// ToUpperGoIdent converts a string into a legal, capitalized Go identifier,
// respecting registered acronyms. Returns the empty string if no conversion
// is possible.
func (cs Caser) ToUpperGoIdent(s string) string {
	san := sanitize(s)
	if san == "" {
		return ""
	}
	return cs.convert(san, cs.appendUpperCamel)
}

// ToLowerGoIdent converts a string into a legal, uncapitalized Go identifier,
// respecting registered acronyms. Returns the empty string if no conversion
// is possible.
func (cs Caser) ToLowerGoIdent(s string) string {
	san := sanitize(s)
	if san == "" {
		return ""
	}
	con := cs.convert(san, cs.appendLowerCamel)
	switch con {
	case "func", "interface", "select", "case", "defer", "go", "map", "struct",
		"chan", "else", "goto", "package", "switch", "const", "fallthrough", "if",
		"range", "type", "continue", "for", "import", "return", "var":
		return con + "_"
	default:
		return con
	}
}

type converter func(*strings.Builder, []byte)

// convert converts a string using converter for each sub-word while
// respecting the registered acronyms.
func (cs Caser) convert(s string, converter converter) string {
	s = strings.TrimSpace(s)
	if len(s) == 0 {
		return s
	}
	sb := &strings.Builder{}
	sb.Grow(len(s))
	chars := []byte(s)
	lo := 0
	for hi := 0; hi < len(chars); {
		ch, size := utf8.DecodeRune(chars[hi:])
		switch {
		case ch == '_':
			converter(sb, chars[lo:hi])
			lo = hi + size // skip underscore
		case unicode.IsUpper(ch):
			converter(sb, chars[lo:hi])
			lo = hi
		}
		hi += size
	}
	converter(sb, chars[lo:])
	return sb.String()
}

func (cs Caser) appendUpperCamel(sb *strings.Builder, chars []byte) {
	if len(chars) == 0 {
		return
	}
	if a, ok := cs.acronyms[string(chars)]; ok {
		sb.WriteString(a)
		return
	}
	firstCh, size := utf8.DecodeRune(chars)
	sb.WriteRune(unicode.ToUpper(firstCh))
	sb.Write(chars[size:])
}

func (cs Caser) appendLowerCamel(sb *strings.Builder, chars []byte) {
	if len(chars) == 0 {
		return
	}
	isFirst := sb.Len() == 0
	if a, ok := cs.acronyms[string(chars)]; ok {
		if isFirst {
			// First word should be uncapitalized. We don't know exactly how to do
			// that, so assume lower casing the acronym is sufficient.
			sb.WriteString(strings.ToLower(a))
		} else {
			sb.WriteString(a)
		}
		return
	}
	firstCh, size := utf8.DecodeRune(chars)
	if isFirst {
		sb.WriteRune(unicode.ToLower(firstCh))
	} else {
		sb.WriteRune(unicode.ToUpper(firstCh))
	}
	sb.Write(chars[size:])
}
