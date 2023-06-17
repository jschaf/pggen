package author

import (
	"github.com/jschaf/pggen"
	"github.com/jschaf/pggen/internal/pgtest"
	"github.com/stretchr/testify/assert"
	"os"
	"path/filepath"
	"testing"
)

func TestGenerate_Go_Example_InlineParamCount(t *testing.T) {
	conn, cleanupFunc := pgtest.NewPostgresSchema(t, []string{"schema.sql"})
	defer cleanupFunc()

	tests := []struct {
		name          string
		opts          pggen.GenerateOptions
		wantQueryPath string
	}{
		{
			name: "inline0",
			opts: pggen.GenerateOptions{
				ConnString:       conn.Config().ConnString(),
				QueryFiles:       []string{"query.sql"},
				GoPackage:        "inline0",
				Language:         pggen.LangGo,
				InlineParamCount: 0,
			},
			wantQueryPath: "inline0/query.sql.go",
		},
		{
			name: "inline1",
			opts: pggen.GenerateOptions{
				ConnString:       conn.Config().ConnString(),
				QueryFiles:       []string{"query.sql"},
				GoPackage:        "inline1",
				Language:         pggen.LangGo,
				InlineParamCount: 1,
			},
			wantQueryPath: "inline1/query.sql.go",
		},
		{
			name: "inline2",
			opts: pggen.GenerateOptions{
				ConnString:       conn.Config().ConnString(),
				QueryFiles:       []string{"query.sql"},
				GoPackage:        "inline2",
				Language:         pggen.LangGo,
				InlineParamCount: 2,
			},
			wantQueryPath: "inline2/query.sql.go",
		},
		{
			name: "inline3",
			opts: pggen.GenerateOptions{
				ConnString:       conn.Config().ConnString(),
				QueryFiles:       []string{"query.sql"},
				GoPackage:        "inline3",
				Language:         pggen.LangGo,
				InlineParamCount: 3,
			},
			wantQueryPath: "inline3/query.sql.go",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpDir := t.TempDir()
			tt.opts.OutputDir = tmpDir
			err := pggen.Generate(tt.opts)
			if err != nil {
				t.Fatalf("Generate() example/author %s: %s", tt.name, err.Error())
			}

			gotQueryFile := filepath.Join(tmpDir, "query.sql.go")
			assert.FileExists(t, gotQueryFile, "Generate() should emit query.sql.go")
			wantQueries, err := os.ReadFile(tt.wantQueryPath)
			if err != nil {
				t.Fatalf("read wanted query.go.sql: %s", err)
			}
			gotQueries, err := os.ReadFile(gotQueryFile)
			if err != nil {
				t.Fatalf("read generated query.go.sql: %s", err)
			}
			assert.Equalf(t, string(wantQueries), string(gotQueries),
				"Got file %s; does not match contents of %s",
				gotQueryFile, tt.wantQueryPath)
		})
	}
}
