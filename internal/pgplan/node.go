package pgplan

// Node is the super-type of all Postgres plan nodes.
// https://doxygen.postgresql.org/nodes_8h.html#a83ba1e84fa23f6619c3d29036b160919
type Node interface {
	Kind() NodeKind
}

// NodeKind is the top-level node plan type that Postgres plans for executing
// query. https://www.postgresql.org/docs/13/executor.html
type NodeKind string

//goland:noinspection GoUnusedConst
const (
	KindBadNode             NodeKind = "BadPlan"
	KindResult              NodeKind = "Result"
	KindProjectSet          NodeKind = "ProjectSet"
	KindModifyTable         NodeKind = "ModifyTable"
	KindAppend              NodeKind = "Append"
	KindMergeAppend         NodeKind = "MergeAppend"
	KindRecursiveUnion      NodeKind = "RecursiveUnion"
	KindBitmapAnd           NodeKind = "BitmapAnd"
	KindBitmapOr            NodeKind = "BitmapOr"
	KindScan                NodeKind = "Scan"
	KindSeqScan             NodeKind = "SeqScan"
	KindSampleScan          NodeKind = "SampleScan"
	KindIndexScan           NodeKind = "IndexScan"
	KindIndexOnlyScan       NodeKind = "IndexOnlyScan"
	KindBitmapIndexScan     NodeKind = "BitmapIndexScan"
	KindBitmapHeapScan      NodeKind = "BitmapHeapScan"
	KindTidScan             NodeKind = "TidScan"
	KindSubqueryScan        NodeKind = "SubqueryScan"
	KindFunctionScan        NodeKind = "FunctionScan"
	KindValuesScan          NodeKind = "ValuesScan"
	KindTableFuncScan       NodeKind = "TableFuncScan"
	KindCteScan             NodeKind = "CteScan"
	KindNamedTuplestoreScan NodeKind = "NamedTuplestoreScan"
	KindWorkTableScan       NodeKind = "WorkTableScan"
	KindForeignScan         NodeKind = "ForeignScan"
	KindCustomScan          NodeKind = "CustomScan"
	KindJoin                NodeKind = "Join"
	KindNestLoop            NodeKind = "NestLoop"
	KindMergeJoin           NodeKind = "MergeJoin"
	KindHashJoin            NodeKind = "HashJoin"
	KindMaterial            NodeKind = "Material"
	KindSort                NodeKind = "Sort"
	KindIncrementalSort     NodeKind = "IncrementalSort"
	KindGroup               NodeKind = "Group"
	KindAgg                 NodeKind = "Agg"
	KindWindowAgg           NodeKind = "WindowAgg"
	KindUnique              NodeKind = "Unique"
	KindGather              NodeKind = "Gather"
	KindGatherMerge         NodeKind = "GatherMerge"
	KindHash                NodeKind = "Hash"
	KindSetOp               NodeKind = "SetOp"
	KindLockRows            NodeKind = "LockRows"
	KindLimit               NodeKind = "Limit"
)

// ParentRelationship describes why this operation needs to be run in order to
// facilitate the parent operation.
type ParentRelationship string

//goland:noinspection GoUnusedConst
const (
	// Means this node is a top-level node. All nodes with a parent have set
	// relationship that is not none.
	ParentRelationshipNone ParentRelationship = ""
	// Most common: means take in the rows from this operation as input, process
	// them and pass them on.
	ParentRelationshipOuter ParentRelationship = "Outer"
	// Only (but always) on second child of join operations. Means a node is the
	// inner part of a loop.
	ParentRelationshipInner ParentRelationship = "Inner"
	// Member is for all children of Append and ModifyTable nodes.
	ParentRelationshipMember ParentRelationship = "Member"
	// InitPlan is calculations performed before query starts executing.
	ParentRelationshipInitPlan ParentRelationship = "InitPlan"
	// Subquery means the node is a subquery of a parent node. Since Postgres
	// always uses subquery scans to feed subquery data to parent queries, only
	// ever appears on the children of subquery scans.
	ParentRelationshipSubquery ParentRelationship = "Subquery"
	// Like a Subquery, represents a new query, but used when a subquery scan is
	// not necessary.
	ParentRelationshipSubPlan ParentRelationship = "SubPlan"
)

// Strategy determines overall execution strategies for Agg plan nodes and SetOp
// nodes.
// https://sourcegraph.com/github.com/postgres/postgres@8facf1ea00b7a0c08c755a0392212b83e04ae28a/-/blob/src/include/nodes/nodes.h?subtree=true#L759:14
type Strategy string

//goland:noinspection GoUnusedConst
const (
	// Simple agg across all input rows.
	StrategyPlain Strategy = "Plain"
	// For grouped agg and SetOp, input must be sorted.
	StrategySorted Strategy = "Sorted"
	// For grouped agg and SetOp, uses internal hashtable.
	StrategyHashed Strategy = "Hashed"
	// Grouped agg, hash and sort both used.
	StrategyMixed Strategy = "Mixed"
	// For unknown aggregates.
	StrategyUnknown Strategy = "???"
)

// Operation for a ModifyTable node.
type Operation string

//goland:noinspection GoUnusedConst
const (
	OperationInsert Operation = "Insert"
	OperationUpdate Operation = "Update"
	OperationDelete Operation = "Delete"
)

// All plan nodes "derive" from the Plan structure by having the
// Plan structure as the first field. This ensures that everything works
// when nodes are cast to Plan's. (node pointers are frequently cast to Plan*
// when passed around generically in the executor)
// https://sourcegraph.com/github.com/postgres/postgres@8facf1ea00b7a0c08c755a0392212b83e04ae28a/-/blob/src/include/nodes/plannodes.h#L110:16
type Plan struct {
	// Estimated execution costs for plan (see costsize.c for more info).
	StartupCost float64 // cost expended before fetching any tuples
	TotalCost   float64 // total cost (assuming all tuples fetched)

	// Planner's estimate of result size of this plan step.
	PlanRows  float64 // number of rows plan is expected to emit
	PlanWidth int     // average row width in bytes

	// Information needed for parallel query.
	ParallelAware bool // engage parallel-aware logic?
	ParallelSafe  bool // OK to use as part of parallel plan?

	// Relationship from this node to its parent. Always set for descendant nodes.
	ParentRelationship ParentRelationship

	// How to execute a node. Used for Agg and SetOp nodes.
	Strategy Strategy

	// Custom plan, if any.
	CustomPlanProvider string
}

type (
	// BadNode is returned whenever a plan is not parseable.
	BadNode struct{}

	// If no outer plan, evaluate a variable-free targetlist.
	// If outer plan, return tuples from outer plan (after a level of
	// projection as shown by targetlist).
	// https://sourcegraph.com/github.com/postgres/postgres@8facf1ea00b7a0c08c755a0392212b83e04ae28a/-/blob/src/include/nodes/plannodes.h#L180:1
	Result struct {
		Plan
		// The column expressions (target list).
		Output []string
	}

	// Generate the concatenation of the results of sub-plans.
	// Combine the results of the child operations. This can be the result of an
	// explicit UNION ALL statement, or the need for a parent operation to
	// consume the results of two or more children together.
	// https://www.pgmustard.com/docs/explain/append
	Append struct {
		Plan
		Nodes []Node
	}

	// ProjectSet appears when the SELECT or ORDER BY clause of the query. They
	// basically just execute the set-returning function(s) for each tuple until
	// none of the functions return any more records.
	// https://www.postgresql.org/message-id/CAKJS1f9pWUwxaD%2B0kxOOUuwaBcpGQtCKi3DKE8ob_uHN-JTJhw%40mail.gmail.com
	ProjectSet struct {
		Plan
		Output []string
		Nodes  []Node
	}

	// Apply rows produced by subplan(s) to result table(s), by inserting,
	// updating, or deleting.
	ModifyTable struct {
		Plan
		Operation    Operation
		RelationName string
		Schema       string
		Alias        string
		Output       []string
		Nodes        []Node
	}

	// Combines the sorted results of the child operations, in a way that
	// preserves their sort order.
	// Can be used for combining already-sorted rows from table partitions.
	// https://www.pgmustard.com/docs/explain/merge-append
	MergeAppend struct {
		Plan
		SortKey []string
		Output  []string
		Nodes   []Node
	}

	RecursiveUnion struct {
		Plan
		Output []string
		Nodes  []Node
	}

	BitmapAnd struct {
		Plan
		Output []string
		Nodes  []Node
	}

	BitmapOr struct {
		Plan
		Output []string
		Nodes  []Node
	}

	Scan struct {
		Plan
		Output []string
		Nodes  []Node
	}

	SeqScan struct {
		Plan
		Output []string
		Nodes  []Node
	}

	SampleScan struct {
		Plan
		Output []string
		Nodes  []Node
	}

	IndexScan struct {
		Plan
		Output []string
		Nodes  []Node
	}

	IndexOnlyScan struct {
		Plan
		Output []string
		Nodes  []Node
	}

	BitmapIndexScan struct {
		Plan
		Output []string
		Nodes  []Node
	}

	BitmapHeapScan struct {
		Plan
		Output []string
		Nodes  []Node
	}

	TidScan struct {
		Plan
		Output []string
		Nodes  []Node
	}

	SubqueryScan struct {
		Plan
		Output []string
		Nodes  []Node
	}

	FunctionScan struct {
		Plan
		Output []string
		Nodes  []Node
	}

	ValuesScan struct {
		Plan
		Output []string
		Nodes  []Node
	}

	TableFuncScan struct {
		Plan
		Output []string
		Nodes  []Node
	}

	CteScan struct {
		Plan
		Output []string
		Nodes  []Node
	}

	NamedTuplestoreScan struct {
		Plan
		Output []string
		Nodes  []Node
	}

	WorkTableScan struct {
		Plan
		Output []string
		Nodes  []Node
	}

	ForeignScan struct {
		Plan
		Output []string
		Nodes  []Node
	}

	CustomScan struct {
		Plan
		Output []string
		Nodes  []Node
	}

	Join struct {
		Plan
		Output []string
		Nodes  []Node
	}

	NestLoop struct {
		Plan
		Output []string
		Nodes  []Node
	}

	MergeJoin struct {
		Plan
		Output []string
		Nodes  []Node
	}

	HashJoin struct {
		Plan
		Output []string
		Nodes  []Node
	}

	Material struct {
		Plan
		Output []string
		Nodes  []Node
	}

	Sort struct {
		Plan
		Output  []string
		SortKey []string
		Nodes   []Node
	}

	IncrementalSort struct {
		Plan
		Output []string
		Nodes  []Node
	}

	Group struct {
		Plan
		Output []string
		Nodes  []Node
	}

	Agg struct {
		Plan
		Output []string
		Nodes  []Node
	}

	WindowAgg struct {
		Plan
		Output []string
		Nodes  []Node
	}

	// Unique is a very simple node type that just filters out duplicate tuples
	// from a stream of sorted tuples from its subplan.
	// https://sourcegraph.com/github.com/postgres/postgres@8facf1ea00b7a0c08c755a0392212b83e04ae28a/-/blob/src/include/nodes/plannodes.h?subtree=true#L864:16
	Unique struct {
		Plan
		Nodes  []Node
		Output []string
	}

	Gather struct {
		Plan
		Output []string
		Nodes  []Node
	}

	GatherMerge struct {
		Plan
		Output []string
		Nodes  []Node
	}

	Hash struct {
		Plan
		Output []string
		Nodes  []Node
	}

	SetOp struct {
		Plan
		Output []string
		Nodes  []Node
	}

	LockRows struct {
		Plan
		Output []string
		Nodes  []Node
	}

	Limit struct {
		Plan
		Output []string
		Nodes  []Node
	}
)

func (BadNode) Kind() NodeKind             { return KindBadNode }
func (Result) Kind() NodeKind              { return KindResult }
func (ProjectSet) Kind() NodeKind          { return KindProjectSet }
func (ModifyTable) Kind() NodeKind         { return KindModifyTable }
func (Append) Kind() NodeKind              { return KindAppend }
func (MergeAppend) Kind() NodeKind         { return KindMergeAppend }
func (RecursiveUnion) Kind() NodeKind      { return KindRecursiveUnion }
func (BitmapAnd) Kind() NodeKind           { return KindBitmapAnd }
func (BitmapOr) Kind() NodeKind            { return KindBitmapOr }
func (Scan) Kind() NodeKind                { return KindScan }
func (SeqScan) Kind() NodeKind             { return KindSeqScan }
func (SampleScan) Kind() NodeKind          { return KindSampleScan }
func (IndexScan) Kind() NodeKind           { return KindIndexScan }
func (IndexOnlyScan) Kind() NodeKind       { return KindIndexOnlyScan }
func (BitmapIndexScan) Kind() NodeKind     { return KindBitmapIndexScan }
func (BitmapHeapScan) Kind() NodeKind      { return KindBitmapHeapScan }
func (TidScan) Kind() NodeKind             { return KindTidScan }
func (SubqueryScan) Kind() NodeKind        { return KindSubqueryScan }
func (FunctionScan) Kind() NodeKind        { return KindFunctionScan }
func (ValuesScan) Kind() NodeKind          { return KindValuesScan }
func (TableFuncScan) Kind() NodeKind       { return KindTableFuncScan }
func (CteScan) Kind() NodeKind             { return KindCteScan }
func (NamedTuplestoreScan) Kind() NodeKind { return KindNamedTuplestoreScan }
func (WorkTableScan) Kind() NodeKind       { return KindWorkTableScan }
func (ForeignScan) Kind() NodeKind         { return KindForeignScan }
func (CustomScan) Kind() NodeKind          { return KindCustomScan }
func (Join) Kind() NodeKind                { return KindJoin }
func (NestLoop) Kind() NodeKind            { return KindNestLoop }
func (MergeJoin) Kind() NodeKind           { return KindMergeJoin }
func (HashJoin) Kind() NodeKind            { return KindHashJoin }
func (Material) Kind() NodeKind            { return KindMaterial }
func (Sort) Kind() NodeKind                { return KindSort }
func (IncrementalSort) Kind() NodeKind     { return KindIncrementalSort }
func (Group) Kind() NodeKind               { return KindGroup }
func (Agg) Kind() NodeKind                 { return KindAgg }
func (WindowAgg) Kind() NodeKind           { return KindWindowAgg }
func (Unique) Kind() NodeKind              { return KindUnique }
func (Gather) Kind() NodeKind              { return KindGather }
func (GatherMerge) Kind() NodeKind         { return KindGatherMerge }
func (Hash) Kind() NodeKind                { return KindHash }
func (SetOp) Kind() NodeKind               { return KindSetOp }
func (LockRows) Kind() NodeKind            { return KindLockRows }
func (Limit) Kind() NodeKind               { return KindLimit }
