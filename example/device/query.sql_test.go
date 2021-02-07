package device

import (
	"context"
	"github.com/jackc/pgtype"
	"github.com/jackc/pgx/v4"
	"github.com/jschaf/pggen/internal/pgtest"
	"github.com/stretchr/testify/assert"
	"net"
	"testing"
)

func TestQuerier(t *testing.T) {
	conn, cleanup := pgtest.NewPostgresSchema(t, []string{"schema.sql"})
	defer cleanup()
	q := NewQuerier(conn)
	ctx := context.Background()

	_, err := q.FindDevicesByUser(ctx, 8)
	assert.NoError(t, err)

	batch := &pgx.Batch{}
	q.FindDevicesByUserBatch(batch, 3)
	results := conn.SendBatch(ctx, batch)
	_, err = q.FindDevicesByUserScan(results)
	assert.NoError(t, err)
}

func TestQuerier_Composite(t *testing.T) {
	conn, cleanup := pgtest.NewPostgresSchema(t, []string{"schema.sql"})
	defer cleanup()
	q := NewQuerier(conn)
	ctx := context.Background()

	userID := 18
	_, err := q.InsertUser(ctx, userID, "foo")
	assert.NoError(t, err)

	mac, _ := net.ParseMAC("00:00:5e:00:53:01")
	_, err = q.InsertDevice(ctx, pgtype.Macaddr{Status: pgtype.Present, Addr: mac}, userID)
	assert.NoError(t, err)

	users, err := q.CompositeUser(ctx)
	if err != nil {
		t.Fatal(err)
	}
	assert.Equal(t, []CompositeUserRow{
		{
			Mac:  pgtype.Macaddr{Addr: mac, Status: pgtype.Present},
			Type: DeviceTypeUndefined,
			Row: User{
				ID:   pgtype.Int8{Int: int64(userID), Status: pgtype.Present},
				Name: pgtype.Text{String: "foo", Status: pgtype.Present},
			},
		},
	}, users)
}
