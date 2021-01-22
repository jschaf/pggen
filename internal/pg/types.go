package pg

import (
	"fmt"
	"github.com/jackc/pgtype"
	"github.com/jackc/pgx/v4"
	"sync"
)

// TypeKind is b for a base type, c for a composite type (e.g., a table's row type), d for a domain, e for an enum type,
// p for a pseudo-type, or r for a range type. See also typrelid and typbasetype.
type TypeKind byte

type OID = uint32

const (
	KindBaseType      TypeKind = 'b'
	KindCompositeType TypeKind = 'c' // table row type
	KindDomainType    TypeKind = 'd'
	KindEnumType      TypeKind = 'e'
	KindPseudoType    TypeKind = 'p'
	KindRangeType     TypeKind = 'r'
)

type Type interface {
	fmt.Stringer
	OID() OID
	pgType()
}

type (
	// https://www.postgresql.org/docs/13/catalog-pg-type.html
	BaseType struct {
		ID         OID            // pg_type.oid: row identifier
		Name       string         // pg_type.typname: data type name
		Kind       TypeKind       // pg_type.typtype: the kind of type
		Composite  *CompositeType // pg_type.typrelid: composite type only, the pg_class for the type
		Dimensions int            // pg_type.typndims: domains on array type only 0 otherwise, number of array dimensions,
	}

	DomainType struct {
		ID         OID      // pg_type.oid: row identifier
		Name       string   // pg_type.typname: data type name
		IsNotNull  bool     // pg_type.typnotnull: domains only, not null constraint for domains
		HasDefault bool     // pg_type.typdefault: domains only, if there's a default value
		BaseType   BaseType // pg_type.typbasetype: domains only, the base type
	}

	// Composite types are represented as a class.
	// https://www.postgresql.org/docs/13/catalog-pg-class.html
	CompositeType struct {
		ID       uint32 // pg_class.oid: row identifier
		TypeName string // pg_class.relname: name of the composite type
		Columns  []Type // pg_attribute: information about columns of the composite type
	}
)

func (b BaseType) String() string { return b.Name }
func (b BaseType) OID() OID       { return b.ID }
func (b BaseType) pgType()        {}

func (c CompositeType) OID() OID { return c.ID }
func (c CompositeType) pgType()  {}

var (
	Bool = BaseType{ID: pgtype.BoolOID, Name: "bool", Kind: KindBaseType}
	Text = BaseType{ID: pgtype.TextOID, Name: "text", Kind: KindBaseType}
	Int4 = BaseType{ID: pgtype.Int4OID, Name: "integer", Kind: KindBaseType}
)

var (
	typeMapLock = &sync.Mutex{}

	typeMap = map[OID]Type{
		pgtype.BoolOID: Bool,
		pgtype.TextOID: Text,
		pgtype.Int4OID: Int4,
	}
)

func FindOIDType(oid OID) (Type, bool) {
	typeMapLock.Lock()
	defer typeMapLock.Unlock()
	t, ok := typeMap[oid]
	return t, ok
}

func FetchOIDTypes(conn *pgx.Conn, oids ...OID) (map[OID]Type, error) {
	types := make(map[OID]Type, len(oids))
	oidsToFetch := make([]OID, 0, len(oids))
	typeMapLock.Lock()
	for _, oid := range oids {
		if t, ok := typeMap[oid]; ok {
			types[oid] = t
		}
		oidsToFetch = append(oidsToFetch, oid)
	}
	typeMapLock.Unlock()

	// TODO: fetch from database

	if len(oids) > len(types) {
		var missing OID
		for _, oid := range oids {
			if _, ok := types[oid]; !ok {
				missing = oid
				break
			}
		}
		return nil, fmt.Errorf("did not find all OIDs; missing %d", missing)
	}

	return types, nil
}
