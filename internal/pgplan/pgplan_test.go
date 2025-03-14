package pgplan

import (
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	"github.com/jschaf/pggen/internal/pgtest"
	"github.com/jschaf/pggen/internal/texts"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseNode(t *testing.T) {
	tests := []struct {
		name string
		plan map[string]interface{}
		want Node
	}{
		{
			name: "Result - common fields",
			plan: map[string]interface{}{
				"Node Type":      "Result",
				"Startup Cost":   88.8,
				"Total Cost":     99.9,
				"Plan Rows":      55.5,
				"Plan Width":     44,
				"Parallel Aware": true,
				"Parallel Safe":  true,
			},
			want: Result{
				Plan: Plan{
					StartupCost:   88.8,
					TotalCost:     99.9,
					PlanRows:      55.5,
					PlanWidth:     44,
					ParallelAware: true,
					ParallelSafe:  true,
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ParseNode(tt.plan)
			if err != nil {
				t.Fatal(err)
			}
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestParseNode_DB(t *testing.T) {
	conn, cleanupFunc := pgtest.NewPostgresSchemaString(t, texts.Dedent(`
		CREATE TABLE author (
			author_id int PRIMARY KEY
		);
	`))
	defer cleanupFunc()
	tests := []struct {
		sql  string
		want Node
	}{
		{
			sql:  "SELECT 1 AS one",
			want: Result{Plan{Outs: []string{"1"}}},
		},
		{
			sql: "SELECT 1 AS num UNION ALL SELECT 2 AS num",
			want: Append{
				Plan{
					Nodes: []Node{
						Result{Plan{Outs: []string{"1"}}},
						Result{Plan{Outs: []string{"2"}}},
					},
				},
			},
		},
		{
			sql: "SELECT 1 AS num UNION SELECT 2 AS num",
			want: Unique{
				Plan: Plan{
					Outs: []string{"(1)"},
					Nodes: []Node{
						Sort{
							Plan: Plan{
								Outs: []string{"(1)"},
								Nodes: []Node{
									Append{
										Plan{Nodes: []Node{
											Result{Plan{Outs: []string{"1"}}},
											Result{Plan{Outs: []string{"2"}}},
										}},
									},
								},
							},
							SortKey: []string{"(1)"},
						},
					},
				},
			},
		},
		{
			sql: "INSERT INTO author (author_id) VALUES (1)",
			want: ModifyTable{
				Plan: Plan{
					Nodes: []Node{Result{Plan{Outs: []string{"1"}}}},
				},
				Operation:    OperationInsert,
				RelationName: "author",
				Alias:        "author",
			},
		},
		{
			sql: "INSERT INTO author (author_id) VALUES (1) RETURNING author_id",
			want: ModifyTable{
				Plan: Plan{
					Outs:  []string{"author.author_id"},
					Nodes: []Node{Result{Plan{Outs: []string{"1"}}}},
				},
				Operation:    OperationInsert,
				RelationName: "author",
				Alias:        "author",
			},
		},
		{
			sql: "SELECT generate_series(1,2)",
			want: ProjectSet{
				Plan{
					Outs:  []string{"generate_series(1, 2)"},
					Nodes: []Node{Result{}},
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.sql, func(t *testing.T) {
			got, err := ExplainQuery(conn, tt.sql)
			require.NoError(t, err)

			opts := cmp.Options{
				cmpopts.IgnoreFields(Plan{},
					"StartupCost", "TotalCost", "ParallelAware", "ParallelSafe",
					"PlanRows", "PlanWidth", "ParentRelationship",
				),
				cmpopts.IgnoreFields(ModifyTable{}, "Schema"),
			}
			if diff := cmp.Diff(tt.want, got, opts); diff != "" {
				t.Errorf("ExplainQuery() mismatch (-want +got):\n%s", diff)
			}
		})
	}
}
