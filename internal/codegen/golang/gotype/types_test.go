package gotype

import (
	"github.com/google/go-cmp/cmp"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestMustParseKnownType(t *testing.T) {
	tests := []struct {
		qualType string
		want     Type
	}{
		{
			qualType: "string",
			want:     &OpaqueType{Name: "string"},
		},
		{
			qualType: "*string",
			want:     &PointerType{Elem: &OpaqueType{Name: "string"}},
		},
		{
			qualType: "[]string",
			want:     &ArrayType{Elem: &OpaqueType{Name: "string"}},
		},
		{
			qualType: "[]*string",
			want:     &ArrayType{Elem: &PointerType{Elem: &OpaqueType{Name: "string"}}},
		},
		{
			qualType: "time.Time",
			want: &ImportType{
				PkgPath: "time",
				Type:    &OpaqueType{Name: "Time"},
			},
		},
		{
			qualType: "[]time.Time",
			want: &ArrayType{
				Elem: &ImportType{PkgPath: "time", Type: &OpaqueType{Name: "Time"}},
			},
		},
		{
			qualType: "[]*time.Time",
			want: &ArrayType{
				Elem: &PointerType{
					Elem: &ImportType{PkgPath: "time", Type: &OpaqueType{Name: "Time"}},
				},
			},
		},
		{
			qualType: "[]util/custom/times.Interval",
			want: &ArrayType{
				Elem: &ImportType{PkgPath: "util/custom/times", Type: &OpaqueType{Name: "Interval"}},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.qualType, func(t *testing.T) {
			got := MustParseOpaqueType(tt.qualType)
			if diff := cmp.Diff(tt.want, got); diff != "" {
				t.Errorf("mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestQualifyType(t *testing.T) {
	tests := []struct {
		name     string
		typ      Type
		otherPkg string
		want     string
	}{
		{
			name:     "string",
			typ:      &OpaqueType{Name: "string"},
			otherPkg: "example.com/foo",
			want:     "string",
		},
		{
			name:     "[]string",
			typ:      &ArrayType{Elem: &OpaqueType{Name: "string"}},
			otherPkg: "example.com/foo",
			want:     "[]string",
		},
		{
			name:     "[]*string",
			typ:      &ArrayType{Elem: &PointerType{Elem: &OpaqueType{Name: "string"}}},
			otherPkg: "example.com/foo",
			want:     "[]*string",
		},
		{
			name:     "foo.com/qux.Bar - example.com/foo",
			typ:      &ImportType{PkgPath: "foo.com/qux", Type: &OpaqueType{Name: "Bar"}},
			otherPkg: "example.com/foo",
			want:     "qux.Bar",
		},
		{
			name:     "[]foo.com/qux.Bar - example.com/foo",
			typ:      &ArrayType{Elem: &ImportType{PkgPath: "foo.com/qux", Type: &OpaqueType{Name: "Bar"}}},
			otherPkg: "example.com/foo",
			want:     "[]qux.Bar",
		},
		{
			name:     "[]example.com/qux.Bar - example.com/foo",
			typ:      &ArrayType{Elem: &ImportType{PkgPath: "example.com/qux", Type: &OpaqueType{Name: "Bar"}}},
			otherPkg: "example.com/foo",
			want:     "[]qux.Bar",
		},
		{
			name:     "[]example.com/foo.Bar - example.com/foo",
			typ:      &ArrayType{Elem: &ImportType{PkgPath: "example.com/foo", Type: &OpaqueType{Name: "Bar"}}},
			otherPkg: "example.com/foo",
			want:     "[]Bar",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := QualifyType(tt.typ, tt.otherPkg)
			assert.Equal(t, tt.want, got)
		})
	}
}
