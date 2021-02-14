package casing

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestCaser_ToUpperGoIdent(t *testing.T) {
	tests := []struct {
		word     string
		want     string
		acronyms map[string]string
	}{
		{"fooBar", "FooBar", nil},
		{"FooBar", "FooBar", nil},
		{"$", "", nil},
		{"user.id", "UserID", map[string]string{"id": "ID"}},
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
		{"Ě", "Ě", nil},
		{"ě", "Ě", nil},
		{"Ěě_ě", "ĚěĚ", nil},
		{"OIDs", "OIDs", map[string]string{"oids": "OIDs"}},
		{"OIDsBar", "OIDsBar", map[string]string{"oids": "OIDs"}},
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
func TestCaser_ToLowerGoIdent(t *testing.T) {
	tests := []struct {
		word     string
		want     string
		acronyms map[string]string
	}{
		{"fooBar", "fooBar", nil},
		{"FooBar", "fooBar", nil},
		{"$", "", nil},
		{"user.id", "userID", map[string]string{"id": "ID"}},
		{"$foo$bar", "fooBar", nil},
		{"foo bar@@@!", "fooBar", nil},
		{"12!foo bar@@@!", "fooBar", nil},
		{"foo", "foo", nil},
		{"foo123", "foo123", nil},
		{"foo123bar", "foo123bar", nil},
		{"foo123bar_baz", "foo123barBaz", nil},
		{"foo", "foo", map[string]string{"foo": "FOO"}},
		{"foo_", "foo", nil},
		{"_foo_", "foo", nil},
		{"foo__", "foo", nil},
		{"foo__bar", "fooBar", nil},
		{"foo_bar", "fooBar", nil},
		{"foo_bar_baz", "fooBarBaz", nil},
		{"foo_bar_baz", "fooBarBAZ", map[string]string{"baz": "BAZ"}},
		{"foo_bar_baz", "fooBARBAZ", map[string]string{"bar": "BAR", "baz": "BAZ"}},
		{"Ě", "ě", nil},
		{"ě", "ě", nil},
		{"Ěě_ě", "ěěĚ", nil},
		{"if", "if_", nil},
		{"type", "type_", nil},
		{"OIDs", "oids", nil},
		{"OIDsBar", "oidsBar", nil},
		{"FindOIDByVal", "findOIDByVal", map[string]string{"oid": "OID"}},
	}
	for _, tt := range tests {
		t.Run(tt.word+"="+tt.want, func(t *testing.T) {
			caser := NewCaser()
			caser.AddAcronyms(tt.acronyms)
			got := caser.ToLowerGoIdent(tt.word)
			assert.Equal(t, tt.want, got)
		})
	}
}
