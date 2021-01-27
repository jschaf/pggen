package golang

import (
	"bytes"
	"fmt"
	"github.com/jackc/pgtype"
	"github.com/jschaf/pggen/internal/pg"
	"strings"
)

// pgToGoType maps a Postgres type to a Go type and its containing package.
func pgToGoType(pgType pg.Type, nullable bool) (pkg string, typ string, err error) {
	goType, ok := goPgTypes[pgType.OID]
	if !ok {
		return "", "", fmt.Errorf("no go type found for Postgres type %s oid=%d", pgType.Name, pgType.OID)
	}
	pkg, typ = goType.splitPkg(nullable)
	return pkg, typ, nil
}

// goType is the nullable and non-nullable types for a Postgres type.
// We use a non-nullable type when possible because it offers better ergonomics.
// It's nicer to get a string as an output column rather than pgtype.Text which
// requires checking for a null value.
type goType struct{ nullable, nonNullable string }

func (gt goType) splitPkg(nullable bool) (pkg string, typ string) {
	if nullable || gt.nonNullable == "" {
		return splitQualifiedType(gt.nullable)
	}
	return splitQualifiedType(gt.nonNullable)
}

// splitQualifiedType splits a qualified or builtin type like
// github.com/jackc/pgtype.Int8range into the package "github.com/jackc/pgtype"
// and type name "Int8range".
//
// For builtin types returns empty string as the pkg.
func splitQualifiedType(qualType string) (pkg string, typ string) {
	if !strings.ContainsRune(qualType, '.') {
		return "", qualType // builtin type like string
	}
	bs := []byte(qualType)
	idx := bytes.LastIndexByte(bs, '.')
	return string(bs[:idx]), string(bs[idx+1:])
}

var goPgTypes = map[pg.OIDInt]goType{
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
