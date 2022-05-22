package pg

import (
	"github.com/jackc/pgtype"
	"github.com/jschaf/pggen/internal/pg/pgoid"
	"strconv"
)

// Type is a Postgres type.
type Type interface {
	OID() pgtype.OID // pg_type.oid: row identifier
	String() string  // pg_type.typname: data type name
	Kind() TypeKind
}

// TypeKind is the pg_type.typtype column, describing the meta type of Type.
type TypeKind byte

const (
	KindBaseType        TypeKind = 'b' // includes array types
	KindCompositeType   TypeKind = 'c'
	KindDomainType      TypeKind = 'd'
	KindEnumType        TypeKind = 'e'
	KindPseudoType      TypeKind = 'p'
	KindRangeType       TypeKind = 'r'
	kindPlaceholderType TypeKind = '?' // pggen only, not part of postgres
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

	// VoidType is an empty type. A void type doesn't appear in output, but it's
	// necessary to scan rows.
	VoidType struct{}

	// ArrayType is an array type where pg_type.typelem != 0 and the name begins
	// with an underscore.
	ArrayType struct {
		ID pgtype.OID // pg_type.oid: row identifier
		// The name of the type, like _int4. Array types in Postgres typically
		// begin with an underscore. From pg_type.typname.
		Name string
		// pg_type.typelem: the element type of the array
		Elem Type
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
		// positions 1...n. But members added later might be given negative or
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

	// UnknownType is a Postgres type that's not a well-known type in
	// defaultKnownTypes, and not an enum, domain, or composite type. The code
	// generator might be able to resolve this type from a user-provided mapping
	// like --go-type my_int=int.
	UnknownType struct {
		ID     pgtype.OID // pg_type.oid: row identifier
		Name   string     // pg_type.typname: data type name
		PgKind TypeKind
	}

	// placeholderType is an internal, temporary type that we resolve in a second
	// pass. Useful because we resolve types sequentially by kind. For example, we
	// resolve all composite types before resolving array types. This approach
	// requires two passes for cases like when a composite type has a child type
	// that's an array.
	placeholderType struct {
		ID pgtype.OID // pg_type.oid: row identifier
	}
)

func (b BaseType) OID() pgtype.OID { return b.ID }
func (b BaseType) String() string  { return b.Name }
func (b BaseType) Kind() TypeKind  { return KindBaseType }

func (b VoidType) OID() pgtype.OID { return pgoid.Void }
func (b VoidType) String() string  { return "void" }
func (b VoidType) Kind() TypeKind  { return KindPseudoType }

func (b ArrayType) OID() pgtype.OID { return b.ID }
func (b ArrayType) String() string  { return b.Name }
func (b ArrayType) Kind() TypeKind  { return KindBaseType }

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

func (p placeholderType) OID() pgtype.OID { return p.ID }
func (p placeholderType) String() string  { return "placeholder-" + strconv.Itoa(int(p.ID)) }
func (p placeholderType) Kind() TypeKind  { return kindPlaceholderType }
