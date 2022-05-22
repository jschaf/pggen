package golang

import (
	"github.com/jschaf/pggen/internal/codegen/golang/gotype"
	"sort"
)

// ImportSet contains a set of imports required by one Go file.
type ImportSet struct {
	imports map[string]struct{}
}

func NewImportSet() *ImportSet {
	return &ImportSet{imports: make(map[string]struct{}, 4)}
}

// AddPackage adds a fully qualified package path to the set, like
// "github.com/jschaf/pggen/foo".
func (s *ImportSet) AddPackage(p string) {
	s.imports[p] = struct{}{}
}

// AddType adds all fully qualified package paths needed for type and any child
// types.
func (s *ImportSet) AddType(typ gotype.Type) {
	s.AddPackage(typ.Import())
	comp, ok := typ.(*gotype.CompositeType)
	if !ok {
		return
	}
	for _, childType := range comp.FieldTypes {
		s.AddType(childType)
	}
}

// SortedPackages returns a new slice containing the sorted packages, suitable
// for an import statement.
func (s *ImportSet) SortedPackages() []string {
	imps := make([]string, 0, len(s.imports))
	for pkg := range s.imports {
		if pkg != "" {
			imps = append(imps, pkg)
		}
	}
	sort.Strings(imps)
	return imps
}
