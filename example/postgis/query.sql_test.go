package postgis

import (
	"context"
	"github.com/jackc/pgtype"
	"github.com/jschaf/pggen/internal/pgtest"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestQuerier(t *testing.T) {
	conn, cleanup := pgtest.NewPostgresSchema(t, []string{"schema.sql"})
	defer cleanup()

	q := NewQuerier(conn)
	ctx := context.Background()

	id := int32(4)
	pt := pgtype.Point{
		P:      pgtype.Vec2{X: 7, Y: 11},
		Status: pgtype.Present,
	}
	_, err := q.CreateVisit(ctx, id, pt)
	if err != nil {
		t.Fatal(err)
	}

	row, err := q.FindVisit(ctx, id)
	if err != nil {
		t.Fatal(err)
	}
	assert.Equal(t, FindVisitRow{
		VisitID: pgtype.Int4{Int: id, Status: pgtype.Present},
		Geo:     pt,
	}, row, "email should match")
}
