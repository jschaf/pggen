package pg

import (
	"github.com/jackc/pgtype"
	"github.com/jschaf/pggen/internal/pg/pgoid"
)

// If you add to this list, also add to defaultKnownTypes below.
//
//goland:noinspection GoNameStartsWithPackageName
var (
	Bool             = BaseType{ID: pgtype.BoolOID, Name: "bool"}
	Bytea            = BaseType{ID: pgtype.ByteaOID, Name: "bytea"}
	QChar            = BaseType{ID: pgtype.QCharOID, Name: "char"}
	Name             = BaseType{ID: pgtype.NameOID, Name: "name"}
	Int8             = BaseType{ID: pgtype.Int8OID, Name: "int8"}
	Int2             = BaseType{ID: pgtype.Int2OID, Name: "int2"}
	Int4             = BaseType{ID: pgtype.Int4OID, Name: "int4"}
	Text             = BaseType{ID: pgtype.TextOID, Name: "text"}
	OID              = BaseType{ID: pgtype.OIDOID, Name: "oid"}
	TID              = BaseType{ID: pgtype.TIDOID, Name: "tid"}
	XID              = BaseType{ID: pgtype.XIDOID, Name: "xid"}
	CID              = BaseType{ID: pgtype.CIDOID, Name: "cid"}
	JSON             = BaseType{ID: pgtype.JSONOID, Name: "json"}
	PgNodeTree       = BaseType{ID: pgoid.PgNodeTree, Name: "pg_node_tree"}
	Point            = BaseType{ID: pgtype.PointOID, Name: "point"}
	Lseg             = BaseType{ID: pgtype.LsegOID, Name: "lseg"}
	Path             = BaseType{ID: pgtype.PathOID, Name: "path"}
	Box              = BaseType{ID: pgtype.BoxOID, Name: "box"}
	Polygon          = BaseType{ID: pgtype.PolygonOID, Name: "polygon"}
	Line             = BaseType{ID: pgtype.LineOID, Name: "line"}
	CIDR             = BaseType{ID: pgtype.CIDROID, Name: "cidr"}
	CIDRArray        = ArrayType{ID: pgtype.CIDRArrayOID, Name: "_cidr"}
	Float4           = BaseType{ID: pgtype.Float4OID, Name: "float4"}
	Float8           = BaseType{ID: pgtype.Float8OID, Name: "float8"}
	Unknown          = BaseType{ID: pgtype.UnknownOID, Name: "unknown"}
	Circle           = BaseType{ID: pgtype.CircleOID, Name: "circle"}
	Macaddr          = BaseType{ID: pgtype.MacaddrOID, Name: "macaddr"}
	Inet             = BaseType{ID: pgtype.InetOID, Name: "inet"}
	BoolArray        = ArrayType{ID: pgtype.BoolArrayOID, Name: "_bool"}
	ByteaArray       = ArrayType{ID: pgtype.ByteaArrayOID, Name: "_bytea"}
	Int2Array        = ArrayType{ID: pgtype.Int2ArrayOID, Name: "_int2"}
	Int4Array        = ArrayType{ID: pgtype.Int4ArrayOID, Name: "_int4"}
	TextArray        = ArrayType{ID: pgtype.TextArrayOID, Name: "_text"}
	BPCharArray      = ArrayType{ID: pgtype.BPCharArrayOID, Name: "_bpchar"}
	VarcharArray     = ArrayType{ID: pgtype.VarcharArrayOID, Name: "_varchar"}
	Int8Array        = ArrayType{ID: pgtype.Int8ArrayOID, Name: "_int8"}
	Float4Array      = ArrayType{ID: pgtype.Float4ArrayOID, Name: "_float4"}
	Float8Array      = ArrayType{ID: pgtype.Float8ArrayOID, Name: "_float8"}
	OIDArray         = ArrayType{ID: pgoid.OIDArray, Name: "_oid"}
	ACLItem          = BaseType{ID: pgtype.ACLItemOID, Name: "aclitem"}
	ACLItemArray     = ArrayType{ID: pgtype.ACLItemArrayOID, Name: "_aclitem"}
	InetArray        = ArrayType{ID: pgtype.InetArrayOID, Name: "_inet"}
	MacaddrArray     = ArrayType{ID: pgoid.MacaddrArray, Name: "_macaddr"}
	BPChar           = BaseType{ID: pgtype.BPCharOID, Name: "bpchar"}
	Varchar          = BaseType{ID: pgtype.VarcharOID, Name: "varchar"}
	Date             = BaseType{ID: pgtype.DateOID, Name: "date"}
	Time             = BaseType{ID: pgtype.TimeOID, Name: "time"}
	Timestamp        = BaseType{ID: pgtype.TimestampOID, Name: "timestamp"}
	TimestampArray   = ArrayType{ID: pgtype.TimestampArrayOID, Name: "_timestamp"}
	DateArray        = ArrayType{ID: pgtype.DateArrayOID, Name: "_date"}
	Timestamptz      = BaseType{ID: pgtype.TimestamptzOID, Name: "timestamptz"}
	TimestamptzArray = ArrayType{ID: pgtype.TimestamptzArrayOID, Name: "_timestamptz"}
	Interval         = BaseType{ID: pgtype.IntervalOID, Name: "interval"}
	NumericArray     = ArrayType{ID: pgtype.NumericArrayOID, Name: "_numeric"}
	Bit              = BaseType{ID: pgtype.BitOID, Name: "bit"}
	Varbit           = BaseType{ID: pgtype.VarbitOID, Name: "varbit"}
	Numeric          = BaseType{ID: pgtype.NumericOID, Name: "numeric"}
	Record           = BaseType{ID: pgtype.RecordOID, Name: "record"}
	Void             = VoidType{}
	UUID             = BaseType{ID: pgtype.UUIDOID, Name: "uuid"}
	UUIDArray        = ArrayType{ID: pgtype.UUIDArrayOID, Name: "_uuid"}
	JSONB            = BaseType{ID: pgtype.JSONBOID, Name: "jsonb"}
	JSONBArray       = ArrayType{ID: pgtype.JSONBArrayOID, Name: "_jsonb"}
	Int4range        = BaseType{ID: pgtype.Int4rangeOID, Name: "int4range"}
	Numrange         = BaseType{ID: pgtype.NumrangeOID, Name: "numrange"}
	Tsrange          = BaseType{ID: pgtype.TsrangeOID, Name: "tsrange"}
	Tstzrange        = BaseType{ID: pgtype.TstzrangeOID, Name: "tstzrange"}
	Daterange        = BaseType{ID: pgtype.DaterangeOID, Name: "daterange"}
	Int8range        = BaseType{ID: pgtype.Int8rangeOID, Name: "int8range"}
)

// All known Postgres types by OID.
var defaultKnownTypes = map[pgtype.OID]Type{
	pgtype.BoolOID:             Bool,
	pgtype.ByteaOID:            Bytea,
	pgtype.QCharOID:            QChar,
	pgtype.NameOID:             Name,
	pgtype.Int8OID:             Int8,
	pgtype.Int2OID:             Int2,
	pgtype.Int4OID:             Int4,
	pgtype.TextOID:             Text,
	pgtype.OIDOID:              OID,
	pgtype.TIDOID:              TID,
	pgtype.XIDOID:              XID,
	pgtype.CIDOID:              CID,
	pgtype.JSONOID:             JSON,
	pgoid.PgNodeTree:           PgNodeTree,
	pgtype.PointOID:            Point,
	pgtype.LsegOID:             Lseg,
	pgtype.PathOID:             Path,
	pgtype.BoxOID:              Box,
	pgtype.PolygonOID:          Polygon,
	pgtype.LineOID:             Line,
	pgtype.CIDROID:             CIDR,
	pgtype.CIDRArrayOID:        CIDRArray,
	pgtype.Float4OID:           Float4,
	pgtype.Float8OID:           Float8,
	pgtype.UnknownOID:          Unknown,
	pgtype.CircleOID:           Circle,
	pgtype.MacaddrOID:          Macaddr,
	pgtype.InetOID:             Inet,
	pgtype.BoolArrayOID:        BoolArray,
	pgtype.ByteaArrayOID:       ByteaArray,
	pgtype.Int2ArrayOID:        Int2Array,
	pgtype.Int4ArrayOID:        Int4Array,
	pgtype.TextArrayOID:        TextArray,
	pgtype.BPCharArrayOID:      BPCharArray,
	pgtype.VarcharArrayOID:     VarcharArray,
	pgtype.Int8ArrayOID:        Int8Array,
	pgtype.Float4ArrayOID:      Float4Array,
	pgtype.Float8ArrayOID:      Float8Array,
	pgoid.OIDArray:             OIDArray,
	pgtype.ACLItemOID:          ACLItem,
	pgtype.ACLItemArrayOID:     ACLItemArray,
	pgtype.InetArrayOID:        InetArray,
	pgoid.MacaddrArray:         MacaddrArray,
	pgtype.BPCharOID:           BPChar,
	pgtype.VarcharOID:          Varchar,
	pgtype.DateOID:             Date,
	pgtype.TimeOID:             Time,
	pgtype.TimestampOID:        Timestamp,
	pgtype.TimestampArrayOID:   TimestampArray,
	pgtype.DateArrayOID:        DateArray,
	pgtype.TimestamptzOID:      Timestamptz,
	pgtype.TimestamptzArrayOID: TimestamptzArray,
	pgtype.IntervalOID:         Interval,
	pgtype.NumericArrayOID:     NumericArray,
	pgtype.BitOID:              Bit,
	pgtype.VarbitOID:           Varbit,
	pgtype.NumericOID:          Numeric,
	pgtype.RecordOID:           Record,
	pgoid.Void:                 Void,
	pgtype.UUIDOID:             UUID,
	pgtype.UUIDArrayOID:        UUIDArray,
	pgtype.JSONBOID:            JSONB,
	pgtype.JSONBArrayOID:       JSONBArray,
	pgtype.Int4rangeOID:        Int4range,
	pgtype.NumrangeOID:         Numrange,
	pgtype.TsrangeOID:          Tsrange,
	pgtype.TstzrangeOID:        Tstzrange,
	pgtype.DaterangeOID:        Daterange,
	pgtype.Int8rangeOID:        Int8range,
}
