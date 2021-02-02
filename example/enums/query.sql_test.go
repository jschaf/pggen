package enums

import (
	"context"
	"github.com/jackc/pgtype"
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

	mac, _ := net.ParseMAC("00:00:5e:00:53:01")
	_, err := q.InsertDevice(ctx, pgtype.Macaddr{
		Addr:   mac,
		Status: pgtype.Present,
	}, Desktop)
	if err != nil {
		t.Fatal(err)
	}

	devices, err := q.FindAllDevices(ctx)
	if err != nil {
		t.Fatal(err)
	}
	assert.Equal(t, []FindAllDevicesRow{{
		Mac:  pgtype.Macaddr{Addr: mac, Status: pgtype.Present},
		Type: Desktop,
	}}, devices)
}
