package pg

import (
	"github.com/jackc/pgx/v5/pgtype"
	"sync"
)

// typeCache caches a map from a Postgres pg_type.oid to a Type.
type typeCache struct {
	types map[uint32]Type
	mu    *sync.Mutex
}

func newTypeCache() *typeCache {
	m := make(map[uint32]Type, len(defaultKnownTypes))
	for oid, typ := range defaultKnownTypes {
		m[oid] = typ
	}
	return &typeCache{
		types: m,
		mu:    &sync.Mutex{},
	}
}

func lookup() {
	!!!fixme
	m := pgtype.NewMap() // !!!
	m.TypeForValue()     // looks up pg type for go type
	m.FormatCodeForOID()
	m.TypeForOID()
	m.TypeForName()

}

// getOIDs returns the cached OIDS (with the type) and uncached OIDs.
func (tc *typeCache) getOIDs(oids ...uint32) (map[uint32]Type, map[uint32]struct{}) {
	cachedTypes := make(map[uint32]Type, len(oids))
	uncachedTypes := make(map[uint32]struct{}, len(oids))
	tc.mu.Lock()
	defer tc.mu.Unlock()
	for _, oid := range oids {
		if t, ok := tc.types[oid]; ok {
			cachedTypes[oid] = t
		} else {
			uncachedTypes[oid] = struct{}{}
		}
	}
	return cachedTypes, uncachedTypes
}

func (tc *typeCache) getOID(oid uint32) (Type, bool) {
	tc.mu.Lock()
	typ, ok := tc.types[oid]
	tc.mu.Unlock()
	return typ, ok
}

func (tc *typeCache) addType(typ Type) {
	tc.mu.Lock()
	tc.types[typ.OID()] = typ
	tc.mu.Unlock()
}
