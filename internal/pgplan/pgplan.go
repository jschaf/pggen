package pgplan

import (
	"context"
	"fmt"
	"github.com/jackc/pgx/v5"
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
	switch kind {
	case KindBadNode:
		return BadNode{Plan: plan}, fmt.Errorf("got BadNode")
	case KindResult:
		return Result{Plan: plan}, nil
	case KindProjectSet:
		return ProjectSet{Plan: plan}, nil
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
		}, nil
	case KindAppend:
		return Append{Plan: plan}, nil
	case KindMergeAppend:
		return MergeAppend{Plan: plan}, nil
	case KindRecursiveUnion:
		return RecursiveUnion{Plan: plan}, nil
	case KindBitmapAnd:
		return BitmapAnd{Plan: plan}, nil
	case KindBitmapOr:
		return BitmapOr{Plan: plan}, nil
	case KindScan:
		return Scan{Plan: plan}, nil
	case KindSeqScan:
		return SeqScan{Plan: plan}, nil
	case KindSampleScan:
		return SampleScan{Plan: plan}, nil
	case KindIndexScan:
		return IndexScan{Plan: plan}, nil
	case KindIndexOnlyScan:
		return IndexOnlyScan{Plan: plan}, nil
	case KindBitmapIndexScan:
		return BitmapIndexScan{Plan: plan}, nil
	case KindBitmapHeapScan:
		return BitmapHeapScan{Plan: plan}, nil
	case KindTidScan:
		return TidScan{Plan: plan}, nil
	case KindSubqueryScan:
		return SubqueryScan{Plan: plan}, nil
	case KindFunctionScan:
		return FunctionScan{Plan: plan}, nil
	case KindValuesScan:
		return ValuesScan{Plan: plan}, nil
	case KindTableFuncScan:
		return TableFuncScan{Plan: plan}, nil
	case KindCteScan:
		return CteScan{Plan: plan}, nil
	case KindNamedTuplestoreScan:
		return NamedTuplestoreScan{Plan: plan}, nil
	case KindWorkTableScan:
		return WorkTableScan{Plan: plan}, nil
	case KindForeignScan:
		return ForeignScan{Plan: plan}, nil
	case KindCustomScan:
		return CustomScan{Plan: plan}, nil
	case KindJoin:
		return Join{Plan: plan}, nil
	case KindNestLoop:
		return NestLoop{Plan: plan}, nil
	case KindMergeJoin:
		return MergeJoin{Plan: plan}, nil
	case KindHashJoin:
		return HashJoin{Plan: plan}, nil
	case KindMaterial:
		return Material{Plan: plan}, nil
	case KindSort:
		sortKey, _ := parseStringSlice(rawPlan, "Sort Key")
		return Sort{Plan: plan, SortKey: sortKey}, nil
	case KindIncrementalSort:
		return IncrementalSort{Plan: plan}, nil
	case KindGroup:
		return Group{Plan: plan}, nil
	case KindAgg:
		return Agg{Plan: plan}, nil
	case KindWindowAgg:
		return WindowAgg{Plan: plan}, nil
	case KindUnique:
		return Unique{Plan: plan}, nil
	case KindGather:
		return Gather{Plan: plan}, nil
	case KindGatherMerge:
		return GatherMerge{Plan: plan}, nil
	case KindHash:
		return Hash{Plan: plan}, nil
	case KindSetOp:
		return SetOp{Plan: plan}, nil
	case KindLockRows:
		return LockRows{Plan: plan}, nil
	case KindLimit:
		return Limit{Plan: plan}, nil
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

	nodes, err := parseChildNodes(plan)
	if err != nil {
		return KindBadNode, Plan{}, fmt.Errorf("parse append node: %w", err)
	}

	output, err := parseStringSlice(plan, "Output")
	if err != nil {
		return KindBadNode, Plan{}, fmt.Errorf("no key \"Output\" for result")
	}

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
		Outs:               output,
		Nodes:              nodes,
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
