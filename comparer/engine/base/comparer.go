package defaultcomparer

import (
	"strings"

	schemaextract "github.com/nsxbet/sql-schema"
	"github.com/nsxbet/sql-schema/comparer/engine"
	"github.com/nsxbet/sql-schema/diff"
)

// BaseComparer provides default implementations for engine-specific comparers.
// Engines can embed this and override specific methods as needed.
type BaseComparer struct {
	engine engine.Engine
}

// NewBaseComparer creates a new base comparer.
func NewBaseComparer(eng engine.Engine) *BaseComparer {
	return &BaseComparer{
		engine: eng,
	}
}

// GetEngine returns the engine type.
func (c *BaseComparer) GetEngine() engine.Engine {
	return c.engine
}

// Expression returns a default expression comparer.
func (c *BaseComparer) Expression() engine.ExpressionComparer {
	return &BaseExpressionComparer{}
}

// Index returns a default index comparer.
func (c *BaseComparer) Index() engine.IndexComparer {
	return &BaseIndexComparer{}
}

// Function returns a default function comparer.
func (c *BaseComparer) Function() engine.FunctionComparer {
	return &BaseFunctionComparer{}
}

// View returns a default view comparer.
func (c *BaseComparer) View() engine.ViewComparer {
	return &BaseViewComparer{}
}

// Column returns a default column comparer.
func (c *BaseComparer) Column() engine.ColumnComparer {
	return &BaseColumnComparer{}
}

// Procedure returns a default procedure comparer.
func (c *BaseComparer) Procedure() engine.ProcedureComparer {
	return &BaseProcedureComparer{}
}

// Sequence returns a default sequence comparer.
func (c *BaseComparer) Sequence() engine.SequenceComparer {
	return &BaseSequenceComparer{}
}

// BaseExpressionComparer provides simple string-based expression comparison.
type BaseExpressionComparer struct{}

// CompareExpressionsSemantically performs simple string comparison after normalization.
func (c *BaseExpressionComparer) CompareExpressionsSemantically(expr1, expr2 string) (bool, error) {
	norm1 := c.NormalizeExpression(expr1)
	norm2 := c.NormalizeExpression(expr2)
	return norm1 == norm2, nil
}

// NormalizeExpression performs basic normalization: trim and lowercase.
func (c *BaseExpressionComparer) NormalizeExpression(expr string) string {
	return strings.TrimSpace(strings.ToLower(expr))
}

// BaseIndexComparer provides basic index comparison.
type BaseIndexComparer struct{}

// CompareIndexes compares two indexes.
func (c *BaseIndexComparer) CompareIndexes(oldIndex, newIndex *schemaextract.Index) (*diff.IndexDiff, error) {
	// Determine action based on existence
	var action diff.MetadataDiffAction
	if oldIndex == nil && newIndex != nil {
		action = diff.MetadataDiffActionCreate
	} else if oldIndex != nil && newIndex == nil {
		action = diff.MetadataDiffActionDrop
	} else if oldIndex != nil && newIndex != nil {
		// Check if indexes are different
		if !c.indexesEqual(oldIndex, newIndex) {
			action = diff.MetadataDiffActionAlter
		} else {
			// No change
			return nil, nil
		}
	}

	return &diff.IndexDiff{
		Action:   action,
		OldIndex: oldIndex,
		NewIndex: newIndex,
	}, nil
}

func (c *BaseIndexComparer) indexesEqual(idx1, idx2 *schemaextract.Index) bool {
	if idx1.Name != idx2.Name {
		return false
	}
	if idx1.Type != idx2.Type {
		return false
	}
	if idx1.Primary != idx2.Primary {
		return false
	}
	if idx1.Unique != idx2.Unique {
		return false
	}
	if len(idx1.Expressions) != len(idx2.Expressions) {
		return false
	}
	for i := range idx1.Expressions {
		if idx1.Expressions[i] != idx2.Expressions[i] {
			return false
		}
	}
	if idx1.WhereClause != idx2.WhereClause {
		return false
	}
	return true
}

// CompareIndexWhereConditions compares WHERE clauses.
func (c *BaseIndexComparer) CompareIndexWhereConditions(def1, def2 string) bool {
	return strings.TrimSpace(def1) == strings.TrimSpace(def2)
}

// ExtractWhereClauseFromIndexDef extracts WHERE clause from index definition.
func (c *BaseIndexComparer) ExtractWhereClauseFromIndexDef(definition string) string {
	// Simple extraction - look for WHERE keyword
	whereLower := strings.ToLower(definition)
	idx := strings.Index(whereLower, "where")
	if idx == -1 {
		return ""
	}
	return strings.TrimSpace(definition[idx+5:])
}

// BaseFunctionComparer provides basic function comparison.
type BaseFunctionComparer struct{}

// Equal checks if two functions are identical.
func (c *BaseFunctionComparer) Equal(oldFunc, newFunc *schemaextract.Function) bool {
	if oldFunc.Name != newFunc.Name {
		return false
	}
	return oldFunc.Definition == newFunc.Definition
}

// CompareDetailed provides detailed comparison.
func (c *BaseFunctionComparer) CompareDetailed(oldFunc, newFunc *schemaextract.Function) (*diff.FunctionComparisonResult, error) {
	result := &diff.FunctionComparisonResult{}

	if oldFunc.Definition != newFunc.Definition {
		result.BodyChanged = true
	}

	result.CanUseAlterFunction = c.CanUseAlterFunction(oldFunc, newFunc)

	return result, nil
}

// GetSignature returns the function name as signature.
func (c *BaseFunctionComparer) GetSignature(function *schemaextract.Function) string {
	return function.Name
}

// CanUseAlterFunction determines if ALTER can be used.
func (c *BaseFunctionComparer) CanUseAlterFunction(oldFunc, newFunc *schemaextract.Function) bool {
	// By default, assume DROP/CREATE is needed
	return false
}

// BaseViewComparer provides basic view comparison.
type BaseViewComparer struct{}

// CompareView compares two views.
func (c *BaseViewComparer) CompareView(oldView, newView *schemaextract.View) (*diff.ViewComparisonResult, error) {
	result := &diff.ViewComparisonResult{}

	if oldView.Definition != newView.Definition {
		result.DefinitionChanged = true
	}

	if oldView.Comment != newView.Comment {
		result.CommentChanged = true
	}

	result.RequiresRecreation = c.RequiresRecreation(oldView, newView)

	return result, nil
}

// CompareMaterializedView compares two materialized views.
func (c *BaseViewComparer) CompareMaterializedView(oldMV, newMV *schemaextract.MaterializedView) (*diff.MaterializedViewComparisonResult, error) {
	result := &diff.MaterializedViewComparisonResult{}

	if oldMV.Definition != newMV.Definition {
		result.DefinitionChanged = true
		// Materialized views usually require recreation for definition changes
		result.RequiresRecreation = true
	}

	if oldMV.Comment != newMV.Comment {
		result.CommentChanged = true
	}

	return result, nil
}

// RequiresRecreation determines if recreation is needed.
func (c *BaseViewComparer) RequiresRecreation(oldView, newView *schemaextract.View) bool {
	// By default, assume recreation is needed for definition changes
	return oldView.Definition != newView.Definition
}

// BaseColumnComparer provides basic column comparison.
type BaseColumnComparer struct{}

// CompareColumns compares two columns.
func (c *BaseColumnComparer) CompareColumns(oldCol, newCol *schemaextract.Column) (*diff.ColumnDiff, error) {
	var action diff.MetadataDiffAction
	if oldCol == nil && newCol != nil {
		action = diff.MetadataDiffActionCreate
	} else if oldCol != nil && newCol == nil {
		action = diff.MetadataDiffActionDrop
	} else if oldCol != nil && newCol != nil {
		if !c.columnsEqual(oldCol, newCol) {
			action = diff.MetadataDiffActionAlter
		} else {
			return nil, nil
		}
	}

	return &diff.ColumnDiff{
		Action:    action,
		OldColumn: oldCol,
		NewColumn: newCol,
	}, nil
}

func (c *BaseColumnComparer) columnsEqual(col1, col2 *schemaextract.Column) bool {
	if col1.Name != col2.Name {
		return false
	}
	if !c.IsEquivalentType(col1.Type, col2.Type) {
		return false
	}
	if col1.Nullable != col2.Nullable {
		return false
	}
	if col1.Default != col2.Default {
		return false
	}
	return true
}

// IsEquivalentType checks if two types are equivalent.
func (c *BaseColumnComparer) IsEquivalentType(type1, type2 string) bool {
	norm1 := c.NormalizeType(type1)
	norm2 := c.NormalizeType(type2)
	return norm1 == norm2
}

// NormalizeType normalizes a column type.
func (c *BaseColumnComparer) NormalizeType(colType string) string {
	return strings.ToUpper(strings.TrimSpace(colType))
}

// BaseProcedureComparer provides basic procedure comparison.
type BaseProcedureComparer struct{}

// CompareProcedures compares two procedures.
func (c *BaseProcedureComparer) CompareProcedures(oldProc, newProc *schemaextract.Procedure) (*diff.ProcedureDiff, error) {
	var action diff.MetadataDiffAction
	if oldProc == nil && newProc != nil {
		action = diff.MetadataDiffActionCreate
	} else if oldProc != nil && newProc == nil {
		action = diff.MetadataDiffActionDrop
	} else if oldProc != nil && newProc != nil {
		if oldProc.Definition != newProc.Definition {
			action = diff.MetadataDiffActionAlter
		} else {
			return nil, nil
		}
	}

	result := &diff.ProcedureDiff{
		Action:       action,
		OldProcedure: oldProc,
		NewProcedure: newProc,
	}

	if action == diff.MetadataDiffActionAlter {
		result.BodyChanged = true
		result.CanUseAlterProcedure = c.CanUseAlterProcedure(oldProc, newProc)
	}

	return result, nil
}

// CanUseAlterProcedure determines if ALTER can be used.
func (c *BaseProcedureComparer) CanUseAlterProcedure(oldProc, newProc *schemaextract.Procedure) bool {
	// By default, assume DROP/CREATE is needed
	return false
}

// BaseSequenceComparer provides basic sequence comparison.
type BaseSequenceComparer struct{}

// CompareSequences compares two sequences.
func (c *BaseSequenceComparer) CompareSequences(oldSeq, newSeq *schemaextract.Sequence) (*diff.SequenceDiff, error) {
	var action diff.MetadataDiffAction
	if oldSeq == nil && newSeq != nil {
		action = diff.MetadataDiffActionCreate
	} else if oldSeq != nil && newSeq == nil {
		action = diff.MetadataDiffActionDrop
	} else if oldSeq != nil && newSeq != nil {
		if !c.sequencesEqual(oldSeq, newSeq) {
			action = diff.MetadataDiffActionAlter
		} else {
			return nil, nil
		}
	}

	result := &diff.SequenceDiff{
		Action:      action,
		OldSequence: oldSeq,
		NewSequence: newSeq,
	}

	if action == diff.MetadataDiffActionAlter {
		c.populateSequenceChanges(result, oldSeq, newSeq)
	}

	return result, nil
}

func (c *BaseSequenceComparer) sequencesEqual(seq1, seq2 *schemaextract.Sequence) bool {
	return seq1.Start == seq2.Start &&
		seq1.Increment == seq2.Increment &&
		seq1.Min == seq2.Min &&
		seq1.Max == seq2.Max &&
		seq1.Cache == seq2.Cache &&
		seq1.Cycle == seq2.Cycle
}

func (c *BaseSequenceComparer) populateSequenceChanges(diff *diff.SequenceDiff, oldSeq, newSeq *schemaextract.Sequence) {
	if oldSeq.Start != newSeq.Start {
		diff.StartChanged = true
	}
	if oldSeq.Increment != newSeq.Increment {
		diff.IncrementChanged = true
	}
	if oldSeq.Min != newSeq.Min {
		diff.MinValueChanged = true
	}
	if oldSeq.Max != newSeq.Max {
		diff.MaxValueChanged = true
	}
	if oldSeq.Cache != newSeq.Cache {
		diff.CacheChanged = true
	}
	if oldSeq.Cycle != newSeq.Cycle {
		diff.CycleChanged = true
	}
}
