package golang

import (
	"flag"
	"github.com/jschaf/pggen/internal/casing"
	"github.com/jschaf/pggen/internal/codegen/golang/gotype"
	"github.com/jschaf/pggen/internal/pg"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"os"
	"strings"
	"testing"
)

var update = flag.Bool("update", false, "update integration tests if true")

func TestDeclarers(t *testing.T) {
	caser := casing.NewCaser()
	caser.AddAcronym("ios", "IOS")
	emptyPkgPath := ""
	pgTypeSomeTable := pg.CompositeType{
		Name:        "some_table",
		ColumnNames: []string{"foo", "bar_baz"},
		ColumnTypes: []pg.Type{pg.Int2, pg.Text},
	}
	goTypeSomeTable := gotype.CompositeType{
		PgComposite: pgTypeSomeTable,
		PkgPath:     "example.com/foo",
		Pkg:         "foo",
		Name:        "SomeTable",
		FieldNames:  []string{"Foo", "BarBaz"},
		FieldTypes:  []gotype.Type{gotype.Int16, gotype.PgText},
	}
	tests := []struct {
		name    string
		typ     gotype.Type
		pkgPath string
	}{
		{
			name:    "composite",
			typ:     goTypeSomeTable,
			pkgPath: "example.com/foo",
		},
		{
			name: "composite_array",
			typ: gotype.ArrayType{
				PkgPath: "example.com/arr",
				Pkg:     "bar",
				Name:    "SomeArray",
				PgArray: pg.ArrayType{Name: "_some_array", ElemType: pgTypeSomeTable},
				Elem:    goTypeSomeTable,
			},
			pkgPath: "example.com/foo",
		},
		{
			name: "composite_enum",
			typ: gotype.CompositeType{
				PgComposite: pg.CompositeType{
					Name:        "some_table_enum",
					ColumnNames: []string{"foo"},
					ColumnTypes: []pg.Type{pg.EnumType{Name: "some_table_enum"}},
				},
				PkgPath:    "example.com/foo",
				Pkg:        "foo",
				Name:       "SomeTableEnum",
				FieldNames: []string{"Foo"},
				FieldTypes: []gotype.Type{
					gotype.NewEnumType(
						emptyPkgPath,
						pg.EnumType{Name: "device_type", Labels: []string{"ios", "mobile"}},
						caser,
					),
				},
			},
			pkgPath: "example.com/foo",
		},
		{
			name: "composite_nested",
			typ: gotype.CompositeType{
				PgComposite: pg.CompositeType{
					Name:        "some_table_nested",
					ColumnNames: []string{"foo", "bar_baz"},
					ColumnTypes: []pg.Type{
						pg.CompositeType{
							Name:        "foo_type",
							ColumnNames: []string{"alpha"},
							ColumnTypes: []pg.Type{pg.Text},
						},
						pg.Text,
					},
				},
				PkgPath:    "example.com/foo",
				Pkg:        "foo",
				Name:       "SomeTableNested",
				FieldNames: []string{"Foo", "BarBaz"},
				FieldTypes: []gotype.Type{
					gotype.CompositeType{
						PgComposite: pg.CompositeType{
							Name:        "foo_type",
							ColumnNames: []string{"alpha"},
							ColumnTypes: []pg.Type{pg.Text},
						},
						PkgPath:    "example.com/foo",
						Pkg:        "foo",
						Name:       "FooType",
						FieldNames: []string{"Alpha"},
						FieldTypes: []gotype.Type{gotype.PgText},
					},
					gotype.PgText,
				},
			},
			pkgPath: "example.com/foo",
		},
		{
			name: "enum_escaping",
			typ: gotype.NewEnumType(
				emptyPkgPath,
				pg.EnumType{Name: "quoting", Labels: []string{"\"\n\t", "`\"`"}},
				casing.NewCaser(),
			),
		},
		{
			name: "enum_simple",
			typ: gotype.NewEnumType(
				emptyPkgPath,
				pg.EnumType{Name: "device_type", Labels: []string{"ios", "mobile"}},
				caser,
			),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name+"_input", func(t *testing.T) {
			golden := "testdata/declarer_" + tt.name + ".input.golden"
			decls := FindInputDeclarers(tt.typ).ListAll()
			sb := &strings.Builder{}
			for i, decl := range decls {
				s, err := decl.Declare(tt.pkgPath)
				if err != nil {
					t.Fatal(err)
				}
				sb.WriteString(s)
				if i < len(decls)-1 {
					sb.WriteString("\n\n")
				}
			}
			got := sb.String()

			if *update {
				err := os.WriteFile(golden, []byte(got), 0644)
				require.NoError(t, err)
				return
			}

			want, err := os.ReadFile(golden)
			require.NoError(t, err)
			assert.Equal(t, string(want), got)
		})

		t.Run(tt.name+"_output", func(t *testing.T) {
			golden := "testdata/declarer_" + tt.name + ".output.golden"
			decls := FindOutputDeclarers(tt.typ).ListAll()
			sb := &strings.Builder{}
			for i, decl := range decls {
				s, err := decl.Declare(tt.pkgPath)
				if err != nil {
					t.Fatal(err)
				}
				sb.WriteString(s)
				if i < len(decls)-1 {
					sb.WriteString("\n\n")
				}
			}
			got := sb.String()

			if *update {
				err := os.WriteFile(golden, []byte(got), 0644)
				require.NoError(t, err)
				return
			}

			want, err := os.ReadFile(golden)
			require.NoError(t, err)
			assert.Equal(t, string(want), got)
		})
	}
}
