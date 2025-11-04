package engine

import (
	schemaextract "github.com/nsxbet/sql-schema"
	"github.com/nsxbet/sql-schema/diff"
)

// ExpressionComparer provides semantic comparison of database expressions.
// Different engines may have different expression syntaxes that are semantically equivalent.
type ExpressionComparer interface {
	// CompareExpressionsSemantically compares two expressions semantically.
	// Returns true if they are semantically equivalent, false otherwise.
	CompareExpressionsSemantically(expr1, expr2 string) (bool, error)

	// NormalizeExpression normalizes an expression to a standard form.
	// This removes engine-specific quirks like type casts, extra parentheses, etc.
	NormalizeExpression(expr string) string
}

// IndexComparer provides engine-specific index comparison logic.
type IndexComparer interface {
	// CompareIndexes compares two indexes and returns detailed differences.
	CompareIndexes(oldIndex, newIndex *schemaextract.Index) (*diff.IndexDiff, error)

	// CompareIndexWhereConditions compares WHERE clauses of partial indexes.
	CompareIndexWhereConditions(def1, def2 string) bool

	// ExtractWhereClauseFromIndexDef extracts the WHERE clause from an index definition.
	ExtractWhereClauseFromIndexDef(definition string) string
}

// FunctionComparer provides engine-specific function comparison logic.
type FunctionComparer interface {
	// Equal checks if two functions are identical.
	Equal(oldFunc, newFunc *schemaextract.Function) bool

	// CompareDetailed provides detailed comparison result with specific change types.
	CompareDetailed(oldFunc, newFunc *schemaextract.Function) (*diff.FunctionComparisonResult, error)

	// GetSignature extracts the function signature (name + parameters).
	GetSignature(function *schemaextract.Function) string

	// CanUseAlterFunction determines if ALTER FUNCTION can be used instead of DROP/CREATE.
	CanUseAlterFunction(oldFunc, newFunc *schemaextract.Function) bool
}

// ViewComparer provides engine-specific view comparison logic.
type ViewComparer interface {
	// CompareView compares two views and returns detailed differences.
	CompareView(oldView, newView *schemaextract.View) (*diff.ViewComparisonResult, error)

	// CompareMaterializedView compares two materialized views and returns detailed differences.
	CompareMaterializedView(oldMV, newMV *schemaextract.MaterializedView) (*diff.MaterializedViewComparisonResult, error)

	// RequiresRecreation determines if a view change requires DROP/CREATE instead of ALTER.
	RequiresRecreation(oldView, newView *schemaextract.View) bool
}

// ColumnComparer provides engine-specific column comparison logic.
type ColumnComparer interface {
	// CompareColumns compares two columns and returns detailed differences.
	CompareColumns(oldCol, newCol *schemaextract.Column) (*diff.ColumnDiff, error)

	// IsEquivalentType checks if two column types are semantically equivalent.
	// For example, in PostgreSQL: INTEGER = INT, CHARACTER VARYING = VARCHAR.
	IsEquivalentType(type1, type2 string) bool

	// NormalizeType normalizes a column type to a standard form.
	NormalizeType(colType string) string
}

// ProcedureComparer provides engine-specific stored procedure comparison logic.
type ProcedureComparer interface {
	// CompareProcedures compares two procedures and returns detailed differences.
	CompareProcedures(oldProc, newProc *schemaextract.Procedure) (*diff.ProcedureDiff, error)

	// CanUseAlterProcedure determines if ALTER PROCEDURE can be used.
	CanUseAlterProcedure(oldProc, newProc *schemaextract.Procedure) bool
}

// SequenceComparer provides engine-specific sequence comparison logic.
type SequenceComparer interface {
	// CompareSequences compares two sequences and returns detailed differences.
	CompareSequences(oldSeq, newSeq *schemaextract.Sequence) (*diff.SequenceDiff, error)
}

// Comparer is the main interface that aggregates all engine-specific comparers.
type Comparer interface {
	// GetEngine returns the engine type this comparer is for.
	GetEngine() Engine

	// Expression returns the expression comparer.
	Expression() ExpressionComparer

	// Index returns the index comparer.
	Index() IndexComparer

	// Function returns the function comparer.
	Function() FunctionComparer

	// View returns the view comparer.
	View() ViewComparer

	// Column returns the column comparer.
	Column() ColumnComparer

	// Procedure returns the procedure comparer.
	Procedure() ProcedureComparer

	// Sequence returns the sequence comparer.
	Sequence() SequenceComparer
}
