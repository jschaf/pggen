package codegen

import (
	"github.com/jschaf/sqld/pgtest"
	"github.com/stretchr/testify/assert"
	"path/filepath"
	"testing"
)

func TestGenerate_Example_Author(t *testing.T) {
	conn, cleanupFunc := pgtest.NewPostgresSchema(t, []string{
		"../example/author/schema.sql",
	})
	defer cleanupFunc()

	tmpDir := t.TempDir()
	err := Generate(
		GenerateOptions{
			ConnString: conn.Config().ConnString(),
			QueryFiles: []string{
				"../example/author/queries.sql",
			},
			Config:    Config{},
			OutputDir: tmpDir,
		})
	if err != nil {
		t.Fatalf("Generate() example/author: %s", err)
	}

	assert.FileExists(t, filepath.Join(tmpDir, "queries.sql.go"),
		"Generate() should emit queries.sql.go")
}
