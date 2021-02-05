package golang

import (
	"github.com/jschaf/pggen/internal/casing"
	"github.com/jschaf/pggen/internal/pg"
	"github.com/jschaf/pggen/internal/texts"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestEnumDeclarer_Declare(t *testing.T) {
	tests := []struct {
		name string
		decl EnumDeclarer
		want string
	}{
		{
			"simple",
			EnumDeclarer{
				enum: NewEnumType(
					pg.EnumType{Name: "device_type", Labels: []string{"ios", "mobile"}},
					casing.NewCaser(),
				),
			},
			texts.Dedent(`
				// DeviceType represents the Postgres enum "device_type".
				type DeviceType string

				const (
					DeviceTypeIOS    DeviceType = "ios"
					DeviceTypeMobile DeviceType = "mobile"
				)

				func (d DeviceType) String() string { return string(d) }
			`),
		},
		{
			"escaping",
			EnumDeclarer{
				enum: NewEnumType(pg.EnumType{
					Name:   "quoting",
					Labels: []string{"\"\n\t", "`\"`"},
				}, casing.NewCaser()),
			},
			texts.Dedent(`
				// Quoting represents the Postgres enum "quoting".
				type Quoting string

				const (
					QuotingQuoteNewlineTab       Quoting = "\"\n\t"
					QuotingBacktickQuoteBacktick Quoting = "` + "`" + `\"` + "`" + `"
				)

				func (q Quoting) String() string { return string(q) }
			`),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := tt.decl.Declare()
			if err != nil {
				t.Fatal(err)
			}
			assert.Equal(t, tt.want, got)
		})
	}
}
