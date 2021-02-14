package pg

import (
	"context"
	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	"github.com/jackc/pgtype"
	"github.com/jschaf/pggen/internal/pg/pgoid"
	"github.com/jschaf/pggen/internal/pgtest"
	"github.com/jschaf/pggen/internal/texts"
	"sort"
	"testing"
)

func TestNewTypeFetcher(t *testing.T) {
	tests := []struct {
		name     string
		schema   string
		fetchOID interface{} // oid type or string name of an type
		wants    []Type
	}{
		{
			name:     "basic int",
			schema:   "",
			fetchOID: Int4.ID,
			wants:    []Type{Int4},
		},
		{
			name:     "Void",
			schema:   "",
			fetchOID: pgoid.Void,
			wants:    []Type{Void},
		},
		{
			name:     "enum",
			schema:   `CREATE TYPE device_type AS ENUM ('computer', 'phone');`,
			fetchOID: "device_type",
			wants: []Type{
				EnumType{
					ID:        0, // set in test
					Name:      "device_type",
					Labels:    []string{"computer", "phone"},
					Orders:    []float32{1, 2},
					ChildOIDs: nil, // ignored
				},
			},
		},
		{
			name:     "composite table",
			schema:   `CREATE TABLE qux (id text, foo int8);`,
			fetchOID: "qux",
			wants: []Type{
				Int8,
				CompositeType{
					ID:          0, // set in test
					Name:        "qux",
					ColumnNames: []string{"id", "foo"},
					ColumnTypes: []Type{Text, Int8},
				},
				Text,
			},
		},
		{
			name: "composite types - depth 2",
			schema: texts.Dedent(`
				CREATE TYPE inventory_item AS (name text);
				CREATE TABLE qux (item inventory_item, foo int8);
			`),
			fetchOID: "qux",
			wants: []Type{
				CompositeType{
					ID:          0, // ignored
					Name:        "inventory_item",
					ColumnNames: []string{"name"},
					ColumnTypes: []Type{Text},
				},
				CompositeType{
					ID:          0, // set in test
					Name:        "qux",
					ColumnNames: []string{"item", "foo"},
					ColumnTypes: []Type{
						CompositeType{
							Name:        "inventory_item",
							ColumnNames: []string{"name"},
							ColumnTypes: []Type{Text},
						},
						Int8,
					},
				},
				Int8,
				Text,
			},
		},
		{
			name: "custom base type",
			schema: texts.Dedent(`
				-- New base type my_int.
				-- https://stackoverflow.com/a/45190420/30900
				CREATE TYPE my_int;

				CREATE FUNCTION my_int_in(cstring) RETURNS my_int
					LANGUAGE internal
					IMMUTABLE STRICT PARALLEL SAFE AS 'int2in';

				CREATE FUNCTION my_int_out(my_int) RETURNS cstring
					LANGUAGE internal
					IMMUTABLE STRICT PARALLEL SAFE AS 'int2out';

				CREATE TYPE my_int (
					INPUT = my_int_in,
					OUTPUT = my_int_out,
					LIKE = smallint,
					CATEGORY = 'N',
					PREFERRED = FALSE,
					DELIMITER = ',',
					COLLATABLE = FALSE
				);
			`),
			fetchOID: "my_int",
			wants: []Type{
				UnknownType{
					ID:     0, // set in test
					Name:   "my_int",
					PgKind: KindBaseType,
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			conn, cleanup := pgtest.NewPostgresSchemaString(t, tt.schema)
			defer cleanup()
			querier := NewQuerier(conn)

			// Act.
			fetcher := NewTypeFetcher(conn)
			oid := findOIDVal(t, tt.fetchOID, querier)
			gotTypeMap, err := fetcher.FindTypesByOIDs(uint32(oid))
			if err != nil {
				t.Fatal(err)
			}
			gotTypes := make([]Type, 0, len(gotTypeMap))
			for _, typ := range gotTypeMap {
				gotTypes = append(gotTypes, typ)
			}

			// Set the OID since we don't know it ahead of time.
			wantTypes := make([]Type, len(tt.wants))
			for i, want := range tt.wants {
				switch typ := want.(type) {
				case BaseType:
					typ.ID = findOIDVal(t, typ.Name, querier)
					wantTypes[i] = typ
				case VoidType:
					wantTypes[i] = VoidType{}
				case EnumType:
					typ.ID = findOIDVal(t, typ.Name, querier)
					wantTypes[i] = typ
				case ArrayType:
					typ.ID = findOIDVal(t, typ.Name, querier)
					wantTypes[i] = typ
				case DomainType:
					typ.ID = findOIDVal(t, typ.Name, querier)
					wantTypes[i] = typ
				case CompositeType:
					typ.ID = findOIDVal(t, typ.Name, querier)
					wantTypes[i] = typ
				case UnknownType:
					typ.ID = findOIDVal(t, typ.Name, querier)
					wantTypes[i] = typ
				default:
					t.Fatalf("unhandled type kind: %T", typ)
				}
			}

			opts := cmp.Options{
				cmpopts.IgnoreFields(EnumType{}, "ChildOIDs"),
				cmpopts.IgnoreFields(CompositeType{}, "ID"),
			}
			sortTypes(wantTypes)
			sortTypes(gotTypes)
			if diff := cmp.Diff(wantTypes, gotTypes, opts); diff != "" {
				t.Errorf("FetchOIDTypes() mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

// Get the OID by name if fetchOID was a string, or just return the OID.
func findOIDVal(t *testing.T, fetchOID interface{}, querier *DBQuerier) pgtype.OID {
	switch rawOID := fetchOID.(type) {
	case string:
		oid, err := querier.FindOIDByName(context.Background(), rawOID)
		if err != nil {
			t.Fatalf("find oid by name %s: %s", rawOID, err)
		}
		return oid
	case pgtype.OID:
		return rawOID
	case int:
		return pgtype.OID(rawOID)
	default:
		t.Fatalf("unhandled oid test value type %T: %v", rawOID, rawOID)
		return 0
	}
}

func sortTypes(types []Type) {
	sort.Slice(types, func(i, j int) bool {
		return types[i].String() < types[j].String()
	})
}