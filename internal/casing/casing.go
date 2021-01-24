package casing

import "strings"

type Caser struct {
	acronyms map[string]string
}

func NewCaser() Caser {
	return Caser{
		acronyms: map[string]string{},
	}
}

func (cs Caser) AddAcronyms(acros map[string]string) {
	for a, b := range acros {
		cs.acronyms[a] = b
	}
}

func (cs Caser) AddAcronym(str, acronym string) {
	cs.acronyms[str] = acronym
}

func isLower(ch byte) bool { return 'a' <= ch && ch <= 'z' }
func isUpper(ch byte) bool { return 'A' <= ch && ch <= 'Z' }

func upper(ch byte) byte { return ch - ('a' - 'A') } // returns upper-case ch iff ch is ASCII letter

func (cs Caser) appendUpperCamel(sb *strings.Builder, chars []byte, lo, hi int) {
	if lo == hi {
		return
	}
	wordChars := chars[lo:hi]
	word := string(wordChars)
	if a, ok := cs.acronyms[word]; ok {
		sb.WriteString(a)
		return
	}
	if isLower(chars[lo]) {
		sb.WriteByte(upper(chars[lo]))
	} else {
		sb.WriteByte(chars[lo])
	}
	for i := lo + 1; i < hi; i++ {
		sb.WriteByte(chars[i])
	}
}

func (cs Caser) ToUpperCamel(s string) string {
	s = strings.TrimSpace(s)
	sb := &strings.Builder{}
	sb.Grow(len(s))
	// Find underscore delimited word.
	chars := []byte(s)
	lo, hi := 0, 0
	for i, ch := range chars {
		hi = i
		switch {
		case ch == '_':
			cs.appendUpperCamel(sb, chars, lo, hi)
			lo = i + 1 // skip underscore
		case isUpper(ch):
			cs.appendUpperCamel(sb, chars, lo, hi)
			lo = i
		}
	}
	cs.appendUpperCamel(sb, chars, lo, hi+1)
	return sb.String()
}
