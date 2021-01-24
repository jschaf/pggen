package codegen

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

// ExpandQueryParams expands the templateQuery.Inputs based on the number of
// params.
func (tq templateQuery) ExpandQueryParams() string {
	switch len(tq.Inputs) {
	case 0:
		return ""
	case 1, 2:
		sb := strings.Builder{}
		for _, input := range tq.Inputs {
			sb.WriteString(", ")
			sb.WriteString(lowercaseFirstLetter(input.Name))
			sb.WriteRune(' ')
			sb.WriteString(input.GoType)
		}
		return sb.String()
	default:
		return ", params " + tq.Name + "Params"
	}
}

// ExpandQueryResult returns the string representing the query result type.
func (tq templateQuery) ExpandQueryResult() (string, error) {
	switch tq.ResultKind {
	case ast.ResultKindExec:
		return "pgconn.CommandTag", nil
	case ast.ResultKindMany:
		switch len(tq.Outputs) {
		case 0:
			return "pgconn.CommandTag", nil
		case 1:
			return "[]" + tq.Outputs[0].GoType, nil
		default:
			return "[]" + tq.Name + "Row", nil
		}
	case ast.ResultKindOne:
		switch len(tq.Outputs) {
		case 0:
			return "pgconn.CommandTag", nil
		case 1:
			return tq.Outputs[0].GoType, nil
		default:
			return tq.Name + "Row", nil
		}
	default:
		return "", fmt.Errorf("unhandled result type: %s", tq.ResultKind)
	}
}

func getLongestField(outs []goOutputColumn) int {
	max := 0
	for _, out := range outs {
		if len(out.GoName) > max {
			max = len(out.GoName)
		}
	}
	return max
}

func (tq templateQuery) ExpandQueryParamsStruct() string {
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
		typeCol := getLongestField(tq.Outputs) + 1 // 1 space
		for _, out := range tq.Outputs {
			sb.WriteString("\t")
			sb.WriteString(out.GoName)
			sb.WriteString(strings.Repeat(" ", typeCol-len(out.GoName)))
			sb.WriteString(out.GoType)
			sb.WriteRune('\n')
		}
		sb.WriteString("}\n")
		return sb.String()
	default:
		panic("unhandled result type: " + tq.ResultKind)
	}
}

// emitAll emits all query files.
func emitAll(outDir string, queries []queryFile) error {
	tmpl, err := parseQueryTemplate()
	if err != nil {
		return err
	}
	for _, query := range queries {
		if err := emitQueryFile(outDir, query, tmpl); err != nil {
			return err
		}
	}
	return nil
}

func parseQueryTemplate() (*template.Template, error) {
	statikFS, err := fs.New()
	if err != nil {
		return nil, fmt.Errorf("create statik filesystem: %w", err)
	}
	tmplFile, err := statikFS.Open("/query.gotemplate")
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
func emitQueryFile(outDir string, queryFile queryFile, tmpl *template.Template) (mErr error) {
	base := filepath.Base(queryFile.Src)
	out := filepath.Join(outDir, base+".go")
	file, err := os.OpenFile(out, os.O_CREATE|os.O_WRONLY, 0644)
	defer errs.Capture(&mErr, file.Close, "close emit query file")
	if err != nil {
		return fmt.Errorf("open generated query file for writing: %w", err)
	}
	if err := tmpl.ExecuteTemplate(file, "gen_query", queryFile); err != nil {
		return fmt.Errorf("execute generated query file template %s: %w", out, err)
	}
	return nil
}
