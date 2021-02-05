package golang

import (
	"fmt"
	"github.com/jackc/pgtype"
	"github.com/jschaf/pggen/internal/casing"
	"github.com/jschaf/pggen/internal/pg"
	"github.com/jschaf/pggen/internal/pg/pgoid"
)

// TypeResolver handles the mapping between Postgres and Go types.
type TypeResolver struct {
	caser     casing.Caser
	overrides map[string]string
}

func NewTypeResolver(c casing.Caser, overrides map[string]string) TypeResolver {
	overs := make(map[string]string, len(overrides))
	for k, v := range overrides {
		overs[k] = v
		// Type aliases.
		// https://www.postgresql.org/docs/13/datatype.html#DATATYPE-TABLE
		switch k {
		case "bigint":
			overs["int8"] = v
		case "int8":
			overs["bigint"] = v

		case "bigserial":
			overs["serial8"] = v
		case "serial8":
			overs["bigserial"] = v

		case "boolean":
			overs["bool"] = v
		case "bool":
			overs["boolean"] = v

		case "double precision":
			overs["float8"] = v
		case "float8":
			overs["double precision"] = v

		case "int":
			overs["integer"] = v
			overs["int4"] = v
		case "integer":
			overs["int"] = v
			overs["int4"] = v
		case "int4":
			overs["integer"] = v
			overs["int"] = v

			// TODO: numeric, multi word aliases

		case "real":
			overs["float4"] = v
		case "float4":
			overs["real"] = v

		case "smallint":
			overs["int2"] = v
		case "int2":
			overs["smallint"] = v

		case "smallserial":
			overs["serial2"] = v
		case "serial2":
			overs["smallserial"] = v

		case "serial":
			overs["serial4"] = v
		case "serial4":
			overs["serial"] = v
		}
	}
	return TypeResolver{caser: c, overrides: overs}
}

// Resolve maps a Postgres type to a Go type.
func (tr TypeResolver) Resolve(pgt pg.Type, nullable bool, pkgPath string) (Type, error) {
	// Custom user override.
	if goType, ok := tr.overrides[pgt.String()]; ok {
		return NewOpaqueType(goType), nil
	}

	// Known type.
	if knownType, ok := knownTypesByOID[pgt.OID()]; ok {
		if nullable || knownType.nonNullable == "" {
			return NewOpaqueType(knownType.nullable), nil
		}
		return NewOpaqueType(knownType.nonNullable), nil
	}

	// New type that pggen will define in generated source code.
	switch pgt := pgt.(type) {
	case pg.EnumType:
		enum := NewEnumType(pkgPath, pgt, tr.caser)
		return enum, nil
	}

	return nil, fmt.Errorf("no go type found for Postgres type %s oid=%d", pgt.String(), pgt.OID())
}

// knownGoType is the nullable and non-nullable types for a Postgres type.
// We use a non-nullable type when possible because it offers better ergonomics.
// It's nicer to get a string as an output column rather than pgtype.Text which
// requires checking for a null value.
type knownGoType struct{ nullable, nonNullable string }

var knownTypesByOID = map[pgtype.OID]knownGoType{
	pgtype.BoolOID:             {"github.com/jackc/pgtype.Bool", "bool"},
	pgtype.QCharOID:            {"github.com/jackc/pgtype.QChar", ""},
	pgtype.NameOID:             {"github.com/jackc/pgtype.Name", ""},
	pgtype.Int8OID:             {"github.com/jackc/pgtype.Int8", "int"},
	pgtype.Int2OID:             {"github.com/jackc/pgtype.Int2", "int16"},
	pgtype.Int4OID:             {"github.com/jackc/pgtype.Int4", "int32"},
	pgtype.TextOID:             {"github.com/jackc/pgtype.Text", "string"},
	pgtype.OIDOID:              {"github.com/jackc/pgtype.OID", ""},
	pgtype.TIDOID:              {"github.com/jackc/pgtype.TID", ""},
	pgtype.XIDOID:              {"github.com/jackc/pgtype.XID", ""},
	pgtype.CIDOID:              {"github.com/jackc/pgtype.CID", ""},
	pgtype.JSONOID:             {"github.com/jackc/pgtype.JSON", ""},
	pgtype.PointOID:            {"github.com/jackc/pgtype.Point", ""},
	pgtype.LsegOID:             {"github.com/jackc/pgtype.Lseg", ""},
	pgtype.PathOID:             {"github.com/jackc/pgtype.Path", ""},
	pgtype.BoxOID:              {"github.com/jackc/pgtype.Box", ""},
	pgtype.PolygonOID:          {"github.com/jackc/pgtype.Polygon", ""},
	pgtype.LineOID:             {"github.com/jackc/pgtype.Line", ""},
	pgtype.CIDROID:             {"github.com/jackc/pgtype.CIDR", ""},
	pgtype.CIDRArrayOID:        {"github.com/jackc/pgtype.CIDRArray", ""},
	pgtype.Float4OID:           {"github.com/jackc/pgtype.Float4", ""},
	pgtype.Float8OID:           {"github.com/jackc/pgtype.Float8", ""},
	pgoid.OIDArray:             {"[]uint32", ""},
	pgtype.UnknownOID:          {"github.com/jackc/pgtype.Unknown", ""},
	pgtype.CircleOID:           {"github.com/jackc/pgtype.Circle", ""},
	pgtype.MacaddrOID:          {"github.com/jackc/pgtype.Macaddr", ""},
	pgtype.InetOID:             {"github.com/jackc/pgtype.Inet", ""},
	pgtype.BoolArrayOID:        {"github.com/jackc/pgtype.BoolArray", ""},
	pgtype.ByteaArrayOID:       {"github.com/jackc/pgtype.ByteaArray", ""},
	pgtype.Int2ArrayOID:        {"github.com/jackc/pgtype.Int2Array", "[]int16"},
	pgtype.Int4ArrayOID:        {"github.com/jackc/pgtype.Int4Array", "[]int32"},
	pgtype.TextArrayOID:        {"github.com/jackc/pgtype.TextArray", "[]string"},
	pgtype.BPCharArrayOID:      {"github.com/jackc/pgtype.BPCharArray", ""},
	pgtype.VarcharArrayOID:     {"github.com/jackc/pgtype.VarcharArray", ""},
	pgtype.Int8ArrayOID:        {"github.com/jackc/pgtype.Int8Array", "[]int"},
	pgtype.Float4ArrayOID:      {"github.com/jackc/pgtype.Float4Array", "[]float32"},
	pgtype.Float8ArrayOID:      {"github.com/jackc/pgtype.Float8Array", "[]float64"},
	pgtype.ACLItemOID:          {"github.com/jackc/pgtype.ACLItem", ""},
	pgtype.ACLItemArrayOID:     {"github.com/jackc/pgtype.ACLItemArray", ""},
	pgtype.InetArrayOID:        {"github.com/jackc/pgtype.InetArray", ""},
	pgoid.MacaddrArray:         {"github.com/jackc/pgtype.MacaddrArray", ""},
	pgtype.BPCharOID:           {"github.com/jackc/pgtype.BPChar", ""},
	pgtype.VarcharOID:          {"github.com/jackc/pgtype.Varchar", ""},
	pgtype.DateOID:             {"github.com/jackc/pgtype.Date", ""},
	pgtype.TimeOID:             {"github.com/jackc/pgtype.Time", ""},
	pgtype.TimestampOID:        {"github.com/jackc/pgtype.Timestamp", ""},
	pgtype.TimestampArrayOID:   {"github.com/jackc/pgtype.TimestampArray", ""},
	pgtype.DateArrayOID:        {"github.com/jackc/pgtype.DateArray", ""},
	pgtype.TimestamptzOID:      {"github.com/jackc/pgtype.Timestamptz", ""},
	pgtype.TimestamptzArrayOID: {"github.com/jackc/pgtype.TimestamptzArray", ""},
	pgtype.IntervalOID:         {"github.com/jackc/pgtype.Interval", ""},
	pgtype.NumericArrayOID:     {"github.com/jackc/pgtype.NumericArray", ""},
	pgtype.BitOID:              {"github.com/jackc/pgtype.Bit", ""},
	pgtype.VarbitOID:           {"github.com/jackc/pgtype.Varbit", ""},
	pgtype.NumericOID:          {"github.com/jackc/pgtype.Numeric", ""},
	pgtype.RecordOID:           {"github.com/jackc/pgtype.Record", ""},
	pgtype.UUIDOID:             {"github.com/jackc/pgtype.UUID", ""},
	pgtype.UUIDArrayOID:        {"github.com/jackc/pgtype.UUIDArray", ""},
	pgtype.JSONBOID:            {"github.com/jackc/pgtype.JSONB", ""},
	pgtype.JSONBArrayOID:       {"github.com/jackc/pgtype.JSONBArray", ""},
	pgtype.Int4rangeOID:        {"github.com/jackc/pgtype.Int4range", ""},
	pgtype.NumrangeOID:         {"github.com/jackc/pgtype.Numrange", ""},
	pgtype.TsrangeOID:          {"github.com/jackc/pgtype.Tsrange", ""},
	pgtype.TstzrangeOID:        {"github.com/jackc/pgtype.Tstzrange", ""},
	pgtype.DaterangeOID:        {"github.com/jackc/pgtype.Daterange", ""},
	pgtype.Int8rangeOID:        {"github.com/jackc/pgtype.Int8range", ""},
}
