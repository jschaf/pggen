package paths

import (
	"fmt"
	"os"
	"path/filepath"
)

// WalkUp traverses up directory tree from dir until it finds an ancestor file
// named name. Checks the current directory first and then iteratively checks
// parent directories.
func WalkUp(dir, name string) (string, error) {
	for dir != string(os.PathSeparator) {
		p := filepath.Join(dir, name)
		if _, err := os.Stat(p); err != nil {
			if !os.IsNotExist(err) {
				return "", fmt.Errorf("stat file %s: %w", p, err)
			}
		} else {
			return dir, nil
		}
		dir = filepath.Dir(dir)
	}
	return "", fmt.Errorf("dir not found in directory tree starting from %s", dir)
}
