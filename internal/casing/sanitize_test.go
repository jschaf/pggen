package casing

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestSanitize(t *testing.T) {
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
			got := sanitize(tt.str)
			assert.Equal(t, tt.want, got)
		})
	}
}
