package pgtest

import (
	"context"
	"github.com/jackc/pgx/v4"
	"math/rand"
	"os"
	"strconv"
	"strings"
	"testing"
	"time"
)

// CleanupFunc deletes the schema and all database objects.
type CleanupFunc func()

type Option func(config *pgx.ConnConfig)

// NewPostgresSchemaString opens a connection with search_path set to a randomly
// named, new schema and loads the sql string.
func NewPostgresSchemaString(t *testing.T, sql string, opts ...Option) (*pgx.Conn, CleanupFunc) {
	t.Helper()
	// Create a new schema.
	connStr := "user=postgres password=hunter2 host=localhost port=5555 dbname=pggen"
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	conn, err := pgx.Connect(ctx, connStr)
	if err != nil {
		t.Fatalf("connect to docker postgres: %s", err)
	}
	schema := "pggen_test_" + strconv.Itoa(int(rand.Int31()))
	if _, err = conn.Exec(ctx, "CREATE SCHEMA "+schema); err != nil {
		t.Fatalf("create new schema: %s", err)
	}
	t.Logf("created schema: %s", schema)

	// Load SQL files into new schema.
	connStr += " search_path=" + schema
	connCfg, err := pgx.ParseConfig(connStr)
	if err != nil {
		t.Fatalf("parse config: %q: %s", connStr, err)
	}
	for _, opt := range opts {
		opt(connCfg)
	}
	schemaConn, err := pgx.ConnectConfig(ctx, connCfg)
	if err != nil {
		t.Fatalf("connect to docker postgres with search path: %s", err)
	}

	if _, err := schemaConn.Exec(ctx, sql); err != nil {
		t.Fatalf("run sql: %s", err)
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

// NewPostgresSchema opens a connection with search_path set to a randomly
// named, new schema and loads all sqlFiles.
func NewPostgresSchema(t *testing.T, sqlFiles []string, opts ...Option) (*pgx.Conn, CleanupFunc) {
	t.Helper()
	sb := &strings.Builder{}
	for _, file := range sqlFiles {
		bs, err := os.ReadFile(file)
		if err != nil {
			t.Fatalf("read test db sql file: %s", err)
		}
		sb.Write(bs)
		sb.WriteString(";\n\n -- FILE: ")
		sb.WriteString(file)
		sb.WriteString("\n")

	}
	return NewPostgresSchemaString(t, sb.String(), opts...)
}
