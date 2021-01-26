package flags

import (
	"flag"
	"strings"
)

// Strings returns a repeated string flag that accumulates value into a slice.
func Strings(fset *flag.FlagSet, name string, value []string, usage string) *[]string {
	sv := &stringsValue{
		strings: &value,
	}
	fset.Var(sv, name, usage)
	return sv.strings
}

type stringsValue struct {
	strings *[]string
}

// String implements flag.Value and fmt.Stringer.
func (sv *stringsValue) String() string {
	return strings.Join(*sv.strings, ",")
}

// Get implements flag.Getter.
func (sv *stringsValue) Get() interface{} {
	return *sv.strings
}

// Set implements flag.Value.
func (sv *stringsValue) Set(value string) error {
	*sv.strings = append(*sv.strings, value)
	return nil
}
