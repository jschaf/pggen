package go_pointer_types

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/jschaf/pggen"
	"github.com/jschaf/pggen/internal/pgtest"
	"github.com/stretchr/testify/assert"
)

func TestGenerate_Go_Example_GoPointerTypes(t *testing.T) {
	conn, cleanupFunc := pgtest.NewPostgresSchema(t, []string{"schema.sql"})
	defer cleanupFunc()

	tmpDir := t.TempDir()
	err := pggen.Generate(
		pggen.GenerateOptions{
			ConnString:       conn.Config().ConnString(),
			QueryFiles:       []string{"query.sql"},
			OutputDir:        tmpDir,
			GoPackage:        "go_pointer_types",
			Language:         pggen.LangGo,
			InlineParamCount: 2,
			TypeOverrides: map[string]string{
				"int4":  "*int",
				"_int4": "[]int",
				"int8":  "*int",
				"_int8": "[]int",
				"text":  "*string",
			},
		})
	if err != nil {
		t.Fatalf("Generate() example/go_pointer_types: %s", err)
	}

	wantQueryFile := "query.sql.go"
	gotQueryFile := filepath.Join(tmpDir, "query.sql.go")
	assert.FileExists(t, gotQueryFile, "Generate() should emit query.sql.go")
	wantQueries, err := os.ReadFile(wantQueryFile)
	if err != nil {
		t.Fatalf("read wanted query.go.sql: %s", err)
	}
	gotQueries, err := os.ReadFile(gotQueryFile)
	if err != nil {
		t.Fatalf("read generated query.go.sql: %s", err)
	}
	assert.Equalf(t, string(wantQueries), string(gotQueries),
		"Got file %s; does not match contents of %s",
		gotQueryFile, wantQueryFile)
}
