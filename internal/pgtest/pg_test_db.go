package pgtest

import (
	"context"
	crand "crypto/rand"
	"encoding/binary"
	"github.com/jackc/pgx/v4"
	"io/ioutil"
	"math/rand"
	"strconv"
	"testing"
	"time"
)

func init() {
	var rngSeed int64
	_ = binary.Read(crand.Reader, binary.LittleEndian, &rngSeed)
	rand.Seed(rngSeed)
}

type CleanupFunc func()

// NewPostgresSchema opens a connection with search_path set to a randomly
// named, new schema.
func NewPostgresSchema(t *testing.T, sqlFiles []string) (*pgx.Conn, CleanupFunc) {
	t.Helper()
	// Create a new schema.
	connStr := "user=postgres password=hunter2 host=localhost port=5555 dbname=sqld"
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	conn, err := pgx.Connect(ctx, connStr)
	if err != nil {
		t.Fatalf("connect to docker postgres: %s", err)
	}
	schema := "sqld_test_" + strconv.Itoa(int(rand.Int31()))
	if _, err = conn.Exec(ctx, "CREATE SCHEMA "+schema); err != nil {
		t.Fatalf("create new schema: %s", err)
	}
	t.Logf("created schema: %s", schema)

	// Load SQL files into new schema.
	schemaConn, err := pgx.Connect(ctx, connStr+" search_path="+schema)
	if err != nil {
		t.Fatalf("connect to docker postgres with search path: %s", err)
	}
	for _, file := range sqlFiles {
		bs, err := ioutil.ReadFile(file)
		if err != nil {
			t.Fatalf("read test db sql file: %s", err)
		}
		if _, err := schemaConn.Exec(ctx, string(bs)); err != nil {
			t.Fatalf("run sql file %s: %s", file, err)
		}
	}

	cleanup := func() {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		if _, err := conn.Exec(ctx, "DROP SCHEMA "+schema+" CASCADE"); err != nil {
			t.Errorf("close conn: %s", err)
		}
		if err := conn.Close(ctx); err != nil {
			t.Errorf("close conn: %s", err)
		}
		if err = schemaConn.Close(ctx); err != nil {
			t.Errorf("close schema conn: %s", err)
		}
	}
	return schemaConn, cleanup
}
