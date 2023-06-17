package custom_types

import (
	"github.com/jschaf/pggen"
	"github.com/jschaf/pggen/internal/pgtest"
	"github.com/stretchr/testify/assert"
	"os"
	"path/filepath"
	"testing"
)

func TestGenerate_Go_Example_CustomTypes(t *testing.T) {
	conn, cleanupFunc := pgtest.NewPostgresSchema(t, []string{"schema.sql"})
	defer cleanupFunc()

	tmpDir := t.TempDir()
	err := pggen.Generate(
		pggen.GenerateOptions{
			ConnString:       conn.Config().ConnString(),
			QueryFiles:       []string{"query.sql"},
			OutputDir:        tmpDir,
			GoPackage:        "custom_types",
			Language:         pggen.LangGo,
			InlineParamCount: 2,
			TypeOverrides: map[string]string{
				"text":    "github.com/jschaf/pggen/example/custom_types/mytype.String",
				"int8":    "github.com/jschaf/pggen/example/custom_types.CustomInt",
				"my_int":  "int",
				"_my_int": "[]int",
			},
		})
	if err != nil {
		t.Fatalf("Generate() example/custom_types: %s", err)
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
