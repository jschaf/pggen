package pg

import (
	"fmt"
	"github.com/jackc/pgtype"
	"github.com/jackc/pgx/v4"
	"sync"
)

// OIDInt is the Postgres oid type.
type OIDInt = uint32

// TypeKind is b for a base type, c for a composite type (e.g., a table's row
// type), d for a domain, e for an enum type, p for a pseudo-type, or r for a
// range type. See also typrelid and typbasetype.
type TypeKind byte

//goland:noinspection GoUnusedConst
const (
	KindBaseType      TypeKind = 'b'
	KindCompositeType TypeKind = 'c' // table row type
	KindDomainType    TypeKind = 'd'
	KindEnumType      TypeKind = 'e'
	KindPseudoType    TypeKind = 'p'
	KindRangeType     TypeKind = 'r'
)

// Type is a Postgres type.
type Type struct {
	OID  OIDInt // pg_type.oid: row identifier
	Name string // pg_type.typname: data type name
}

type (
	// BaseType is a fundamental Postgres type like text and bool.
	// https://www.postgresql.org/docs/13/catalog-pg-type.html
	BaseType struct {
		ID         OIDInt         // pg_type.oid: row identifier
		Name       string         // pg_type.typname: data type name
		Kind       TypeKind       // pg_type.typtype: the kind of type
		Composite  *CompositeType // pg_type.typrelid: composite type only, the pg_class for the type
		Dimensions int            // pg_type.typndims: domains on array type only 0 otherwise, number of array dimensions,
	}

	// DomainType is a user-create domain type.
	DomainType struct {
		ID         OIDInt   // pg_type.oid: row identifier
		Name       string   // pg_type.typname: data type name
		IsNotNull  bool     // pg_type.typnotnull: domains only, not null constraint for domains
		HasDefault bool     // pg_type.typdefault: domains only, if there's a default value
		BaseType   BaseType // pg_type.typbasetype: domains only, the base type
	}

	// CompositeType is a type containing multiple columns and is represented as
	// a class. https://www.postgresql.org/docs/13/catalog-pg-class.html
	CompositeType struct {
		ID       uint32 // pg_class.oid: row identifier
		TypeName string // pg_class.relname: name of the composite type
		Columns  []Type // pg_attribute: information about columns of the composite type
	}
)

//goland:noinspection GoUnusedGlobalVariable
var (
	Bool             = Type{OID: pgtype.BoolOID, Name: "bool"}
	Bytea            = Type{OID: pgtype.ByteaOID, Name: "bytea"}
	QChar            = Type{OID: pgtype.QCharOID, Name: "char"}
	Name             = Type{OID: pgtype.NameOID, Name: "name"}
	Int8             = Type{OID: pgtype.Int8OID, Name: "int8"}
	Int2             = Type{OID: pgtype.Int2OID, Name: "int2"}
	Int4             = Type{OID: pgtype.Int4OID, Name: "int4"}
	Text             = Type{OID: pgtype.TextOID, Name: "text"}
	OID              = Type{OID: pgtype.OIDOID, Name: "oid"}
	TID              = Type{OID: pgtype.TIDOID, Name: "tid"}
	XID              = Type{OID: pgtype.XIDOID, Name: "xid"}
	CID              = Type{OID: pgtype.CIDOID, Name: "cid"}
	JSON             = Type{OID: pgtype.JSONOID, Name: "json"}
	Point            = Type{OID: pgtype.PointOID, Name: "point"}
	Lseg             = Type{OID: pgtype.LsegOID, Name: "lseg"}
	Path             = Type{OID: pgtype.PathOID, Name: "path"}
	Box              = Type{OID: pgtype.BoxOID, Name: "box"}
	Polygon          = Type{OID: pgtype.PolygonOID, Name: "polygon"}
	Line             = Type{OID: pgtype.LineOID, Name: "line"}
	CIDR             = Type{OID: pgtype.CIDROID, Name: "cidr"}
	CIDRArray        = Type{OID: pgtype.CIDRArrayOID, Name: "_cidr"}
	Float4           = Type{OID: pgtype.Float4OID, Name: "float4"}
	Float8           = Type{OID: pgtype.Float8OID, Name: "float8"}
	Unknown          = Type{OID: pgtype.UnknownOID, Name: "unknown"}
	Circle           = Type{OID: pgtype.CircleOID, Name: "circle"}
	Macaddr          = Type{OID: pgtype.MacaddrOID, Name: "macaddr"}
	Inet             = Type{OID: pgtype.InetOID, Name: "inet"}
	BoolArray        = Type{OID: pgtype.BoolArrayOID, Name: "_bool"}
	ByteaArray       = Type{OID: pgtype.ByteaArrayOID, Name: "_bytea"}
	Int2Array        = Type{OID: pgtype.Int2ArrayOID, Name: "_int2"}
	Int4Array        = Type{OID: pgtype.Int4ArrayOID, Name: "_int4"}
	TextArray        = Type{OID: pgtype.TextArrayOID, Name: "_text"}
	BPCharArray      = Type{OID: pgtype.BPCharArrayOID, Name: "_bpchar"}
	VarcharArray     = Type{OID: pgtype.VarcharArrayOID, Name: "_varchar"}
	Int8Array        = Type{OID: pgtype.Int8ArrayOID, Name: "_int8"}
	Float4Array      = Type{OID: pgtype.Float4ArrayOID, Name: "_float4"}
	Float8Array      = Type{OID: pgtype.Float8ArrayOID, Name: "_float8"}
	ACLItem          = Type{OID: pgtype.ACLItemOID, Name: "aclitem"}
	ACLItemArray     = Type{OID: pgtype.ACLItemArrayOID, Name: "_aclitem"}
	InetArray        = Type{OID: pgtype.InetArrayOID, Name: "_inet"}
	BPChar           = Type{OID: pgtype.BPCharOID, Name: "bpchar"}
	Varchar          = Type{OID: pgtype.VarcharOID, Name: "varchar"}
	Date             = Type{OID: pgtype.DateOID, Name: "date"}
	Time             = Type{OID: pgtype.TimeOID, Name: "time"}
	Timestamp        = Type{OID: pgtype.TimestampOID, Name: "timestamp"}
	TimestampArray   = Type{OID: pgtype.TimestampArrayOID, Name: "_timestamp"}
	DateArray        = Type{OID: pgtype.DateArrayOID, Name: "_date"}
	Timestamptz      = Type{OID: pgtype.TimestamptzOID, Name: "timestamptz"}
	TimestamptzArray = Type{OID: pgtype.TimestamptzArrayOID, Name: "_timestamptz"}
	Interval         = Type{OID: pgtype.IntervalOID, Name: "interval"}
	NumericArray     = Type{OID: pgtype.NumericArrayOID, Name: "_numeric"}
	Bit              = Type{OID: pgtype.BitOID, Name: "bit"}
	Varbit           = Type{OID: pgtype.VarbitOID, Name: "varbit"}
	Numeric          = Type{OID: pgtype.NumericOID, Name: "numeric"}
	Record           = Type{OID: pgtype.RecordOID, Name: "record"}
	UUID             = Type{OID: pgtype.UUIDOID, Name: "uuid"}
	UUIDArray        = Type{OID: pgtype.UUIDArrayOID, Name: "_uuid"}
	JSONB            = Type{OID: pgtype.JSONBOID, Name: "jsonb"}
	JSONBArray       = Type{OID: pgtype.JSONBArrayOID, Name: "_jsonb"}
	Int4range        = Type{OID: pgtype.Int4rangeOID, Name: "int4range"}
	Numrange         = Type{OID: pgtype.NumrangeOID, Name: "numrange"}
	Tsrange          = Type{OID: pgtype.TsrangeOID, Name: "tsrange"}
	Tstzrange        = Type{OID: pgtype.TstzrangeOID, Name: "tstzrange"}
	Daterange        = Type{OID: pgtype.DaterangeOID, Name: "daterange"}
	Int8range        = Type{OID: pgtype.Int8rangeOID, Name: "int8range"}
)

var (
	typeMapLock = &sync.Mutex{}

	typeMap = map[OIDInt]Type{
		pgtype.BoolOID:             Bool,
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
		pgtype.ACLItemOID:          ACLItem,
		pgtype.ACLItemArrayOID:     ACLItemArray,
		pgtype.InetArrayOID:        InetArray,
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
)

// FetchOIDTypes gets the Postgres type for each of the oids.
func FetchOIDTypes(_ *pgx.Conn, oids ...OIDInt) (map[OIDInt]Type, error) {
	types := make(map[OIDInt]Type, len(oids))
	oidsToFetch := make([]OIDInt, 0, len(oids))
	typeMapLock.Lock()
	for _, oid := range oids {
		if t, ok := typeMap[oid]; ok {
			types[oid] = t
		} else {
			// We'll use oidsToFetch once we fetch from the database.
			oidsToFetch = append(oidsToFetch, oid) // nolint
		}
	}
	typeMapLock.Unlock()

	// TODO: fetch from database

	// Check that we found all OIDs.
	for _, oid := range oids {
		if _, ok := types[oid]; !ok {
			return nil, fmt.Errorf("did not find all OIDs; missing OID %d", oid)
		}
	}

	return types, nil
}
