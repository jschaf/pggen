package golang

import (
	"fmt"
	"github.com/jschaf/pggen/internal/ast"
	"github.com/jschaf/pggen/internal/errs"
	"github.com/rakyll/statik/fs"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"text/template"
)

var templateFuncs = template.FuncMap{
	"lowercaseFirstLetter": lowercaseFirstLetter,
	"trimTrailingNewline":  func(s string) string { return strings.TrimSuffix(s, "\n") },
}

// isLast returns true if index is the last index in item.
func lowercaseFirstLetter(s string) string {
	if s == "" {
		return ""
	}
	first, rest := s[0], s[1:]
	return strings.ToLower(string(first)) + rest
}

// EmitParams emits the goTemplateQuery.Inputs into method parameters with both
// a name and type based on the number of params. For use in a method
// definition.
func (tq goTemplateQuery) EmitParams() string {
	switch len(tq.Inputs) {
	case 0:
		return ""
	case 1, 2:
		sb := strings.Builder{}
		for _, input := range tq.Inputs {
			sb.WriteString(", ")
			sb.WriteString(lowercaseFirstLetter(input.Name))
			sb.WriteRune(' ')
			sb.WriteString(input.Type)
		}
		return sb.String()
	default:
		return ", params " + tq.Name + "Params"
	}
}

// EmitPreparedSQL emits the prepared SQL query with appropriate quoting.
func (tq goTemplateQuery) EmitPreparedSQL() string {
	hasBacktick := strings.ContainsRune(tq.PreparedSQL, '`')
	if !hasBacktick {
		return "`" + tq.PreparedSQL + "`"
	}
	hasDoubleQuote := strings.ContainsRune(tq.PreparedSQL, '"')
	hasNewline := strings.ContainsAny(tq.PreparedSQL, "\r\n")
	if !hasDoubleQuote && !hasNewline {
		hasBackslash := strings.ContainsRune(tq.PreparedSQL, '\\')
		sql := tq.PreparedSQL
		if hasBackslash {
			sql = strings.ReplaceAll(sql, `\`, `\\`)
		}
		return `"` + sql + `"`
	}
	// The SQL query contains both '`' and '"'.
	// We can't use unicode escapes like U&'d\0061t\+000061' because the backtick
	// can appear in either a double-quoted identifier like "abc`" or a string
	// literal. Similarly, a double quote either delimits an identifier or can
	// appear in a string literal. We'll break up the string using Go string
	// concatenation using both types of Go string literals. Meaning, convert:
	//     sql := `SELECT '`"'`
	// Into:
	//     sql := `SELECT '` + "`" + `"'`
	return "`" + strings.ReplaceAll(tq.PreparedSQL, "`", "` + \"`\" + `") + "`"
}

func getLongestInput(inputs []goInputParam) int {
	max := 0
	for _, in := range inputs {
		if len(in.Name) > max {
			max = len(in.Name)
		}
	}
	return max
}

// EmitParamStruct emits the struct definition for query params if needed.
func (tq goTemplateQuery) EmitParamStruct() string {
	if len(tq.Inputs) < 3 {
		return ""
	}
	sb := &strings.Builder{}
	sb.WriteString("\n\ntype ")
	sb.WriteString(tq.Name)
	sb.WriteString("Params struct {\n")
	typeCol := getLongestInput(tq.Inputs) + 1 // 1 space
	for _, out := range tq.Inputs {
		sb.WriteString("\t")
		sb.WriteString(out.Name)
		sb.WriteString(strings.Repeat(" ", typeCol-len(out.Name)))
		sb.WriteString(out.Type)
		sb.WriteRune('\n')
	}
	sb.WriteString("}")
	return sb.String()
}

// EmitParamNames emits the goTemplateQuery.Inputs into comma separated names
// for use in a method invocation.
func (tq goTemplateQuery) EmitParamNames() string {
	switch len(tq.Inputs) {
	case 0:
		return ""
	case 1, 2:
		sb := strings.Builder{}
		for _, input := range tq.Inputs {
			sb.WriteString(", ")
			sb.WriteString(lowercaseFirstLetter(input.Name))
		}
		return sb.String()
	default:
		sb := strings.Builder{}
		for _, input := range tq.Inputs {
			sb.WriteString(", params.")
			sb.WriteString(input.Name)
		}
		return sb.String()
	}
}

// EmitRowScanArgs emits the args to scan a single row from a pgx.Row or
// pgx.Rows.
func (tq goTemplateQuery) EmitRowScanArgs() (string, error) {
	switch tq.ResultKind {
	case ast.ResultKindExec:
		return "", fmt.Errorf("cannot EmitRowScanArgs for :exec query %s", tq.Name)
	case ast.ResultKindMany, ast.ResultKindOne:
		switch len(tq.Outputs) {
		case 0:
			return "", nil
		case 1:
			// If there's only 1 output column, we return it directly, without
			// wrapping in a struct.
			return "&item", nil
		default:
			sb := strings.Builder{}
			sb.Grow(15 * len(tq.Outputs))
			for i, out := range tq.Outputs {
				sb.WriteString("&item.")
				sb.WriteString(out.Name)
				if i < len(tq.Outputs)-1 {
					sb.WriteString(", ")
				}
			}
			return sb.String(), nil
		}
	default:
		return "", fmt.Errorf("unhandled EmitRowScanArgs type: %s", tq.ResultKind)
	}
}

// EmitResultType returns the string representing the overall query result type,
// meaning the return result.
func (tq goTemplateQuery) EmitResultType() (string, error) {
	switch tq.ResultKind {
	case ast.ResultKindExec:
		return "pgconn.CommandTag", nil
	case ast.ResultKindMany:
		switch len(tq.Outputs) {
		case 0:
			return "pgconn.CommandTag", nil
		case 1:
			return "[]" + tq.Outputs[0].Type, nil
		default:
			return "[]" + tq.Name + "Row", nil
		}
	case ast.ResultKindOne:
		switch len(tq.Outputs) {
		case 0:
			return "pgconn.CommandTag", nil
		case 1:
			return tq.Outputs[0].Type, nil
		default:
			return tq.Name + "Row", nil
		}
	default:
		return "", fmt.Errorf("unhandled EmitResultType type: %s", tq.ResultKind)
	}
}

// EmitResultElem returns the string representing a single item in the overall
// query result type. For :one and :exec queries, this is the same as
// EmitResultType. For :many queries, this is the element type of the slice
// result type.
func (tq goTemplateQuery) EmitResultElem() (string, error) {
	result, err := tq.EmitResultType()
	if err != nil {
		return "", fmt.Errorf("unhandled EmitResultElem type: %w", err)
	}
	return strings.TrimPrefix(result, "[]"), nil
}

func getLongestOutput(outs []goOutputColumn) int {
	max := 0
	for _, out := range outs {
		if len(out.Name) > max {
			max = len(out.Name)
		}
	}
	return max
}

// EmitRowStruct writes the struct definition for query output row if one is
// needed.
func (tq goTemplateQuery) EmitRowStruct() string {
	switch tq.ResultKind {
	case ast.ResultKindExec:
		return ""
	case ast.ResultKindOne, ast.ResultKindMany:
		if len(tq.Outputs) <= 1 {
			return ""
		}
		sb := &strings.Builder{}
		sb.WriteString("\n\ntype ")
		sb.WriteString(tq.Name)
		sb.WriteString("Row struct {\n")
		typeCol := getLongestOutput(tq.Outputs) + 1 // 1 space
		for _, out := range tq.Outputs {
			sb.WriteString("\t")
			sb.WriteString(out.Name)
			sb.WriteString(strings.Repeat(" ", typeCol-len(out.Name)))
			sb.WriteString(out.Type)
			sb.WriteRune('\n')
		}
		sb.WriteString("}")
		return sb.String()
	default:
		panic("unhandled result type: " + tq.ResultKind)
	}
}

func parseQueryTemplate() (*template.Template, error) {
	statikFS, err := fs.New()
	if err != nil {
		return nil, fmt.Errorf("create statik filesystem: %w", err)
	}
	tmplFile, err := statikFS.Open("/golang/query.gotemplate")
	if err != nil {
		return nil, fmt.Errorf("open embedded template file: %w", err)
	}
	tmplBytes, err := ioutil.ReadAll(tmplFile)
	if err != nil {
		return nil, fmt.Errorf("read embedded template file: %w", err)
	}

	tmpl, err := template.New("gen_query").Funcs(templateFuncs).Parse(string(tmplBytes))
	if err != nil {
		return nil, fmt.Errorf("parse query.gotemplate: %w", err)
	}
	return tmpl, nil
}

// emitQueryFile emits a single query file.
func emitQueryFile(outDir string, queryFile goQueryFile, tmpl *template.Template) (mErr error) {
	base := filepath.Base(queryFile.BaseName)
	out := filepath.Join(outDir, base+".go")
	file, err := os.OpenFile(out, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0644)
	defer errs.Capture(&mErr, file.Close, "close emit query file")
	if err != nil {
		return fmt.Errorf("open generated query file for writing: %w", err)
	}
	if err := tmpl.ExecuteTemplate(file, "gen_query", queryFile); err != nil {
		return fmt.Errorf("execute generated query file template %s: %w", out, err)
	}
	return nil
}
