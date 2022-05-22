package gotype

import (
	"testing"
)

func TestHasCompositeType(t *testing.T) {
	tests := []struct {
		name string
		typ  Type
		want bool
	}{
		{"enum", &EnumType{}, false},
		{"void", &VoidType{}, false},
		{"opaque", &OpaqueType{}, false},
		{"empty array", &ArrayType{}, false},
		{"array with composite", &ArrayType{Elem: &CompositeType{}}, true},
		{"composite", &CompositeType{}, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := HasCompositeType(tt.typ); got != tt.want {
				t.Errorf("HasCompositeType() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestHasArrayType(t *testing.T) {
	tests := []struct {
		name string
		typ  Type
		want bool
	}{
		{"enum", &EnumType{}, false},
		{"void", &VoidType{}, false},
		{"opaque", &OpaqueType{}, false},
		{"empty array", &ArrayType{}, true},
		{"array with composite", &ArrayType{Elem: &CompositeType{}}, true},
		{"empty composite", &CompositeType{}, false},
		{"composite with array", &CompositeType{FieldTypes: []Type{&ArrayType{}}}, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := HasArrayType(tt.typ); got != tt.want {
				t.Errorf("HasArrayType() = %v, want %v", got, tt.want)
			}
		})
	}
}
