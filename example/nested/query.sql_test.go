package nested

import (
	"context"
	"github.com/jackc/pgtype"
	"github.com/jackc/pgx/v4"
	"github.com/jschaf/pggen/internal/pgtest"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestQuerier(t *testing.T) {
	conn, cleanup := pgtest.NewPostgresSchema(t, []string{"schema.sql"})
	defer cleanup()

	q := NewQuerier(conn)
	ctx := context.Background()

	want := []Qux{
		{
			InvItem: InventoryItem{
				ItemName: pgtype.Text{
					String: "item_name",
					Status: pgtype.Present,
				},
				Sku: Sku{SkuID: pgtype.Text{
					String: "sku_id",
					Status: pgtype.Present,
				}},
			},
			Foo: pgtype.Int8{
				Int:    88,
				Status: pgtype.Present,
			},
		},
	}
	{
		rows, err := q.Nested3(ctx)
		if err != nil {
			t.Fatal(err)
		}
		assert.Equal(t, want, rows)
	}

	{
		batch := &pgx.Batch{}
		q.Nested3Batch(batch)
		results := conn.SendBatch(ctx, batch)
		rows, err := q.Nested3Scan(results)
		if err != nil {
			t.Fatal(err)
		}
		assert.Equal(t, want, rows)
	}
}
