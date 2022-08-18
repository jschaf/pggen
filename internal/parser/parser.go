package parser

import (
	"fmt"
	"github.com/jschaf/pggen/internal/ast"
	"github.com/jschaf/pggen/internal/scanner"
	"github.com/jschaf/pggen/internal/token"
	goscan "go/scanner"
	gotok "go/token"
	"regexp"
	"strconv"
	"strings"
)

type parser struct {
	file    *gotok.File
	errors  goscan.ErrorList
	scanner scanner.Scanner
	src     []byte // original source

	// Tracing and debugging
	mode   Mode // parsing mode
	trace  bool // == (mode & Trace != 0)
	indent int  // indentation used for tracing output

	// Comments
	comments    []*ast.CommentGroup
	leadComment *ast.CommentGroup // last lead comment

	// Next token
	pos gotok.Pos   // token position
	tok token.Token // one token look-ahead
	lit string      // token literal
}

func (p *parser) init(fset *gotok.FileSet, filename string, src []byte, mode Mode) {
	p.file = fset.AddFile(filename, -1, len(src))
	eh := func(pos gotok.Position, msg string) { p.errors.Add(pos, msg) }
	p.scanner.Init(p.file, src, eh)
	p.src = src

	p.mode = mode
	p.trace = mode&Trace != 0 // for convenience (p.trace is used frequently)

	p.next() // parse overall doc comments
}

// Parsing support

func (p *parser) printTrace(a ...interface{}) {
	const dots = ". . . . . . . . . . . . . . . . . . . . . . . . . . . . . . . . "
	const n = len(dots)
	pos := p.file.Position(p.pos)
	fmt.Printf("%5d:%3d: ", pos.Line, pos.Column)
	i := 2 * p.indent
	for i > n {
		fmt.Print(dots)
		i -= n
	}
	// i <= n
	fmt.Print(dots[0:i])
	fmt.Println(a...)
}

func trace(p *parser, msg string) *parser {
	p.printTrace(msg, "(")
	p.indent++
	return p
}

// Usage pattern: defer un(trace(p, "..."))
func un(p *parser) {
	p.indent--
	p.printTrace(")")
}

// Advance to the next token.
func (p *parser) next0() {
	// Because of one-token look-ahead, print the previous token when tracing as
	// it provides a more readable output. The very first token (!p.pos.IsValid())
	// is not initialized (it is token.ILLEGAL), so don't print it.
	if p.trace && p.pos.IsValid() {
		s := p.tok.String()
		switch {
		case p.tok == token.String || p.tok == token.QueryFragment:
			lit := p.lit
			// Simplify trace expression.
			if lit != "" {
				lit = `"` + lit + `"`
			}
			p.printTrace(s, lit)
		default:
			p.printTrace(s)
		}
	}

	p.pos, p.tok, p.lit = p.scanner.Scan()
}

// Consume a comment and return it and the line on which it ends.
func (p *parser) consumeComment() (comment *ast.LineComment, endLine int) {
	endLine = p.file.Line(p.pos)
	comment = &ast.LineComment{Start: p.pos, Text: p.lit}
	p.next0()
	return
}

// Consume a group of adjacent comments, add it to the parser's comments list,
// and return it together with the line at which the last comment in the group
// ends. A non-comment token or an empty lines terminate a comment group.
func (p *parser) consumeCommentGroup(n int) (comments *ast.CommentGroup, endLine int) {
	var list []*ast.LineComment
	endLine = p.file.Line(p.pos)
	for p.tok == token.LineComment && p.file.Line(p.pos) <= endLine+n {
		var comment *ast.LineComment
		comment, endLine = p.consumeComment()
		list = append(list, comment)
	}

	// Add comment group to the comments list.
	comments = &ast.CommentGroup{List: list}
	p.comments = append(p.comments, comments)

	return
}

// Advance to the next non-comment token. In the process, collect any comment
// groups encountered, and remember the last lead and line comments.
//
// A lead comment is a comment group that starts and ends in a line without any
// other tokens and that is followed by a non-comment token on the line
// immediately after the comment group.
//
// A line comment is a comment group that follows a non-comment token on the
// same line, and that has no tokens after it on the line where it ends.
//
// Lead comments may be considered documentation that is stored in the AST.
func (p *parser) next() {
	p.leadComment = nil
	prev := p.pos
	p.next0()

	if p.tok == token.LineComment {
		var comment *ast.CommentGroup
		var endLine int

		if p.file.Line(p.pos) == p.file.Line(prev) {
			// The comment is on same line as the previous token; it/ cannot be a
			// lead comment but may be a line comment.
			comment, endLine = p.consumeCommentGroup(0)
		}

		// consume successor comments, if any
		for p.tok == token.LineComment {
			comment, endLine = p.consumeCommentGroup(1)
		}

		if endLine+1 == p.file.Line(p.pos) {
			// The next token is following on the line immediately after the
			// comment group, thus the last comment group is a lead comment.
			p.leadComment = comment
		}
	}
}

// A bailout panic is raised to indicate early termination.
type bailout struct{}

func (p *parser) error(pos gotok.Pos, msg string) {
	epos := p.file.Position(pos)

	// Discard errors reported on the same line as the last recorded error and
	// stop parsing if there are more than 10 errors.
	n := len(p.errors)
	if n > 0 && p.errors[n-1].Pos.Line == epos.Line {
		return // discard - likely a spurious error
	}
	if n > 10 {
		panic(bailout{})
	}

	p.errors.Add(epos, msg)
}

func (p *parser) expect(tok token.Token) gotok.Pos {
	pos := p.pos
	if p.tok != tok {
		p.errorExpected(pos, "'"+tok.String()+"'")
	}
	p.next() // make progress
	return pos
}

func (p *parser) errorExpected(pos gotok.Pos, msg string) {
	msg = "expected " + msg
	if pos == p.pos {
		// The error happened at the current position; make the error message more
		// specific.
		msg += ", found '" + p.tok.String() + "'"
	}
	p.error(pos, msg)
}

// Regexp to extract query annotations that control output.
var annotationRegexp = regexp.MustCompile(`name: ([a-zA-Z0-9_$]+)[ \t]+(:many|:one|:exec)[ \t]*(.*)`)
var annotationArgRegexp = regexp.MustCompile(`arg: ([a-zA-Z_][[a-zA-Z0-9_]*)[ \t]+(.*[a-zA-Z_][[a-zA-Z0-9_$]*)[ \t]*(.*)`)

func (p *parser) parseQuery() ast.Query {
	if p.trace {
		defer un(trace(p, "Query"))
	}

	doc := p.leadComment
	sql := &strings.Builder{}
	pos := p.pos

	names := make([]argPos, 0, 4) // all pggen.arg names in order, can be duplicated
	for p.tok != token.Semicolon {
		if p.tok == token.EOF || p.tok == token.Illegal {
			p.error(p.pos, "unterminated query (no semicolon): "+string(p.src[pos:p.pos]))
			return &ast.BadQuery{From: pos, To: p.pos}
		}
		hasPggenArg := strings.HasSuffix(p.lit, "pggen.arg(") ||
			strings.HasSuffix(p.lit, "pggen.arg (")
		if p.tok == token.QueryFragment && hasPggenArg {
			arg, ok := p.parsePggenArg()
			if !ok {
				return &ast.BadQuery{From: pos, To: p.pos}
			}
			arg.lo -= int(pos) - 1 // adjust lo,hi to be relative to query start
			arg.hi -= int(pos) - 1 // subtract 1 because pos is 1-based

			names = append(names, arg)
			// Don't consume last query fragment that has closing paren ")" because
			// the fragment might contain the start of another pggen.arg.
			continue
		}
		p.next()
	}

	semi := p.pos
	p.expect(token.Semicolon)
	sql.Write(p.src[pos-1 : semi])

	// Extract annotations
	if doc == nil || doc.List == nil || len(doc.List) == 0 {
		p.error(pos, "no comment preceding query")
		return &ast.BadQuery{From: pos, To: p.pos}
	}

	paramTypeOverrides := make(map[string]string, 4)
	annotationPos := len(doc.List) - 1

	for ; annotationPos > 0; annotationPos-- {
		argAnnotations := annotationArgRegexp.FindStringSubmatch(doc.List[annotationPos].Text)
		if argAnnotations == nil {
			break
		}

		argName, argGoType := argAnnotations[1], argAnnotations[2]

		if _, present := paramTypeOverrides[argName]; present {
			p.error(pos, "duplicate arg type override for "+argName+": "+argAnnotations[0])
			return &ast.BadQuery{From: pos, To: p.pos}
		}

		unknownArg := true
		for _, arg := range names {
			if argName == arg.name {
				unknownArg = false
				break
			}
		}
		if unknownArg {
			p.error(pos, "arg type override for unknown arg "+argName+": "+argAnnotations[0])
			return &ast.BadQuery{From: pos, To: p.pos}
		}

		paramTypeOverrides[argName] = argGoType

	}

	last := doc.List[annotationPos]
	annotations := annotationRegexp.FindStringSubmatch(last.Text)
	if annotations == nil {
		p.error(pos, "no 'name: <name> :<type>' token found in comment before query; comment line: \""+last.Text+`"`)
		return &ast.BadQuery{From: pos, To: p.pos}
	}
	args := annotations[3]
	pragmas, err := parsePragmas(args)
	if err != nil {
		p.error(pos, "invalid query pragma: "+err.Error())
		return &ast.BadQuery{From: pos, To: p.pos}
	}

	templateSQL := sql.String()
	preparedSQL, params := prepareSQL(templateSQL, names)

	return &ast.SourceQuery{
		Name:               annotations[1],
		Doc:                doc,
		Start:              pos,
		SourceSQL:          templateSQL,
		PreparedSQL:        preparedSQL,
		ParamNames:         params,
		ParamTypeOverrides: paramTypeOverrides,
		ResultKind:         ast.ResultKind(annotations[2]),
		Pragmas:            pragmas,
		Semi:               semi,
	}
}

// parsePragmas parses optional pragmas for a query like proto-type=foo.bar.Msg.
func parsePragmas(allPragmas string) (ast.Pragmas, error) {
	if allPragmas == "" {
		return ast.Pragmas{}, nil
	}
	ss := strings.Fields(allPragmas)
	qp := ast.Pragmas{}
	for _, s := range ss {
		arg := strings.Split(s, "=")
		if len(arg) != 2 {
			return ast.Pragmas{}, fmt.Errorf("expected arg format x=y; got %s", s)
		}
		key, val := arg[0], arg[1]
		switch key {
		case "proto-type":
			p, err := validateProtoMsgType(val)
			if err != nil {
				return ast.Pragmas{}, err
			}
			qp.ProtobufType = p
		default:
			return ast.Pragmas{}, fmt.Errorf("unsupported pramga %q", key)
		}
	}
	return qp, nil
}

// validateProtoMsgType checks that val is a valid message name.
// https://developers.google.com/protocol-buffers/docs/reference/proto3-spec#identifiers
func validateProtoMsgType(val string) (string, error) {
	isStart := true
	for i, v := range val {
		isLast := i == len(val)-1
		switch {
		case ('a' <= v && v <= 'z') || ('A' <= v && v <= 'Z'):
			isStart = false
		case ('0' <= v && v <= '9') || v == '_':
			if isStart {
				return "", fmt.Errorf("invalid proto-type, proto package cannot start with 0-9 or _; got %q", val)
			}
		case v == '.':
			if isStart || isLast {
				return "", fmt.Errorf("invalid proto-type, proto package cannot start or end with '.'; got %q", val)
			}
			isStart = true
		default:
			return "", fmt.Errorf("invalid proto-type, proto message must only contain [a-zA-Z0-9.-]; got %q", val)
		}
	}
	return val, nil
}

// argPos is the name and position of expression like pggen.arg('foo').
type argPos struct {
	lo, hi int
	name   string
}

// parsePggenArg parses the name from: pggen.arg('foo') and pos for the start
// and end.
func (p *parser) parsePggenArg() (argPos, bool) {
	lo := int(p.pos) + strings.LastIndex(p.lit, "pggen") - 1
	p.next() // consume query fragment that contains "pggen.arg("
	if p.tok != token.String {
		p.error(p.pos, `expected string literal after "pggen.arg("`)
		return argPos{}, false
	}
	if len(p.lit) < 3 || p.lit[0] != '\'' || p.lit[len(p.lit)-1] != '\'' {
		p.error(p.pos, `expected single-quoted string literal after "pggen.arg("`)
		return argPos{}, false
	}
	name := p.lit[1 : len(p.lit)-1]
	p.next() // consume string literal
	if p.tok != token.QueryFragment {
		p.error(p.pos, `expected query fragment after parsing pggen.arg string`)
		return argPos{}, false
	}
	if !strings.HasPrefix(p.lit, ")") {
		p.error(p.pos, `expected closing paren ")" after parsing pggen.arg string`)
		return argPos{}, false
	}
	hi := int(p.pos)
	return argPos{lo: lo, hi: hi, name: name}, true
}

// prepareSQL replaces each pggen.arg with the $n, respecting the order that the
// arg first appeared. Args with the same name use the same $n.
func prepareSQL(sql string, args []argPos) (string, []string) {
	if len(args) == 0 {
		return sql, nil
	}
	// Figure out order of each params.
	paramOrders := make(map[string]int, len(args))
	params := make([]string, 0, len(args))
	idx := 1
	for _, arg := range args {
		if _, ok := paramOrders[arg.name]; !ok {
			params = append(params, arg.name)
			paramOrders[arg.name] = idx
			idx++
		}
	}

	// Replace each pggen.arg with the prepare order, like $1. We're not using
	// strings.NewReplacer because pggen.arg might appear in a comment.
	bs := []byte(sql)
	sb := &strings.Builder{}
	sb.Grow(len(sql))
	prev := 0
	for _, arg := range args {
		sb.Write(bs[prev:arg.lo])
		sb.WriteByte('$')
		sb.WriteString(strconv.Itoa(paramOrders[arg.name]))
		prev = arg.hi
	}
	sb.Write(bs[prev:])

	return sb.String(), params
}

// ----------------------------------------------------------------------------
// Source files

func (p *parser) parseFile() *ast.File {
	if p.trace {
		defer un(trace(p, "File"))
	}

	// Don't bother parsing the rest if we had errors scanning the first token.
	// Likely not a query source file at all.
	if p.errors.Len() != 0 {
		return nil
	}

	// Opening comment
	doc := p.leadComment

	var queries []ast.Query
	for p.tok != token.EOF && p.tok != token.Illegal {
		queries = append(queries, p.parseQuery())
	}

	return &ast.File{
		Doc:      doc,
		Queries:  queries,
		Comments: p.comments,
	}
}
