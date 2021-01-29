package syntax

import (
	"github.com/jschaf/pggen/codegen"
	"github.com/jschaf/pggen/codegen/gen"
	"github.com/jschaf/pggen/internal/pgtest"
	"github.com/stretchr/testify/assert"
	"io/ioutil"
	"path/filepath"
	"testing"
)

func TestGenerate_Go_Example_Syntax(t *testing.T) {
	conn, cleanupFunc := pgtest.NewPostgresSchema(t, nil)
	defer cleanupFunc()

	tmpDir := t.TempDir()
	err := codegen.Generate(
		gen.GenerateOptions{
			ConnString: conn.Config().ConnString(),
			QueryFiles: []string{
				"query.sql",
			},
			OutputDir: tmpDir,
			GoPackage: "syntax",
			Language:  gen.LangGo,
		})
	if err != nil {
		t.Fatalf("Generate() example/syntax: %s", err)
	}

	wantQueriesFile := "query.sql.go"
	gotQueriesFile := filepath.Join(tmpDir, "query.sql.go")
	assert.FileExists(t, gotQueriesFile, "Generate() should emit query.sql.go")
	wantQueries, err := ioutil.ReadFile(wantQueriesFile)
	if err != nil {
		t.Fatalf("read wanted query.go.sql: %s", err)
	}
	gotQueries, err := ioutil.ReadFile(gotQueriesFile)
	if err != nil {
		t.Fatalf("read generated query.go.sql: %s", err)
	}
	assert.Equalf(t, string(wantQueries), string(gotQueries),
		"Got file %s; does not match contents of %s",
		gotQueriesFile, wantQueriesFile)
}
