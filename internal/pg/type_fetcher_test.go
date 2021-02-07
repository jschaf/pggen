package pg

import (
	"context"
	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	"github.com/jackc/pgtype"
	"github.com/jschaf/pggen/internal/pgtest"
	"github.com/jschaf/pggen/internal/texts"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestNewTypeFetcher(t *testing.T) {
	tests := []struct {
		name     string
		schema   string
		fetchOID interface{} // oid type or string name of an type
		want     Type
	}{
		{
			name:     "basic int",
			schema:   "",
			fetchOID: Int4.ID,
			want:     Int4,
		},
		{
			name:     "enum",
			schema:   `CREATE TYPE device_type AS ENUM ('computer', 'phone');`,
			fetchOID: "device_type",
			want: EnumType{
				ID:        0, // set in test
				Name:      "device_type",
				Labels:    []string{"computer", "phone"},
				Orders:    []float32{1, 2},
				ChildOIDs: nil, // ignored
			},
		},
		{
			name:     "composite table",
			schema:   `CREATE TABLE qux (id text, foo int8);`,
			fetchOID: "qux",
			want: CompositeType{
				ID:          0, // set in test
				Name:        "qux",
				ColumnNames: []string{"id", "foo"},
				ColumnTypes: []Type{Text, Int8},
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
			want: UnknownType{
				ID:     0, // set in test
				Name:   "my_int",
				PgKind: KindBaseType,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			conn, cleanup := pgtest.NewPostgresSchemaString(t, tt.schema)
			defer cleanup()
			querier := NewQuerier(conn)

			// Get the OID by name if fetchOID was a string.
			var oid pgtype.OID
			switch rawOID := tt.fetchOID.(type) {
			case string:
				var err error
				oid, err = querier.FindOIDByName(context.Background(), rawOID)
				if err != nil {
					t.Fatalf("find oid by name %s: %s", rawOID, err)
				}
			case pgtype.OID:
				oid = rawOID
			default:
				t.Fatalf("unhandled oid test value type %T: %v", rawOID, rawOID)
			}

			// Act.
			fetcher := NewTypeFetcher(conn)
			types, err := fetcher.FindTypesByOIDs(uint32(oid))
			if err != nil {
				t.Fatal(err)
			}
			assert.Len(t, types, 1)
			var gotType Type
			for _, typ := range types {
				gotType = typ
				break
			}

			// Set the OID since we don't know it ahead of time.
			var wantType Type
			switch typ := tt.want.(type) {
			case BaseType:
				typ.ID = oid
				wantType = typ
			case EnumType:
				typ.ID = oid
				wantType = typ
			case ArrayType:
				typ.ID = oid
				wantType = typ
			case DomainType:
				typ.ID = oid
				wantType = typ
			case CompositeType:
				typ.ID = oid
				wantType = typ
			case UnknownType:
				typ.ID = oid
				wantType = typ
			default:
				t.Fatalf("unhandled type kind: %T", typ)
			}

			ignoreEnumChildren := cmpopts.IgnoreFields(EnumType{}, "ChildOIDs")
			if diff := cmp.Diff(wantType, gotType, ignoreEnumChildren); diff != "" {
				t.Errorf("FetchOIDTypes() mismatch (-want +got):\n%s", diff)
			}
		})
	}
}
