package nested

import (
	"context"
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

	item := "item_name"
	sku := "sku_id"
	foo := 88
	want := []Qux{
		{
			InvItem: InventoryItem{ItemName: &item, Sku: Sku{SkuID: &sku}},
			Foo:     &foo,
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
