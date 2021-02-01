package pg

import (
	"context"
	"fmt"
	"github.com/jackc/pgtype"
	"github.com/jackc/pgx/v4"
	"github.com/jschaf/pggen/internal/pg/pgoid"
	"sync"
	"time"
)

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
	OID  pgtype.OID // pg_type.oid: row identifier
	Name string     // pg_type.typname: data type name
}

type (
	// BaseType is a fundamental Postgres type like text and bool.
	// https://www.postgresql.org/docs/13/catalog-pg-type.html
	BaseType struct {
		ID         pgtype.OID     // pg_type.oid: row identifier
		Name       string         // pg_type.typname: data type name
		Kind       TypeKind       // pg_type.typtype: the kind of type
		Composite  *CompositeType // pg_type.typrelid: composite type only, the pg_class for the type
		Dimensions int            // pg_type.typndims: domains on array type only 0 otherwise, number of array dimensions,
	}

	EnumType struct {
		ID pgtype.OID
		// The name of the enum, like 'device_type' in:
		//     CREATE TYPE device_type AS ENUM ('foo');
		Name string
		// All textual labels for this enum in sort order.
		Labels []string
		// When an enum type is created, its members are assigned sort-order
		// positions 1..n. But members added later might be given negative or
		// fractional values of enumsortorder. The only requirement on these
		// values is that they be correctly ordered and unique within each enum
		// type.
		Orders    []float32
		ChildOIDs []pgtype.OID
	}

	// DomainType is a user-create domain type.
	DomainType struct {
		ID         pgtype.OID // pg_type.oid: row identifier
		Name       string     // pg_type.typname: data type name
		IsNotNull  bool       // pg_type.typnotnull: domains only, not null constraint for domains
		HasDefault bool       // pg_type.typdefault: domains only, if there's a default value
		BaseType   BaseType   // pg_type.typbasetype: domains only, the base type
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
	PgNodeTree       = Type{OID: pgoid.PgNodeTree, Name: "pg_node_tree"}
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
	OIDArray         = Type{OID: 1028, Name: "_oid"}
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

	typeMap = map[uint32]Type{
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
func FetchOIDTypes(conn *pgx.Conn, oids ...uint32) (map[pgtype.OID]Type, error) {
	types := make(map[pgtype.OID]Type, len(oids))
	oidsToFetch := make([]uint32, 0, len(oids))
	typeMapLock.Lock()
	for _, oid := range oids {
		if t, ok := typeMap[oid]; ok {
			types[pgtype.OID(oid)] = t
		} else {
			oidsToFetch = append(oidsToFetch, oid)
		}
	}
	typeMapLock.Unlock()

	querier := NewQuerier(conn)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	enums, err := querier.FindEnumTypes(ctx, oidsToFetch)
	if err != nil {
		return nil, fmt.Errorf("find enum oid types: %w", err)
	}
	// TODO: aggregate all enum elements into a single row.
	for _, enum := range enums {
		types[enum.OID] = Type{
			OID:  enum.OID,
			Name: enum.TypeName.String,
		}
	}

	// Check that we found all OIDs.
	for _, oid := range oids {
		if _, ok := types[pgtype.OID(oid)]; !ok {
			return nil, fmt.Errorf("did not find all OIDs; missing OID %d", oid)
		}
	}

	return types, nil
}
