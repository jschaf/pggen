package golang

import (
	"flag"
	"github.com/jschaf/pggen/internal/casing"
	"github.com/jschaf/pggen/internal/codegen/golang/gotype"
	"github.com/jschaf/pggen/internal/difftest"
	"github.com/jschaf/pggen/internal/pg"
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
	goTypeSomeTable := &gotype.CompositeType{
		PgComposite: pgTypeSomeTable,
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
			pkgPath: "example.com/foo",
			typ:     goTypeSomeTable,
		},
		{
			name:    "composite_array",
			pkgPath: "example.com/foo",
			typ: &gotype.ArrayType{
				PgArray: pg.ArrayType{Name: "_some_array", Elem: pgTypeSomeTable},
				Elem:    &gotype.ImportType{PkgPath: "example.com/foo", Type: goTypeSomeTable},
			},
		},
		{
			name:    "composite_enum",
			pkgPath: "example.com/foo",
			typ: &gotype.ImportType{
				PkgPath: "example.com/foo",
				Type: &gotype.CompositeType{
					PgComposite: pg.CompositeType{
						Name:        "some_table_enum",
						ColumnNames: []string{"foo"},
						ColumnTypes: []pg.Type{pg.EnumType{Name: "some_table_enum"}},
					},
					Name:       "SomeTableEnum",
					FieldNames: []string{"Foo"},
					FieldTypes: []gotype.Type{
						gotype.NewEnumType(
							emptyPkgPath,
							pg.EnumType{Name: "device_type", Labels: []string{"ios", "mobile"}},
							caser,
						),
					},
				}},
		},
		{
			name:    "composite_nested",
			pkgPath: "example.com/foo",
			typ: &gotype.ImportType{
				PkgPath: "example.com/foo",
				Type: &gotype.CompositeType{
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
					Name:       "SomeTableNested",
					FieldNames: []string{"Foo", "BarBaz"},
					FieldTypes: []gotype.Type{
						&gotype.ImportType{
							PkgPath: "example.com/foo",
							Type: &gotype.CompositeType{
								PgComposite: pg.CompositeType{
									Name:        "foo_type",
									ColumnNames: []string{"alpha"},
									ColumnTypes: []pg.Type{pg.Text},
								},
								Name:       "FooType",
								FieldNames: []string{"Alpha"},
								FieldTypes: []gotype.Type{gotype.PgText},
							},
						},
						gotype.PgText,
					},
				},
			},
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
			difftest.AssertSame(t, string(want), got)
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

			difftest.AssertSame(t, string(want), got)
		})
	}
}
