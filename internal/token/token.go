package token

import "strconv"

// Token is the minimal set of lexical tokens for SQL that we need to extract
// queries.
type Token int

const (
	Illegal Token = iota
	EOF
	LineComment
	BlockComment
	String
	QueryFragment
)

func (t Token) String() string {
	switch t {
	case Illegal:
		return "Illegal"
	case EOF:
		return "EOF"
	case LineComment:
		return "LineComment"
	case BlockComment:
		return "BlockComment"
	case String:
		return "String"
	case QueryFragment:
		return "QueryFragment"
	default:
		panic("unhandled token.String(): " + strconv.Itoa(int(t)))
	}
}
