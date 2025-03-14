package pg

import (
	"sync"

	"github.com/jackc/pgtype"
)

// typeCache caches a map from a Postgres pg_type.oid to a Type.
type typeCache struct {
	types map[pgtype.OID]Type
	mu    *sync.Mutex
}

func newTypeCache() *typeCache {
	m := make(map[pgtype.OID]Type, len(defaultKnownTypes))
	for oid, typ := range defaultKnownTypes {
		m[oid] = typ
	}
	return &typeCache{
		types: m,
		mu:    &sync.Mutex{},
	}
}

// getOIDs returns the cached OIDS (with the type) and uncached OIDs.
func (tc *typeCache) getOIDs(oids ...uint32) (map[pgtype.OID]Type, map[pgtype.OID]struct{}) {
	cachedTypes := make(map[pgtype.OID]Type, len(oids))
	uncachedTypes := make(map[pgtype.OID]struct{}, len(oids))
	tc.mu.Lock()
	defer tc.mu.Unlock()
	for _, oid := range oids {
		if t, ok := tc.types[pgtype.OID(oid)]; ok {
			cachedTypes[pgtype.OID(oid)] = t
		} else {
			uncachedTypes[pgtype.OID(oid)] = struct{}{}
		}
	}
	return cachedTypes, uncachedTypes
}

func (tc *typeCache) getOID(oid uint32) (Type, bool) {
	tc.mu.Lock()
	typ, ok := tc.types[pgtype.OID(oid)]
	tc.mu.Unlock()
	return typ, ok
}

func (tc *typeCache) addType(typ Type) {
	tc.mu.Lock()
	tc.types[typ.OID()] = typ
	tc.mu.Unlock()
}
