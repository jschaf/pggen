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

// Type is a Postgres type.
type Type interface {
	OID() pgtype.OID // pg_type.oid: row identifier
	String() string  // pg_type.typname: data type name
	Kind() TypeKind
}

type TypeKind byte

const (
	KindBaseType      TypeKind = 'b' // includes array types
	KindCompositeType TypeKind = 'c'
	KindDomainType    TypeKind = 'd'
	KindEnumType      TypeKind = 'e'
	KindPseudoType    TypeKind = 'p'
	KindRangeType     TypeKind = 'r'
)

func (k TypeKind) String() string {
	switch k {
	case KindBaseType:
		return "BaseType"
	case KindCompositeType:
		return "CompositeType"
	case KindDomainType:
		return "DomainType"
	case KindEnumType:
		return "EnumType"
	case KindPseudoType:
		return "PseudoType"
	case KindRangeType:
		return "RangeType"
	default:
		panic("unhandled TypeKind: " + string(k))
	}
}

type (
	// BaseType is a fundamental Postgres type like text and bool.
	// https://www.postgresql.org/docs/13/catalog-pg-type.html
	BaseType struct {
		ID   pgtype.OID // pg_type.oid: row identifier
		Name string     // pg_type.typname: data type name
	}

	// ArrayType is an array type where pg_type.typelem != 0 and the name begins
	// with an underscore.
	ArrayType struct {
		ID pgtype.OID // pg_type.oid: row identifier
		// The name of the type, like _int4. Array types in Postgres typically
		// begin with an underscore.
		// From pg_type.typname:
		Name string
		// pg_type.typelem: the element type of the array
		ElemType Type
	}

	EnumType struct {
		ID pgtype.OID // pg_type.oid: row identifier
		// The name of the enum, like 'device_type' in:
		//     CREATE TYPE device_type AS ENUM ('foo');
		// From pg_type.typname.
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
		Dimensions int        // pg_type.typndims: domains on array type only, 0 otherwise, number of array dimensions
	}

	// CompositeType is a type containing multiple columns and is represented as
	// a class. https://www.postgresql.org/docs/13/catalog-pg-class.html
	CompositeType struct {
		ID          pgtype.OID // pg_class.oid: row identifier
		Name        string     // pg_class.relname: name of the composite type
		ColumnNames []string   // pg_attribute.attname: names of the column, in order
		ColumnTypes []Type     // pg_attribute JOIN pg_type: information about columns of the composite type
	}

	// UnknownType is a Postgres type that's not a well-known type in typeMap, and
	// not an enum, domain, or composite type. The code generator might be able to
	// resolve this type from a user-provided mapping like --go-type my_int=int.
	UnknownType struct {
		ID     pgtype.OID // pg_type.oid: row identifier
		Name   string     // pg_type.typname: data type name
		PgKind TypeKind
	}
)

func (b BaseType) OID() pgtype.OID { return b.ID }
func (b BaseType) String() string  { return b.Name }
func (b BaseType) Kind() TypeKind  { return KindBaseType }

func (b ArrayType) OID() pgtype.OID { return b.ID }
func (b ArrayType) String() string  { return b.Name }
func (b ArrayType) Kind() TypeKind  { return KindBaseType } // arrays are base types

func (e EnumType) OID() pgtype.OID { return e.ID }
func (e EnumType) String() string  { return e.Name }
func (e EnumType) Kind() TypeKind  { return KindEnumType }

func (e DomainType) OID() pgtype.OID { return e.ID }
func (e DomainType) String() string  { return e.Name }
func (e DomainType) Kind() TypeKind  { return KindDomainType }

func (e CompositeType) OID() pgtype.OID { return e.ID }
func (e CompositeType) String() string  { return e.Name }
func (e CompositeType) Kind() TypeKind  { return KindCompositeType }

func (e UnknownType) OID() pgtype.OID { return e.ID }
func (e UnknownType) String() string  { return e.Name }
func (e UnknownType) Kind() TypeKind  { return e.PgKind }

//goland:noinspection GoUnusedGlobalVariable,GoNameStartsWithPackageName
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
	BoolArray        = BaseType{ID: pgtype.BoolArrayOID, Name: "_bool"}
	ByteaArray       = BaseType{ID: pgtype.ByteaArrayOID, Name: "_bytea"}
	Int2Array        = BaseType{ID: pgtype.Int2ArrayOID, Name: "_int2"}
	Int4Array        = BaseType{ID: pgtype.Int4ArrayOID, Name: "_int4"}
	TextArray        = BaseType{ID: pgtype.TextArrayOID, Name: "_text"}
	BPCharArray      = BaseType{ID: pgtype.BPCharArrayOID, Name: "_bpchar"}
	VarcharArray     = BaseType{ID: pgtype.VarcharArrayOID, Name: "_varchar"}
	Int8Array        = BaseType{ID: pgtype.Int8ArrayOID, Name: "_int8"}
	Float4Array      = BaseType{ID: pgtype.Float4ArrayOID, Name: "_float4"}
	Float8Array      = BaseType{ID: pgtype.Float8ArrayOID, Name: "_float8"}
	OIDArray         = BaseType{ID: pgoid.OIDArray, Name: "_oid"}
	ACLItem          = BaseType{ID: pgtype.ACLItemOID, Name: "aclitem"}
	ACLItemArray     = BaseType{ID: pgtype.ACLItemArrayOID, Name: "_aclitem"}
	InetArray        = BaseType{ID: pgtype.InetArrayOID, Name: "_inet"}
	MacaddrArray     = BaseType{ID: pgoid.MacaddrArray, Name: "_macaddr"}
	BPChar           = BaseType{ID: pgtype.BPCharOID, Name: "bpchar"}
	Varchar          = BaseType{ID: pgtype.VarcharOID, Name: "varchar"}
	Date             = BaseType{ID: pgtype.DateOID, Name: "date"}
	Time             = BaseType{ID: pgtype.TimeOID, Name: "time"}
	Timestamp        = BaseType{ID: pgtype.TimestampOID, Name: "timestamp"}
	TimestampArray   = BaseType{ID: pgtype.TimestampArrayOID, Name: "_timestamp"}
	DateArray        = BaseType{ID: pgtype.DateArrayOID, Name: "_date"}
	Timestamptz      = BaseType{ID: pgtype.TimestamptzOID, Name: "timestamptz"}
	TimestamptzArray = BaseType{ID: pgtype.TimestamptzArrayOID, Name: "_timestamptz"}
	Interval         = BaseType{ID: pgtype.IntervalOID, Name: "interval"}
	NumericArray     = BaseType{ID: pgtype.NumericArrayOID, Name: "_numeric"}
	Bit              = BaseType{ID: pgtype.BitOID, Name: "bit"}
	Varbit           = BaseType{ID: pgtype.VarbitOID, Name: "varbit"}
	Numeric          = BaseType{ID: pgtype.NumericOID, Name: "numeric"}
	Record           = BaseType{ID: pgtype.RecordOID, Name: "record"}
	UUID             = BaseType{ID: pgtype.UUIDOID, Name: "uuid"}
	UUIDArray        = BaseType{ID: pgtype.UUIDArrayOID, Name: "_uuid"}
	JSONB            = BaseType{ID: pgtype.JSONBOID, Name: "jsonb"}
	JSONBArray       = BaseType{ID: pgtype.JSONBArrayOID, Name: "_jsonb"}
	Int4range        = BaseType{ID: pgtype.Int4rangeOID, Name: "int4range"}
	Numrange         = BaseType{ID: pgtype.NumrangeOID, Name: "numrange"}
	Tsrange          = BaseType{ID: pgtype.TsrangeOID, Name: "tsrange"}
	Tstzrange        = BaseType{ID: pgtype.TstzrangeOID, Name: "tstzrange"}
	Daterange        = BaseType{ID: pgtype.DaterangeOID, Name: "daterange"}
	Int8range        = BaseType{ID: pgtype.Int8rangeOID, Name: "int8range"}
)

var (
	typeMapLock = &sync.Mutex{}

	typeMap = map[pgtype.OID]Type{
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
	// The return value
	types := make(map[pgtype.OID]Type, len(oids))

	// Figure out which OIDs that need type information.
	oidsToFetch := make([]uint32, 0, len(oids))
	typeMapLock.Lock()
	for _, oid := range oids {
		if t, ok := typeMap[pgtype.OID(oid)]; ok {
			types[pgtype.OID(oid)] = t
		} else {
			oidsToFetch = append(oidsToFetch, oid)
		}
	}
	typeMapLock.Unlock()

	querier := NewQuerier(conn)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Get enum types.
	enums, err := querier.FindEnumTypes(ctx, oidsToFetch)
	if err != nil {
		return nil, fmt.Errorf("find enum oid types: %w", err)
	}
	for _, enum := range enums {
		labels := make([]string, len(enum.Labels.Elements))
		if err := enum.Labels.AssignTo(&labels); err != nil {
			return nil, fmt.Errorf("assign labels to string slice for enum %s: %w", enum.TypeName.String, err)
		}
		orders := make([]float32, len(enum.Orders.Elements))
		if err := enum.Orders.AssignTo(&orders); err != nil {
			return nil, fmt.Errorf("assign orders to float32 slice for enum %s: %w", enum.TypeName.String, err)
		}
		childOIDUint32s := make([]uint32, len(enum.ChildOIDs.Elements))
		if err := enum.ChildOIDs.AssignTo(&childOIDUint32s); err != nil {
			return nil, fmt.Errorf("assign child OIDs to uint32 slice for enum %s: %w", enum.TypeName.String, err)
		}
		childOIDs := make([]pgtype.OID, len(enum.ChildOIDs.Elements))
		for i, oidUint32 := range childOIDUint32s {
			childOIDs[i] = pgtype.OID(oidUint32)
		}
		types[enum.OID] = EnumType{
			ID:        enum.OID,
			Name:      enum.TypeName.String,
			Labels:    labels,
			Orders:    orders,
			ChildOIDs: childOIDs,
		}
	}

	// Get composite types.
	composites, err := querier.FindCompositeTypes(ctx, oidsToFetch)
	if err != nil {
		return nil, fmt.Errorf("find composite types: %w", err)
	}
	for _, c := range composites {
		colTypes := make([]Type, len(c.ColOIDs.Elements))
		colNames := make([]string, len(c.ColOIDs.Elements))
		// Build each column of the composite type.
		for i, colOID := range c.ColOIDs.Elements {
			typeMapLock.Lock()
			colType, ok := typeMap[pgtype.OID(colOID.Int)]
			typeMapLock.Unlock()
			if !ok {
				// TODO: recursively resolve child types.
				return nil, fmt.Errorf("find type for composite column %s oid=%d",
					c.ColNames.Elements[i].String, c.ColOIDs.Elements[i].Int)
			}
			colTypes[i] = colType
			colNames[i] = c.ColNames.Elements[i].String
		}
		// Add overall composite type.
		types[c.TableTypeOID] = CompositeType{
			ID:          c.TableTypeOID,
			Name:        c.TableName.String,
			ColumnNames: colNames,
			ColumnTypes: colTypes,
		}
	}

	// Mark all remaining types as unknown. Let the code generator try to figure
	// a mapping.
	unknownOIDs := make([]uint32, 0, len(oidsToFetch))
	for _, oid := range oids {
		if _, ok := types[pgtype.OID(oid)]; !ok {
			unknownOIDs = append(unknownOIDs, oid)
		}
	}
	unknownOIDRows, err := querier.FindOIDNames(ctx, unknownOIDs)
	if err != nil {
		return nil, fmt.Errorf("find OID names for unknown OIDs: %w", err)
	}
	for _, row := range unknownOIDRows {
		types[row.OID] = UnknownType{
			ID:     row.OID,
			Name:   row.Name.String,
			PgKind: TypeKind(row.Kind.Int),
		}
	}

	// Update type cache.
	typeMapLock.Lock()
	for oid, typ := range types {
		typeMap[oid] = typ
	}
	typeMapLock.Unlock()

	return types, nil
}
