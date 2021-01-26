package pginfer

import (
	"context"
	"fmt"
	"github.com/jschaf/pggen/internal/ast"
)

// PlanType is the top-level node plan type that Postgres plans for executing
// query. https://www.postgresql.org/docs/13/executor.html
type PlanType string

const (
	PlanResult      PlanType = "Result"      // select statement
	PlanModifyTable PlanType = "ModifyTable" // update, insert, or delete statement
)

// explainQuery executes explain plan to get the node plan type and the format
// of the output columns.
func (inf *Inferrer) explainQuery(query *ast.SourceQuery) (PlanType, []string, error) {
	explainQuery := `EXPLAIN (VERBOSE, FORMAT JSON) ` + query.PreparedSQL
	ctx, cancel := context.WithTimeout(context.Background(), defaultTimeout)
	defer cancel()
	row := inf.conn.QueryRow(ctx, explainQuery, createParamArgs(query)...)
	explain := make([]map[string]map[string]interface{}, 0, 1)
	if err := row.Scan(&explain); err != nil {
		return "", nil, fmt.Errorf("explain prepared query: %w", err)
	}
	if len(explain) == 0 {
		return "", nil, fmt.Errorf("no explain output")
	}
	plan, ok := explain[0]["Plan"]
	if !ok {
		return "", nil, fmt.Errorf("explain output had no 'Plan' node")
	}
	node, ok := plan["Node Type"]
	if !ok {
		return "", nil, fmt.Errorf("explain output had no 'Plan[Node Type]' node")
	}
	strNode, ok := node.(string)
	if !ok {
		return "", nil, fmt.Errorf("explain output 'Plan[Node Type]' is not string; got type %T for value %v", node, node)
	}
	rawOuts, ok := plan["Output"]
	if !ok {
		return "", nil, fmt.Errorf("explain output had no 'Plan.Output' node")
	}
	outs, ok := rawOuts.([]interface{})
	if !ok {
		return "", nil, fmt.Errorf("explain output 'Plan.Output' is not []interface{}; got type %T for value %v", outs, outs)
	}
	strOuts := make([]string, len(outs))
	for i, out := range outs {
		out, ok := out.(string)
		if !ok {
			return "", nil, fmt.Errorf("explain output 'Plan.Output[%d]' was not a string; got type %T for value %v", i, out, out)
		}
		strOuts[i] = out
	}
	return PlanType(strNode), strOuts, nil
}
