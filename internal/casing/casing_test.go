package casing

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestCaser_ToUpperCamel(t *testing.T) {
	tests := []struct {
		word     string
		want     string
		acronyms map[string]string
	}{
		{"fooBar", "FooBar", nil},
		{"foo", "Foo", nil},
		{"foo123", "Foo123", nil},
		{"foo123bar", "Foo123bar", nil},
		{"foo123bar_baz", "Foo123barBaz", nil},
		{"foo", "FOO", map[string]string{"foo": "FOO"}},
		{"foo_", "Foo", nil},
		{"_foo_", "Foo", nil},
		{"foo__", "Foo", nil},
		{"foo__bar", "FooBar", nil},
		{"foo_bar", "FooBar", nil},
		{"foo_bar_baz", "FooBarBaz", nil},
		{"foo_bar_baz", "FooBarBAZ", map[string]string{"baz": "BAZ"}},
		{"foo_bar_baz", "FooBARBAZ", map[string]string{"bar": "BAR", "baz": "BAZ"}},
	}
	for _, tt := range tests {
		t.Run(tt.word+"="+tt.want, func(t *testing.T) {
			caser := NewCaser()
			caser.AddAcronyms(tt.acronyms)
			got := caser.ToUpperCamel(tt.word)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestCaser_ToUpperGoIdent(t *testing.T) {
	tests := []struct {
		word     string
		want     string
		acronyms map[string]string
	}{
		{"fooBar", "FooBar", nil},
		{"$", "", nil},
		{"$foo$bar", "FooBar", nil},
		{"foo bar@@@!", "FooBar", nil},
		{"12!foo bar@@@!", "FooBar", nil},
		{"foo", "Foo", nil},
		{"foo123", "Foo123", nil},
		{"foo123bar", "Foo123bar", nil},
		{"foo123bar_baz", "Foo123barBaz", nil},
		{"foo", "FOO", map[string]string{"foo": "FOO"}},
		{"foo_", "Foo", nil},
		{"_foo_", "Foo", nil},
		{"foo__", "Foo", nil},
		{"foo__bar", "FooBar", nil},
		{"foo_bar", "FooBar", nil},
		{"foo_bar_baz", "FooBarBaz", nil},
		{"foo_bar_baz", "FooBarBAZ", map[string]string{"baz": "BAZ"}},
		{"foo_bar_baz", "FooBARBAZ", map[string]string{"bar": "BAR", "baz": "BAZ"}},
	}
	for _, tt := range tests {
		t.Run(tt.word+"="+tt.want, func(t *testing.T) {
			caser := NewCaser()
			caser.AddAcronyms(tt.acronyms)
			got := caser.ToUpperGoIdent(tt.word)
			assert.Equal(t, tt.want, got)
		})
	}
}
