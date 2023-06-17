package token

import "strconv"

// Token is the minimal set of lexical tokens for SQL that we need to extract
// queries.
type Token int

const (
	Illegal Token = iota
	EOF
	LineComment   // -- foo
	BlockComment  // /* foo */
	String        // 'foo', $$bar$$, $a$baz$a$
	QuotedIdent   // "foo_bar""baz"
	QueryFragment // anything else
	Semicolon     // semicolon ending a query
	Directive     // default value supplied to two argument version of ppgen.arg()
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
	case QuotedIdent:
		return "QuotedIdent"
	case QueryFragment:
		return "QueryFragment"
	case Semicolon:
		return "Semicolon"
	case Directive:
		return "Directive"
	default:
		panic("unhandled token.String(): " + strconv.Itoa(int(t)))
	}
}
