package golang

import (
	"fmt"
	"github.com/jschaf/pggen/internal/errs"
	"os"
	"path/filepath"
	"strconv"
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

// EmitAllQueryFiles emits a query file for each TemplatedFile. Ensure that
// emitted files don't clash by prefixing with the parent directory if
// necessary.
func (em Emitter) EmitAllQueryFiles(tfs []TemplatedFile) (mErr error) {
	outs := em.chooseOutputFiles(tfs)
	for i, tf := range tfs {
		if err := em.emitQueryFile(outs[i], tf); err != nil {
			return err
		}
	}
	return nil
}

// chooseOutputFiles returns the output paths to use for each TemplatedFile.
// Necessary for cases like "alpha/query.sql" and "bravo/query.sql" where
// we can't simply use "query.sql.go".
func (em Emitter) chooseOutputFiles(tfs []TemplatedFile) []string {
	// Check for any basename collisions.
	seenBaseNames := make(map[string]struct{}, len(tfs))
	hasBaseCollision := false
	for _, tf := range tfs {
		base := filepath.Base(tf.SourcePath)
		if _, ok := seenBaseNames[base]; ok {
			hasBaseCollision = true
		}
		seenBaseNames[base] = struct{}{}
	}

	// If no base collision, just use base names. If no collisions, use the
	// basename, like "query.sql" => "query.go.sql".
	if !hasBaseCollision {
		outNames := make([]string, len(tfs))
		for i, tf := range tfs {
			out := filepath.Base(tf.SourcePath)
			out += ".go"
			outNames[i] = out
		}
		return outNames
	}

	// If there's a basename collision, check for collisions after prefixing the
	// parent directory name. If there's still a collision we'll make each name
	// unique with a numeric literal suffix. Occurs with a file pattern like:
	// "alpha/query.sql" and "parent/alpha/query.sql".
	outNames := make([]string, len(tfs)) // names to return
	usedNames := make(map[string]int)    // next int to use for a collision on key
	firstIdx := make(map[string]int)     // first index a name was used
	for i, tf := range tfs {
		out := filepath.Base(tf.SourcePath)
		parent := filepath.Base(filepath.Dir(tf.SourcePath))
		out = parent + "_" + out
		n, ok := usedNames[out]
		usedNames[out] = n + 1
		if ok {
			// We've seen this entry already.
			firstI := firstIdx[out]
			outNames[i] = out + "." + strconv.Itoa(n)
			if n == 1 {
				// Add suffix to first entry since we didn't do it the first time around
				// because we didn't know if it had a collision.
				outNames[firstI] += ".0"
			}
		} else {
			// First time seeing an entry.
			outNames[i] = out
			firstIdx[out] = i
		}
	}
	for i := range outNames {
		outNames[i] += ".go"
	}
	return outNames
}

// emitQueryFile emits a single query file.
func (em Emitter) emitQueryFile(outRelPath string, tf TemplatedFile) (mErr error) {
	out := filepath.Join(em.outDir, outRelPath)
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
