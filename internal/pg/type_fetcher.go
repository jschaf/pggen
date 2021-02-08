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
		types[i] = EnumType{
			ID:        enum.OID,
			Name:      enum.TypeName.String,
			Labels:    labels,
			Orders:    orders,
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
	types := make([]CompositeType, len(rows))
	for i, c := range rows {
		colTypes := make([]Type, len(c.ColOIDs.Elements))
		colNames := make([]string, len(c.ColOIDs.Elements))
		// Build each column of the composite type.
		for i, colOID := range c.ColOIDs.Elements {
			colType, ok := tf.cache.getOID(uint32(colOID.Int))
			if !ok {
				// TODO: recursively resolve child types.
				return nil, fmt.Errorf("find type for composite column %s oid=%d",
					c.ColNames.Elements[i].String, c.ColOIDs.Elements[i].Int)
			}
			colTypes[i] = colType
			colNames[i] = c.ColNames.Elements[i].String
		}
		types[i] = CompositeType{
			ID:          c.TableTypeOID,
			Name:        c.TableName.String,
			ColumnNames: colNames,
			ColumnTypes: colTypes,
		}
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

func oidKeys(os map[pgtype.OID]struct{}) []uint32 {
	oids := make([]uint32, 0, len(os))
	for oid := range os {
		oids = append(oids, uint32(oid))
	}
	return oids
}
