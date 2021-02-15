package pgplan

import (
	"context"
	"fmt"
	"github.com/jackc/pgx/v4"
	"time"
)

// ExplainQuery executes an explain query and parses the plan.
func ExplainQuery(conn *pgx.Conn, sql string) (Node, error) {
	explainQuery := `EXPLAIN (VERBOSE, FORMAT JSON) ` + sql
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	row := conn.QueryRow(ctx, explainQuery)
	explain := make([]map[string]map[string]interface{}, 0, 1)
	if err := row.Scan(&explain); err != nil {
		return BadNode{}, fmt.Errorf("execute explain query: %w", err)
	}

	if len(explain) == 0 {
		return BadNode{}, fmt.Errorf("no explain output")
	}
	// TODO: when would there be multiple plans?
	plan, ok := explain[0]["Plan"]
	if !ok {
		return BadNode{}, fmt.Errorf("explain output had no 'Plan' node")
	}
	return ParseNode(plan)
}

func ParseNode(rawPlan map[string]interface{}) (Node, error) {
	kind, plan, err := parseBasePlan(rawPlan)
	if err != nil {
		return nil, fmt.Errorf("parse common fields of plan node: %w", err)
	}

	nodes, err := parseChildNodes(rawPlan)
	if err != nil {
		return nil, fmt.Errorf("parse append node: %w", err)
	}

	output, err := parseStringSlice(rawPlan, "Output")
	if err != nil {
		return BadNode{}, fmt.Errorf("no key \"Output\" for result")
	}

	switch kind {
	case KindBadNode:
		return BadNode{}, fmt.Errorf("got BadNode")
	case KindResult:
		return Result{Plan: plan, Output: output}, nil
	case KindProjectSet:
		return ProjectSet{Plan: plan, Output: output, Nodes: nodes}, nil
	case KindModifyTable:
		op, _ := parseString(rawPlan, "Operation")
		schema, _ := parseString(rawPlan, "Schema")
		relationName, _ := parseString(rawPlan, "Relation Name")
		alias, _ := parseString(rawPlan, "Alias")
		return ModifyTable{
			Operation:    Operation(op),
			Plan:         plan,
			RelationName: relationName,
			Schema:       schema,
			Alias:        alias,
			Output:       output,
			Nodes:        nodes,
		}, nil
	case KindAppend:
		return Append{Plan: plan, Nodes: nodes}, nil
	case KindMergeAppend:
		return MergeAppend{Plan: plan, Output: output, Nodes: nodes}, nil
	case KindRecursiveUnion:
		return RecursiveUnion{Plan: plan, Output: output, Nodes: nodes}, nil
	case KindBitmapAnd:
		return BitmapAnd{Plan: plan, Output: output, Nodes: nodes}, nil
	case KindBitmapOr:
		return BitmapOr{Plan: plan, Output: output, Nodes: nodes}, nil
	case KindScan:
		return Scan{Plan: plan, Output: output, Nodes: nodes}, nil
	case KindSeqScan:
		return SeqScan{Plan: plan, Output: output, Nodes: nodes}, nil
	case KindSampleScan:
		return SampleScan{Plan: plan, Output: output, Nodes: nodes}, nil
	case KindIndexScan:
		return IndexScan{Plan: plan, Output: output, Nodes: nodes}, nil
	case KindIndexOnlyScan:
		return IndexOnlyScan{Plan: plan, Output: output, Nodes: nodes}, nil
	case KindBitmapIndexScan:
		return BitmapIndexScan{Plan: plan, Output: output, Nodes: nodes}, nil
	case KindBitmapHeapScan:
		return BitmapHeapScan{Plan: plan, Output: output, Nodes: nodes}, nil
	case KindTidScan:
		return TidScan{Plan: plan, Output: output, Nodes: nodes}, nil
	case KindSubqueryScan:
		return SubqueryScan{Plan: plan, Output: output, Nodes: nodes}, nil
	case KindFunctionScan:
		return FunctionScan{Plan: plan, Output: output, Nodes: nodes}, nil
	case KindValuesScan:
		return ValuesScan{Plan: plan, Output: output, Nodes: nodes}, nil
	case KindTableFuncScan:
		return TableFuncScan{Plan: plan, Output: output, Nodes: nodes}, nil
	case KindCteScan:
		return CteScan{Plan: plan, Output: output, Nodes: nodes}, nil
	case KindNamedTuplestoreScan:
		return NamedTuplestoreScan{Plan: plan, Output: output, Nodes: nodes}, nil
	case KindWorkTableScan:
		return WorkTableScan{Plan: plan, Output: output, Nodes: nodes}, nil
	case KindForeignScan:
		return ForeignScan{Plan: plan, Output: output, Nodes: nodes}, nil
	case KindCustomScan:
		return CustomScan{Plan: plan, Output: output, Nodes: nodes}, nil
	case KindJoin:
		return Join{Plan: plan, Output: output, Nodes: nodes}, nil
	case KindNestLoop:
		return NestLoop{Plan: plan, Output: output, Nodes: nodes}, nil
	case KindMergeJoin:
		return MergeJoin{Plan: plan, Output: output, Nodes: nodes}, nil
	case KindHashJoin:
		return HashJoin{Plan: plan, Output: output, Nodes: nodes}, nil
	case KindMaterial:
		return Material{Plan: plan, Output: output, Nodes: nodes}, nil
	case KindSort:
		sortKey, _ := parseStringSlice(rawPlan, "Sort Key")
		return Sort{Plan: plan, Output: output, SortKey: sortKey, Nodes: nodes}, nil
	case KindIncrementalSort:
		return IncrementalSort{Plan: plan, Output: output, Nodes: nodes}, nil
	case KindGroup:
		return Group{Plan: plan, Output: output, Nodes: nodes}, nil
	case KindAgg:
		return Agg{Plan: plan, Output: output, Nodes: nodes}, nil
	case KindWindowAgg:
		return WindowAgg{Plan: plan, Output: output, Nodes: nodes}, nil
	case KindUnique:
		return Unique{Plan: plan, Output: output, Nodes: nodes}, nil
	case KindGather:
		return Gather{Plan: plan, Output: output, Nodes: nodes}, nil
	case KindGatherMerge:
		return GatherMerge{Plan: plan, Output: output, Nodes: nodes}, nil
	case KindHash:
		return Hash{Plan: plan, Output: output, Nodes: nodes}, nil
	case KindSetOp:
		return SetOp{Plan: plan, Output: output, Nodes: nodes}, nil
	case KindLockRows:
		return LockRows{Plan: plan, Output: output, Nodes: nodes}, nil
	case KindLimit:
		return Limit{Plan: plan, Output: output, Nodes: nodes}, nil
	default:
		return BadNode{}, fmt.Errorf("unhandled node kind: %s", kind)
	}
}

func parseChildNodes(plan map[string]interface{}) ([]Node, error) {
	rawPlans, ok := plan["Plans"]
	if !ok {
		return nil, nil
	}
	ps, ok := rawPlans.([]interface{})
	if !ok {
		return nil, fmt.Errorf("plans is not type []interface{}; got %T", rawPlans)
	}
	nodes := make([]Node, len(ps))
	for i, child := range ps {
		c, ok := child.(map[string]interface{})
		if !ok {
			return nil, fmt.Errorf("child plan is not type []map[string]interface{}; got %T", child)
		}
		node, err := ParseNode(c)
		if err != nil {
			return nil, fmt.Errorf("parse child plan: %w", err)
		}
		nodes[i] = node
	}
	return nodes, nil
}

// parseBasePlan parses the common plan fields of every node.
func parseBasePlan(plan map[string]interface{}) (NodeKind, Plan, error) {
	node, ok := plan["Node Type"]
	if !ok {
		return KindBadNode, Plan{}, fmt.Errorf("explain output had no 'Plan[Node Type]' node")
	}
	kind, ok := node.(string)
	if !ok {
		return KindBadNode, Plan{}, fmt.Errorf("explain output 'Plan[Node Type]' is not string; got type %T for value %v", node, node)
	}

	startupCost, _ := parseFloat64(plan, "Startup Cost")
	totalCost, _ := parseFloat64(plan, "Total Cost")
	planRows, _ := parseFloat64(plan, "Plan Rows")
	planWidth, _ := parseInt(plan, "Plan Width")
	parallelAware, _ := parseBool(plan, "Parallel Aware")
	parallelSafe, _ := parseBool(plan, "Parallel Safe")
	parentRel, _ := parseString(plan, "Parent Relationship")
	strategy, _ := parseString(plan, "Strategy")
	customPlanProvider, _ := parseString(plan, "Custom Plan Provider")

	return NodeKind(kind), Plan{
		StartupCost:        startupCost,
		TotalCost:          totalCost,
		PlanRows:           planRows,
		PlanWidth:          planWidth,
		ParallelAware:      parallelAware,
		ParallelSafe:       parallelSafe,
		Strategy:           Strategy(strategy),
		ParentRelationship: ParentRelationship(parentRel),
		CustomPlanProvider: customPlanProvider,
	}, nil
}

func parseInt(plan map[string]interface{}, key string) (int, bool) {
	if c, ok := plan[key]; ok {
		if n, ok := c.(int); ok {
			return n, true
		}
	}
	return 0, false
}

func parseFloat64(plan map[string]interface{}, key string) (float64, bool) {
	if c, ok := plan[key]; ok {
		if n, ok := c.(float64); ok {
			return n, true
		}
	}
	return 0, false
}

func parseBool(plan map[string]interface{}, key string) (bool, bool) {
	if c, ok := plan[key]; ok {
		if n, ok := c.(bool); ok {
			return n, true
		}
	}
	return false, false
}

func parseString(plan map[string]interface{}, key string) (string, bool) {
	if c, ok := plan[key]; ok {
		if n, ok := c.(string); ok {
			return n, true
		}
	}
	return "", false
}

func parseStringSlice(plan map[string]interface{}, key string) ([]string, error) {
	rawOuts, ok := plan[key]
	if !ok {
		return nil, nil
	}
	outs, ok := rawOuts.([]interface{})
	if !ok {
		return nil, fmt.Errorf("explain key %s is not []interface{}; got type %T", key, rawOuts)
	}
	strOuts := make([]string, len(outs))
	for i, out := range outs {
		out, ok := out.(string)
		if !ok {
			return nil, fmt.Errorf("explain key is not a string; got type %T for value %v", out, out)
		}
		strOuts[i] = out
	}
	return strOuts, nil
}
