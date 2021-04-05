package pg

import (
	"context"
	"fmt"
	"github.com/jackc/pgtype"
	"github.com/jackc/pgx/v4"
	"time"
)

// TypeFetcher fetches Postgres types by the OID.
type TypeFetcher struct {
	cache   *typeCache
	querier *DBQuerier
}

func NewTypeFetcher(conn *pgx.Conn) *TypeFetcher {
	return &TypeFetcher{
		cache:   newTypeCache(),
		querier: NewQuerier(conn),
	}
}

// FindTypesByOIDs returns a map of a type OID to the Type description. The
// returned map contains every unique OID in oids (oids may contain duplicates)
// unless there's an error.
func (tf *TypeFetcher) FindTypesByOIDs(oids ...uint32) (map[pgtype.OID]Type, error) {
	if types, uncached := tf.cache.getOIDs(oids...); len(uncached) == 0 {
		return types, nil
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// First, recursively find all descendant OIDs from composite or array types.
	// Composite types are the only type that can be nested. Array types have an
	// element type that might be a composite type.
	descOIDs, err := tf.querier.FindDescendantOIDs(ctx, oids)
	if err != nil {
		return nil, fmt.Errorf("find descendant oids: %w", err)
	}
	allOIDs := make([]uint32, len(descOIDs))
	for i, d := range descOIDs {
		allOIDs[i] = uint32(d)
	}
	types, uncached := tf.cache.getOIDs(allOIDs...)

	enums, err := tf.findEnumTypes(ctx, uncached)
	if err != nil {
		return nil, fmt.Errorf("find enum types: %w", err)
	}
	for _, enum := range enums {
		types[enum.ID] = enum
		tf.cache.addType(enum)
		delete(uncached, enum.ID)
	}

	comps, err := tf.findCompositeTypes(ctx, uncached)
	if err != nil {
		return nil, fmt.Errorf("find composite types: %w", err)
	}
	for _, comp := range comps {
		types[comp.ID] = comp
		tf.cache.addType(comp)
		delete(uncached, comp.ID)
	}

	arrs, err := tf.findArrayTypes(ctx, uncached)
	if err != nil {
		return nil, fmt.Errorf("find array types: %w", err)
	}
	for _, arr := range arrs {
		types[arr.ID] = arr
		tf.cache.addType(arr)
		delete(uncached, arr.ID)
	}

	unknowns, err := tf.findUnknownTypes(ctx, uncached)
	if err != nil {
		return nil, fmt.Errorf("find unknown types: %w", err)
	}
	for _, unk := range unknowns {
		types[unk.ID] = unk
		tf.cache.addType(unk)
		delete(uncached, unk.ID)
	}

	if len(uncached) > 0 {
		return nil, fmt.Errorf("had %d unclassified types: %v", len(uncached), uncached)
	}
	return types, nil
}

func (tf *TypeFetcher) findEnumTypes(ctx context.Context, uncached map[pgtype.OID]struct{}) ([]EnumType, error) {
	oids := oidKeys(uncached)
	rows, err := tf.querier.FindEnumTypes(ctx, oids)
	if err != nil {
		return nil, fmt.Errorf("find enum oid types: %w", err)
	}
	types := make([]EnumType, len(rows))
	for i, enum := range rows {
		childOIDs := make([]pgtype.OID, len(enum.ChildOIDs))
		for i, oidUint32 := range enum.ChildOIDs {
			childOIDs[i] = pgtype.OID(oidUint32)
		}
		types[i] = EnumType{
			ID:        enum.OID,
			Name:      enum.TypeName,
			Labels:    enum.Labels,
			Orders:    enum.Orders,
			ChildOIDs: childOIDs,
		}
	}
	return types, nil
}

func (tf *TypeFetcher) findCompositeTypes(ctx context.Context, uncached map[pgtype.OID]struct{}) ([]CompositeType, error) {
	oids := oidKeys(uncached)
	rows, err := tf.querier.FindCompositeTypes(ctx, oids)
	if err != nil {
		return nil, fmt.Errorf("find composite types: %w", err)
	}
	// Record all composite types to fake a topological sort by repeated iteration.
	allComposites := make(map[pgtype.OID]struct{}, len(rows))
	for _, row := range rows {
		allComposites[row.TableTypeOID] = struct{}{}
	}

	types := make([]CompositeType, 0, len(rows))
	idx := -1
outer:
	for len(types) < len(rows) {
		idx = (idx + 1) % len(rows)
		row := rows[idx]

		// Check if we can resolve all columns for the composite type.
		for i, colOID := range row.ColOIDs {
			if _, isInCache := tf.cache.getOID(uint32(colOID)); !isInCache {
				if _, isInComposite := allComposites[pgtype.OID(colOID)]; !isInComposite {
					// We won't ever be able resolve this composite type.
					return nil, fmt.Errorf("find type for composite column %s oid=%d", row.ColNames[i], row.ColOIDs[i])
				}
				// We'll be able to resolve this after one of the for loop iteration
				// adds another composite to the cache.
				continue outer
			}
		}

		colTypes := make([]Type, len(row.ColOIDs))
		colNames := make([]string, len(row.ColOIDs))
		// Build each column of the composite type.
		for i, colOID := range row.ColOIDs {
			colType, ok := tf.cache.getOID(uint32(colOID))
			if !ok {
				return nil, fmt.Errorf("find type for composite column %s oid=%d", row.ColNames[i], row.ColOIDs[i])
			}
			colTypes[i] = colType
			colNames[i] = row.ColNames[i]
		}
		typ := CompositeType{
			ID:          row.TableTypeOID,
			Name:        row.TableName.String,
			ColumnNames: colNames,
			ColumnTypes: colTypes,
		}
		tf.cache.addType(typ)
		types = append(types, typ)
	}
	return types, nil
}

func (tf *TypeFetcher) findUnknownTypes(ctx context.Context, uncached map[pgtype.OID]struct{}) ([]UnknownType, error) {
	oids := oidKeys(uncached)
	rows, err := tf.querier.FindOIDNames(ctx, oids)
	if err != nil {
		return nil, fmt.Errorf("find OID names for unknown OIDs: %w", err)
	}
	types := make([]UnknownType, len(rows))
	for i, row := range rows {
		types[i] = UnknownType{
			ID:     row.OID,
			Name:   row.Name.String,
			PgKind: TypeKind(row.Kind.Int),
		}
	}
	return types, nil
}

func (tf *TypeFetcher) findArrayTypes(ctx context.Context, uncached map[pgtype.OID]struct{}) ([]ArrayType, error) {
	oids := oidKeys(uncached)
	rows, err := tf.querier.FindArrayTypes(ctx, oids)
	if err != nil {
		return nil, fmt.Errorf("find OID names for array OIDs: %w", err)
	}
	types := make([]ArrayType, len(rows))
	for i, row := range rows {
		elemType, ok := tf.cache.getOID(uint32(row.ElemOID))
		if !ok {
			return nil, fmt.Errorf("find type for array elem %s oid=%d", row.TypeName, row.OID)
		}
		types[i] = ArrayType{
			ID:       row.OID,
			Name:     row.TypeName,
			ElemType: elemType,
		}
	}
	return types, nil
}

func oidKeys(os map[pgtype.OID]struct{}) []uint32 {
	oids := make([]uint32, 0, len(os))
	for oid := range os {
		oids = append(oids, uint32(oid))
	}
	return oids
}
