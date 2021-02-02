package golang

import (
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
				PgName:   "device_type",
				GoName:   "DeviceType",
				GoLabels: []string{"DeviceTypeIOS", "DeviceTypeWeb"},
				PgLabels: []string{"ios", "web"},
			},
			texts.Dedent(`
				// DeviceType represents the Postgres enum type device_type.
				type DeviceType string

				const (
					DeviceTypeIOS DeviceType = "ios"
					DeviceTypeWeb DeviceType = "web"
				)

				func (d DeviceType) String() string { return string(d) }
			`),
		},
		{
			"escaping",
			EnumDeclarer{
				PgName:   "quoting",
				GoName:   "Quoting",
				GoLabels: []string{"QuotingQuoteNewlineTab", "QuotingBacktickQuoteBacktick"},
				PgLabels: []string{"\"\n\t", "`\"`"},
			},
			texts.Dedent(`
				// Quoting represents the Postgres enum type quoting.
				type Quoting string

				const (
					QuotingQuoteNewlineTab Quoting = "\"\n\t"
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
			assert.Equal(t, tt.want+"\n", got)
		})
	}
}
