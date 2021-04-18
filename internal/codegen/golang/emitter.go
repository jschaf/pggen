package golang

import (
	"fmt"
	"github.com/jschaf/pggen/internal/errs"
	"os"
	"path/filepath"
	"text/template"
)

// Emitter writes a templated query file to a file.
type Emitter struct {
	outDir string
	tmpl   *template.Template
}

func NewEmitter(outDir string, tmpl *template.Template) Emitter {
	return Emitter{outDir: outDir, tmpl: tmpl}
}

// EmitQueryFile emits a single query file.
func (em Emitter) EmitQueryFile(tf TemplatedFile) (mErr error) {
	base := filepath.Base(tf.SourcePath)
	out := filepath.Join(em.outDir, base+".go")
	file, err := os.OpenFile(out, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0644)
	defer errs.Capture(&mErr, file.Close, "close emit query file")
	if err != nil {
		return fmt.Errorf("open generated query file for writing: %w", err)
	}
	if err := em.tmpl.ExecuteTemplate(file, "gen_query", tf); err != nil {
		return fmt.Errorf("execute generated query file template %s: %w", out, err)
	}
	return nil
}
