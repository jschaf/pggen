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

// ToUpperCamel converts a string to UpperCamelCase respecting the registered
// acronyms.
func (cs Caser) ToUpperCamel(s string) string {
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
			cs.appendUpperCamel(sb, chars[lo:hi])
			lo = hi + size // skip underscore
		case unicode.IsUpper(ch):
			cs.appendUpperCamel(sb, chars[lo:hi])
			lo = hi
		}
		hi += size
	}
	cs.appendUpperCamel(sb, chars[lo:])
	return sb.String()
}

// ToUpperGoIdent converts a string into a legal, capitalized Go identifier,
// respecting registered acronyms. // Returns the empty string if no conversion
// is possible.
func (cs Caser) ToUpperGoIdent(s string) string {
	san := sanitize(s)
	if san == "" {
		return ""
	}
	return cs.ToUpperCamel(san)
}
