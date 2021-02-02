package casing

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestSanitizer_Sanitize(t *testing.T) {
	tests := []struct {
		str  string
		want string
	}{
		{"a", "a"},
		{"a.", "a_"},
		{"a.b", "a_b"},
		{"", ""},
		{"1", ""},
		{"1abc", "abc"},
		{"abc@123", "abc_123"},
		{"abc@!123", "abc_123"},
		{"T食", "T食"},
	}
	for _, tt := range tests {
		t.Run(tt.str, func(t *testing.T) {
			got := Sanitize(tt.str)
			assert.Equal(t, tt.want, got)
		})
	}
}
