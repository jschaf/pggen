package difftest

import (
	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	"testing"
)

func AssertSame(t *testing.T, want, got interface{}, opts ...cmp.Option) {
	t.Helper()
	allOpts := append([]cmp.Option{
		cmpopts.EquateEmpty(), // useful so nil is same as 0-sized slice
	}, opts...)
	if diff := cmp.Diff(want, got, allOpts...); diff != "" {
		t.Errorf("mismatch (-want +got)\n%s", diff)
	}
}
