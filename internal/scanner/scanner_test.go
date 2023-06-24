package scanner

import (
	gotok "go/token"
	"testing"

	"github.com/jschaf/pggen/internal/token"
	"github.com/stretchr/testify/assert"
)

func newlineCount(s string) int {
	n := 0
	for i := 0; i < len(s); i++ {
		if s[i] == '\n' {
			n++
		}
	}
	return n
}

// stringTok represents a token that's used for testing.
type stringTok struct {
	t   token.Token
	lit string
	raw string
}

func (st stringTok) newlineCount() int {
	return newlineCount(st.raw)
}

func (st stringTok) size() int {
	if st.raw == "" {
		return len(st.lit)
	}
	return len(st.raw)
}

type errorCollector struct {
	cnt  int              // number of errors encountered
	msgs []string         // all error messages encountered
	pos  []gotok.Position // error positions encountered
}

func (ec *errorCollector) asHandler() ErrorHandler {
	return func(pos gotok.Position, msg string) {
		ec.cnt++
		ec.msgs = append(ec.msgs, msg)
		ec.pos = append(ec.pos, pos)
	}
}

func frag(lit string) stringTok      { return stringTok{t: token.QueryFragment, lit: lit, raw: lit} }
func str(lit string) stringTok       { return stringTok{t: token.String, lit: lit, raw: lit} }
func ident(ident string) stringTok   { return stringTok{t: token.QuotedIdent, lit: ident, raw: ident} }
func directive(lit string) stringTok { return stringTok{t: token.Directive, lit: lit, raw: lit} }
func illegal(lit string) stringTok   { return stringTok{t: token.Illegal, lit: lit, raw: lit} }

func TestScanner_Scan(t *testing.T) {
	type testCase struct {
		lit  string
		toks []stringTok
		errs []string
	}
	tests := []testCase{
		{"", nil, nil},
		{"-- abc", []stringTok{{t: token.LineComment, lit: "-- abc"}}, nil},
		{"'foo'", []stringTok{{t: token.String, lit: "'foo'"}}, nil},
		{"'foo''bar'", []stringTok{str("'foo''bar'")}, nil},
		{`"foo_bar"`, []stringTok{ident(`"foo_bar"`)}, nil},
		{`"foo$$ $$bar"`, []stringTok{ident(`"foo$$ $$bar"`)}, nil},
		{`"foo""bar"`, []stringTok{ident(`"foo""bar"`)}, nil},
		{`"fo$o"`, []stringTok{ident(`"fo$o"`)}, nil},
		{"/* abc */", []stringTok{{t: token.BlockComment, lit: "/* abc */"}}, nil},
		{"/* /* abc */ */", []stringTok{{t: token.BlockComment, lit: "/* /* abc */ */"}}, nil},
		{"SELECT 1", []stringTok{frag("SELECT 1")}, nil},
		{"SELECT pggen.arg('arg1')", []stringTok{frag("SELECT "), directive("pggen.arg('arg1')")}, nil},
		{"SELECT pggen.arg('arg2', null::int)", []stringTok{frag("SELECT "), directive("pggen.arg('arg2', null::int)")}, nil},
		{"SELECT pggen.arg('arg2', pggen.arg('arg3'))", []stringTok{frag("SELECT "), illegal("pggen.arg('arg2', ")}, []string{"illegal pggen.arg() expression -- nested use of pggen.arg() is not allowed: pggen.arg('arg2', "}},
		{"SELECT pggen.arg('arg2', exists(SELECT 1 FROM bar))", []stringTok{frag("SELECT "), directive("pggen.arg('arg2', exists(SELECT 1 FROM bar))")}, nil},
		{"SELECT pggen.arg('arg2', exists(SELECT '}'\n-- test comment }\n/* test comment }*/ FROM bar))", []stringTok{frag("SELECT "), directive("pggen.arg('arg2', exists(SELECT '}'\n-- test comment }\n/* test comment }*/ FROM bar))")}, nil},
		{"SELECT abc$", []stringTok{frag("SELECT abc$")}, nil},
		{"SELECT a$$bc", []stringTok{frag("SELECT a$$bc")}, nil},
		{"SELECT a$$$bc", []stringTok{frag("SELECT a$$$bc")}, nil},
		{"SELECT abc$foo", []stringTok{frag("SELECT abc$foo")}, nil},
		{"SELECT $$a$$", []stringTok{frag("SELECT "), str("$$a$$")}, nil},
		{"SELECT func($$a$$)", []stringTok{frag("SELECT func("), str("$$a$$"), frag(")")}, nil},
		{"SELECT 'a'||$$a$$", []stringTok{frag("SELECT "), str("'a'"), frag("||"), str("$$a$$")}, nil},
		{"SELECT '`\\n' as \"$\"", []stringTok{
			frag("SELECT "),
			{t: token.String, lit: "'`\\n'", raw: "'`\\n' "}, // consumes trailing space
			frag("as "),
		}, nil},
	}
	for _, tt := range tests {
		t.Run(tt.lit, func(t *testing.T) {
			ec := &errorCollector{} // error handler

			// init scanner
			fset := gotok.NewFileSet()
			var s Scanner
			s.Init(fset.AddFile("", fset.Base(), len(tt.lit)), []byte(tt.lit), ec.asHandler())

			// setup expected position
			wantPos := gotok.Position{
				Filename: "",
				Offset:   0,
				Line:     1,
				Column:   1,
			}

			for i, wantTok := range tt.toks {
				p, tok, lit := s.Scan()
				t.Logf("index %2d, gotTok: %-14s wantTok: %-14s  gotLit: %-5q wantLit: %q",
					i, tok, wantTok.t, lit, wantTok.lit)
				pos := fset.Position(p)

				checkPosOffset(t, wantPos, pos, lit)
				checkPosLine(t, wantPos, pos, lit)
				checkToken(t, wantTok.t, tok, lit)
				checkLiteral(t, wantTok.lit, lit)

				wantPos.Offset += wantTok.size()
				wantPos.Line += wantTok.newlineCount()
			}

			assert.Equal(t, tt.errs, ec.msgs, "error messages should match")
		})
	}

}

func checkPosLine(t *testing.T, want, got gotok.Position, lit string) {
	t.Helper()
	if got.Line != want.Line {
		t.Errorf("bad line for %q: got %d, expected %d", lit, got.Line, want.Line)
	}
}

func checkPosOffset(t *testing.T, want, got gotok.Position, lit string) {
	t.Helper()
	if got.Offset != want.Offset {
		t.Errorf("bad position for %q: got %d, expected %d", lit, got.Offset, want.Offset)
	}
}

func checkToken(t *testing.T, want, got token.Token, lit string) {
	t.Helper()
	if got != want {
		t.Errorf("bad token for %q: got %s, expected %s", lit, got, want)
	}
}
func checkLiteral(t *testing.T, wantLit string, gotLit string) {
	t.Helper()
	if wantLit != gotLit {
		t.Errorf("bad literal: got %q, expected %q", gotLit, wantLit)
	}
}
