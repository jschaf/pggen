package out

import (
	"github.com/jschaf/pggen"
	"github.com/jschaf/pggen/internal/pgtest"
	"github.com/stretchr/testify/assert"
	"os"
	"path/filepath"
	"testing"
)

func TestGenerate_Go_Example_SeparateOutDir(t *testing.T) {
	conn, cleanupFunc := pgtest.NewPostgresSchema(t, []string{
		"../schema.sql",
	})
	defer cleanupFunc()

	tmpDir := t.TempDir()
	err := pggen.Generate(
		pggen.GenerateOptions{
			ConnString: conn.Config().ConnString(),
			QueryFiles: []string{
				"../alpha/query.sql",
				"../alpha/alpha/query.sql",
				"../bravo/query.sql",
			},
			OutputDir: tmpDir,
			GoPackage: "out",
			Language:  pggen.LangGo,
		})
	if err != nil {
		t.Fatalf("Generate(): %s", err)
	}

	for _, file := range []string{
		"alpha_query.sql.0.go",
		"alpha_query.sql.1.go",
		"bravo_query.sql.go",
	} {
		wantQueries, err := os.ReadFile(file)
		if err != nil {
			t.Fatalf("read wanted file %s: %s", file, err)
		}

		gotFile := filepath.Join(tmpDir, file)
		assert.FileExists(t, gotFile, "Generate() should emit "+file)
		gotQueries, err := os.ReadFile(gotFile)
		if err != nil {
			t.Fatalf("read generated %s: %s", file, err)
		}
		assert.Equalf(t, string(wantQueries), string(gotQueries),
			"Got file %s; does not match contents of file %s",
			gotFile, file)
	}
}
