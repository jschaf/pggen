package texts

import "testing"

func TestDedent(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{"empty", "", ""},
		{"only whitespace", "\n ", ""},
		{"trailing newline", "foo\n ", "foo"},
		{"trailing newline + whitespace", "foo\n   ", "foo"},
		{"simple", "foo", "foo"},
		{"leading space 1 line", "   foo", "foo"},
		{"trailing space 1 line", "foo   ", "foo"},
		{"leading + trailing space 1 line", "  foo   ", "foo"},
		{"preceding newline", "\n   foo", "foo"},
		{"preceding newline", "\n   foo \n bar  \n", "  foo\nbar"},
		{"leading space same 3 lines", "  foo\n  bar\n  qux", "foo\nbar\nqux"},
		{"leading space diff 3 lines", "   foo\n  bar\n  qux", " foo\nbar\nqux"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := Dedent(tt.input); got != tt.want {
				t.Errorf("Dedent():\n '%s'\nwant:\n'%s'", got, tt.want)
			}
		})
	}
}
