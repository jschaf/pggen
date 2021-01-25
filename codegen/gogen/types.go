package gogen

import (
	"github.com/jackc/pgtype"
	"github.com/jschaf/pggen/internal/pg"
)

var goPgTypes = map[pg.OIDInt]string{
	pgtype.BoolOID:             "pgtype.Bool",
	pgtype.QCharOID:            "pgtype.QChar",
	pgtype.NameOID:             "pgtype.Name",
	pgtype.Int8OID:             "pgtype.Int8",
	pgtype.Int2OID:             "pgtype.Int2",
	pgtype.Int4OID:             "pgtype.Int4",
	pgtype.TextOID:             "pgtype.Text",
	pgtype.OIDOID:              "pgtype.OID",
	pgtype.TIDOID:              "pgtype.TID",
	pgtype.XIDOID:              "pgtype.XID",
	pgtype.CIDOID:              "pgtype.CID",
	pgtype.JSONOID:             "pgtype.JSON",
	pgtype.PointOID:            "pgtype.Point",
	pgtype.LsegOID:             "pgtype.Lseg",
	pgtype.PathOID:             "pgtype.Path",
	pgtype.BoxOID:              "pgtype.Box",
	pgtype.PolygonOID:          "pgtype.Polygon",
	pgtype.LineOID:             "pgtype.Line",
	pgtype.CIDROID:             "pgtype.CIDR",
	pgtype.CIDRArrayOID:        "pgtype.CIDRArray",
	pgtype.Float4OID:           "pgtype.Float4",
	pgtype.Float8OID:           "pgtype.Float8",
	pgtype.UnknownOID:          "pgtype.Unknown",
	pgtype.CircleOID:           "pgtype.Circle",
	pgtype.MacaddrOID:          "pgtype.Macaddr",
	pgtype.InetOID:             "pgtype.Inet",
	pgtype.BoolArrayOID:        "pgtype.BoolArray",
	pgtype.ByteaArrayOID:       "pgtype.ByteaArray",
	pgtype.Int2ArrayOID:        "pgtype.Int2Array",
	pgtype.Int4ArrayOID:        "pgtype.Int4Array",
	pgtype.TextArrayOID:        "pgtype.TextArray",
	pgtype.BPCharArrayOID:      "pgtype.BPCharArray",
	pgtype.VarcharArrayOID:     "pgtype.VarcharArray",
	pgtype.Int8ArrayOID:        "pgtype.Int8Array",
	pgtype.Float4ArrayOID:      "pgtype.Float4Array",
	pgtype.Float8ArrayOID:      "pgtype.Float8Array",
	pgtype.ACLItemOID:          "pgtype.ACLItem",
	pgtype.ACLItemArrayOID:     "pgtype.ACLItemArray",
	pgtype.InetArrayOID:        "pgtype.InetArray",
	pgtype.BPCharOID:           "pgtype.BPChar",
	pgtype.VarcharOID:          "pgtype.Varchar",
	pgtype.DateOID:             "pgtype.Date",
	pgtype.TimeOID:             "pgtype.Time",
	pgtype.TimestampOID:        "pgtype.Timestamp",
	pgtype.TimestampArrayOID:   "pgtype.TimestampArray",
	pgtype.DateArrayOID:        "pgtype.DateArray",
	pgtype.TimestamptzOID:      "pgtype.Timestamptz",
	pgtype.TimestamptzArrayOID: "pgtype.TimestamptzArray",
	pgtype.IntervalOID:         "pgtype.Interval",
	pgtype.NumericArrayOID:     "pgtype.NumericArray",
	pgtype.BitOID:              "pgtype.Bit",
	pgtype.VarbitOID:           "pgtype.Varbit",
	pgtype.NumericOID:          "pgtype.Numeric",
	pgtype.RecordOID:           "pgtype.Record",
	pgtype.UUIDOID:             "pgtype.UUID",
	pgtype.UUIDArrayOID:        "pgtype.UUIDArray",
	pgtype.JSONBOID:            "pgtype.JSONB",
	pgtype.JSONBArrayOID:       "pgtype.JSONBArray",
	pgtype.Int4rangeOID:        "pgtype.Int4range",
	pgtype.NumrangeOID:         "pgtype.Numrange",
	pgtype.TsrangeOID:          "pgtype.Tsrange",
	pgtype.TstzrangeOID:        "pgtype.Tstzrange",
	pgtype.DaterangeOID:        "pgtype.Daterange",
	pgtype.Int8rangeOID:        "pgtype.Int8range",
}

// pgToGoType maps a Postgres type to a Go type.
func pgToGoType(pgType pg.Type) string {
	if goType, ok := goPgTypes[pgType.OID]; ok {
		return goType
	}
	return pgType.Name
}
