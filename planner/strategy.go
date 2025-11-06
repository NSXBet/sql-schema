package planner

import (
	"fmt"
	"slices"

	"github.com/nsxbet/sql-schema/comparer/engine"
	"github.com/nsxbet/sql-schema/diff"
	"github.com/pkg/errors"
)

// RiskLevel represents the risk level of a migration operation.
type RiskLevel string

const (
	RiskLevelNone   RiskLevel = "NONE"   // No risk, fully reversible
	RiskLevelLow    RiskLevel = "LOW"    // Low risk, minor impact
	RiskLevelMedium RiskLevel = "MEDIUM" // Medium risk, requires attention
	RiskLevelHigh   RiskLevel = "HIGH"   // High risk, potential data loss
)

// OperationType categorizes the type of migration operation.
type OperationType string

const (
	OperationTypeSchema     OperationType = "SCHEMA"
	OperationTypeTable      OperationType = "TABLE"
	OperationTypeColumn     OperationType = "COLUMN"
	OperationTypeIndex      OperationType = "INDEX"
	OperationTypeConstraint OperationType = "CONSTRAINT"
	OperationTypeView       OperationType = "VIEW"
	OperationTypeFunction   OperationType = "FUNCTION"
	OperationTypeProcedure  OperationType = "PROCEDURE"
	OperationTypeSequence   OperationType = "SEQUENCE"
	OperationTypeEnum       OperationType = "ENUM"
	OperationTypeEvent      OperationType = "EVENT"
	OperationTypeExtension  OperationType = "EXTENSION"
)

// MigrationOperation represents a single DDL operation in the migration.
type MigrationOperation struct {
	ID           string                  // Unique identifier
	Type         OperationType           // Type of operation
	Action       diff.MetadataDiffAction // CREATE, DROP, or ALTER
	Description  string                  // Human-readable description
	Risk         RiskLevel               // Risk level
	RiskReason   string                  // Explanation of risk
	Dependencies []string                // IDs of operations this depends on
	SQL          string                  // Generated SQL (empty if not generated yet)
	Reversible   bool                    // Whether this operation can be rolled back
	RollbackSQL  string                  // SQL to rollback this operation
	Warnings     []string                // Any warnings about this operation
}

// MigrationStrategy represents the complete migration execution plan.
type MigrationStrategy struct {
	Engine            engine.Engine
	Operations        []*MigrationOperation
	OrderedOperations []*MigrationOperation // Operations in dependency order
	HighRiskOps       []*MigrationOperation // Operations flagged as high risk
	DataLossOps       []*MigrationOperation // Operations that may cause data loss
	Warnings          []string              // Global warnings
}

// AnalyzeStrategy analyzes a MetadataDiff and produces a migration strategy.
func AnalyzeStrategy(mdiff *diff.MetadataDiff, eng engine.Engine) (*MigrationStrategy, error) {
	strategy := &MigrationStrategy{
		Engine:     eng,
		Operations: []*MigrationOperation{},
		Warnings:   []string{},
	}

	// Convert diffs to operations
	if err := convertSchemaChanges(strategy, mdiff); err != nil {
		return nil, errors.Wrap(err, "failed to convert schema changes")
	}
	if err := convertExtensionChanges(strategy, mdiff); err != nil {
		return nil, errors.Wrap(err, "failed to convert extension changes")
	}
	if err := convertEnumTypeChanges(strategy, mdiff); err != nil {
		return nil, errors.Wrap(err, "failed to convert enum type changes")
	}
	if err := convertSequenceChanges(strategy, mdiff); err != nil {
		return nil, errors.Wrap(err, "failed to convert sequence changes")
	}
	if err := convertTableChanges(strategy, mdiff); err != nil {
		return nil, errors.Wrap(err, "failed to convert table changes")
	}
	if err := convertViewChanges(strategy, mdiff); err != nil {
		return nil, errors.Wrap(err, "failed to convert view changes")
	}
	if err := convertMaterializedViewChanges(strategy, mdiff); err != nil {
		return nil, errors.Wrap(err, "failed to convert materialized view changes")
	}
	if err := convertFunctionChanges(strategy, mdiff); err != nil {
		return nil, errors.Wrap(err, "failed to convert function changes")
	}
	if err := convertProcedureChanges(strategy, mdiff); err != nil {
		return nil, errors.Wrap(err, "failed to convert procedure changes")
	}
	if err := convertEventChanges(strategy, mdiff); err != nil {
		return nil, errors.Wrap(err, "failed to convert event changes")
	}

	// Build dependency graph and order operations
	if err := orderOperations(strategy); err != nil {
		return nil, errors.Wrap(err, "failed to order operations")
	}

	// Identify high-risk and data-loss operations
	categorizeRisks(strategy)

	return strategy, nil
}

// convertSchemaChanges converts schema diffs to migration operations.
func convertSchemaChanges(strategy *MigrationStrategy, mdiff *diff.MetadataDiff) error {
	for i, schemaDiff := range mdiff.SchemaChanges {
		op := &MigrationOperation{
			ID:          fmt.Sprintf("schema_%d", i),
			Type:        OperationTypeSchema,
			Action:      schemaDiff.Action,
			Description: fmt.Sprintf("%s schema '%s'", schemaDiff.Action, schemaDiff.SchemaName),
		}

		switch schemaDiff.Action {
		case diff.MetadataDiffActionCreate:
			op.Risk = RiskLevelNone
			op.Reversible = true
			op.RiskReason = "Creating new schema has no impact on existing data"
		case diff.MetadataDiffActionDrop:
			op.Risk = RiskLevelHigh
			op.Reversible = false
			op.RiskReason = "Dropping schema will permanently delete all objects and data within it"
			op.Warnings = append(op.Warnings, "All tables, views, functions, and data in this schema will be lost")
		case diff.MetadataDiffActionAlter:
			op.Risk = RiskLevelLow
			op.Reversible = true
			op.RiskReason = "Schema alterations typically have minimal risk"
		}

		strategy.Operations = append(strategy.Operations, op)
	}
	return nil
}

// convertExtensionChanges converts extension diffs to migration operations.
func convertExtensionChanges(strategy *MigrationStrategy, mdiff *diff.MetadataDiff) error {
	for i, extDiff := range mdiff.ExtensionChanges {
		op := &MigrationOperation{
			ID:          fmt.Sprintf("extension_%d", i),
			Type:        OperationTypeExtension,
			Action:      extDiff.Action,
			Description: fmt.Sprintf("%s extension '%s'", extDiff.Action, extDiff.ExtensionName),
		}

		switch extDiff.Action {
		case diff.MetadataDiffActionCreate:
			op.Risk = RiskLevelLow
			op.Reversible = true
			op.RiskReason = "Creating extension is generally safe but may affect database behavior"
			op.Warnings = append(op.Warnings, "Ensure extension is available in target database")
		case diff.MetadataDiffActionDrop:
			op.Risk = RiskLevelMedium
			op.Reversible = true
			op.RiskReason = "Dropping extension may break objects that depend on it"
			op.Warnings = append(op.Warnings, "Check for dependent objects before dropping")
		case diff.MetadataDiffActionAlter:
			op.Risk = RiskLevelLow
			op.Reversible = true
			op.RiskReason = "Altering extension version typically has low risk"
		}

		strategy.Operations = append(strategy.Operations, op)
	}
	return nil
}

// convertEnumTypeChanges converts enum type diffs to migration operations.
func convertEnumTypeChanges(strategy *MigrationStrategy, mdiff *diff.MetadataDiff) error {
	for i, enumDiff := range mdiff.EnumTypeChanges {
		op := &MigrationOperation{
			ID:          fmt.Sprintf("enum_%d", i),
			Type:        OperationTypeEnum,
			Action:      enumDiff.Action,
			Description: fmt.Sprintf("%s enum type '%s.%s'", enumDiff.Action, enumDiff.SchemaName, enumDiff.EnumName),
		}

		switch enumDiff.Action {
		case diff.MetadataDiffActionCreate:
			op.Risk = RiskLevelNone
			op.Reversible = true
			op.RiskReason = "Creating new enum type has no impact on existing data"
		case diff.MetadataDiffActionDrop:
			op.Risk = RiskLevelHigh
			op.Reversible = false
			op.RiskReason = "Dropping enum type requires no columns use it"
			op.Warnings = append(op.Warnings, "Ensure no columns reference this enum type")
		case diff.MetadataDiffActionAlter:
			if len(enumDiff.RemovedValues) > 0 {
				op.Risk = RiskLevelHigh
				op.Reversible = false
				op.RiskReason = "Removing enum values may fail if values are in use"
				op.Warnings = append(
					op.Warnings,
					fmt.Sprintf("Cannot remove enum values that are in use: %v", enumDiff.RemovedValues),
				)
			} else if len(enumDiff.AddedValues) > 0 {
				op.Risk = RiskLevelLow
				op.Reversible = true
				op.RiskReason = "Adding enum values is safe"
			}
			if enumDiff.OrderChanged {
				op.Warnings = append(op.Warnings, "Enum value order changed, may affect ORDER BY queries")
			}
		}

		strategy.Operations = append(strategy.Operations, op)
	}
	return nil
}

// convertSequenceChanges converts sequence diffs to migration operations.
func convertSequenceChanges(strategy *MigrationStrategy, mdiff *diff.MetadataDiff) error {
	for i, seqDiff := range mdiff.SequenceChanges {
		op := &MigrationOperation{
			ID:          fmt.Sprintf("sequence_%d", i),
			Type:        OperationTypeSequence,
			Action:      seqDiff.Action,
			Description: fmt.Sprintf("%s sequence '%s'", seqDiff.Action, seqDiff.NewSequence.Name),
		}

		switch seqDiff.Action {
		case diff.MetadataDiffActionCreate:
			op.Risk = RiskLevelNone
			op.Reversible = true
			op.RiskReason = "Creating new sequence has no impact"
		case diff.MetadataDiffActionDrop:
			op.Risk = RiskLevelMedium
			op.Reversible = false
			op.RiskReason = "Dropping sequence may affect default values for columns"
			op.Warnings = append(op.Warnings, "Check if any columns use this sequence for default values")
		case diff.MetadataDiffActionAlter:
			op.Risk = RiskLevelLow
			op.Reversible = true
			op.RiskReason = "Altering sequence properties is generally safe"
			if seqDiff.StartChanged {
				op.Warnings = append(op.Warnings, "Changing START value only affects new sequences, not existing ones")
			}
		}

		strategy.Operations = append(strategy.Operations, op)
	}
	return nil
}

// convertTableChanges converts table diffs to migration operations.
func convertTableChanges(strategy *MigrationStrategy, mdiff *diff.MetadataDiff) error {
	for i, tableDiff := range mdiff.TableChanges {
		opID := fmt.Sprintf("table_%d", i)

		switch tableDiff.Action {
		case diff.MetadataDiffActionCreate:
			op := &MigrationOperation{
				ID:          opID,
				Type:        OperationTypeTable,
				Action:      tableDiff.Action,
				Description: fmt.Sprintf("CREATE table '%s.%s'", tableDiff.SchemaName, tableDiff.TableName),
				Risk:        RiskLevelNone,
				Reversible:  true,
				RiskReason:  "Creating new table has no impact on existing data",
			}
			strategy.Operations = append(strategy.Operations, op)

		case diff.MetadataDiffActionDrop:
			op := &MigrationOperation{
				ID:          opID,
				Type:        OperationTypeTable,
				Action:      tableDiff.Action,
				Description: fmt.Sprintf("DROP table '%s.%s'", tableDiff.SchemaName, tableDiff.TableName),
				Risk:        RiskLevelHigh,
				Reversible:  false,
				RiskReason:  "Dropping table will permanently delete all data",
			}
			op.Warnings = append(op.Warnings, "All data in this table will be permanently lost")
			strategy.Operations = append(strategy.Operations, op)

		case diff.MetadataDiffActionAlter:
			// Process sub-changes for ALTER TABLE
			convertTableSubChanges(strategy, tableDiff, opID)
		}
	}
	return nil
}

// convertTableSubChanges processes column, index, and constraint changes.
func convertTableSubChanges(strategy *MigrationStrategy, tableDiff *diff.TableDiff, tableOpID string) {
	tableRef := fmt.Sprintf("%s.%s", tableDiff.SchemaName, tableDiff.TableName)

	// Column changes
	for i, colDiff := range tableDiff.ColumnChanges {
		op := &MigrationOperation{
			ID:           fmt.Sprintf("%s_col_%d", tableOpID, i),
			Type:         OperationTypeColumn,
			Action:       colDiff.Action,
			Description:  fmt.Sprintf("%s column '%s.%s'", colDiff.Action, tableRef, getColumnName(colDiff)),
			Dependencies: []string{tableOpID},
		}

		switch colDiff.Action {
		case diff.MetadataDiffActionCreate:
			op.Risk = RiskLevelLow
			op.Reversible = true
			op.RiskReason = "Adding column is safe if nullable or has default"
			if colDiff.NewColumn != nil && !colDiff.NewColumn.Nullable && colDiff.NewColumn.Default == "" {
				op.Risk = RiskLevelMedium
				op.RiskReason = "Adding NOT NULL column without default requires all rows to have a value"
				op.Warnings = append(op.Warnings, "NOT NULL column without default may fail if table has existing rows")
			}

		case diff.MetadataDiffActionDrop:
			op.Risk = RiskLevelHigh
			op.Reversible = false
			op.RiskReason = "Dropping column permanently deletes data"
			op.Warnings = append(op.Warnings, "Column data will be permanently lost")

		case diff.MetadataDiffActionAlter:
			op.Risk = RiskLevelMedium
			op.Reversible = false
			op.RiskReason = "Altering column may require data conversion"
			if colDiff.OldColumn != nil && colDiff.NewColumn != nil {
				if colDiff.OldColumn.Nullable && !colDiff.NewColumn.Nullable {
					op.Risk = RiskLevelHigh
					op.Warnings = append(op.Warnings, "Changing to NOT NULL may fail if NULL values exist")
				}
				if colDiff.OldColumn.Type != colDiff.NewColumn.Type {
					op.Warnings = append(
						op.Warnings,
						"Type change may require explicit USING clause or fail for incompatible data",
					)
				}
			}
		}

		strategy.Operations = append(strategy.Operations, op)
	}

	// Index changes
	for i, idxDiff := range tableDiff.IndexChanges {
		op := &MigrationOperation{
			ID:           fmt.Sprintf("%s_idx_%d", tableOpID, i),
			Type:         OperationTypeIndex,
			Action:       idxDiff.Action,
			Description:  fmt.Sprintf("%s index '%s' on '%s'", idxDiff.Action, getIndexName(idxDiff), tableRef),
			Dependencies: []string{tableOpID},
		}

		switch idxDiff.Action {
		case diff.MetadataDiffActionCreate:
			op.Risk = RiskLevelLow
			op.Reversible = true
			op.RiskReason = "Creating index is safe but may take time on large tables"
			op.Warnings = append(op.Warnings, "Use CREATE INDEX CONCURRENTLY for large tables to avoid locking")

		case diff.MetadataDiffActionDrop:
			op.Risk = RiskLevelMedium
			op.Reversible = true
			op.RiskReason = "Dropping index may impact query performance"
			op.Warnings = append(op.Warnings, "Ensure no queries depend on this index for performance")

		case diff.MetadataDiffActionAlter:
			op.Risk = RiskLevelMedium
			op.Reversible = true
			op.RiskReason = "Altering index typically requires DROP and CREATE"
		}

		strategy.Operations = append(strategy.Operations, op)
	}

	// Foreign key changes
	for i, fkDiff := range tableDiff.ForeignKeyChanges {
		op := &MigrationOperation{
			ID:           fmt.Sprintf("%s_fk_%d", tableOpID, i),
			Type:         OperationTypeConstraint,
			Action:       fkDiff.Action,
			Description:  fmt.Sprintf("%s foreign key on '%s'", fkDiff.Action, tableRef),
			Dependencies: []string{tableOpID},
		}

		switch fkDiff.Action {
		case diff.MetadataDiffActionCreate:
			op.Risk = RiskLevelMedium
			op.Reversible = true
			op.RiskReason = "Adding foreign key validates existing data"
			op.Warnings = append(op.Warnings, "Foreign key creation may fail if existing data violates constraint")

		case diff.MetadataDiffActionDrop:
			op.Risk = RiskLevelLow
			op.Reversible = true
			op.RiskReason = "Dropping foreign key removes data integrity check"

		case diff.MetadataDiffActionAlter:
			op.Risk = RiskLevelMedium
			op.Reversible = true
			op.RiskReason = "Altering foreign key requires DROP and CREATE"
		}

		strategy.Operations = append(strategy.Operations, op)
	}

	// Check constraint changes
	for i, chkDiff := range tableDiff.CheckConstraintChanges {
		op := &MigrationOperation{
			ID:           fmt.Sprintf("%s_chk_%d", tableOpID, i),
			Type:         OperationTypeConstraint,
			Action:       chkDiff.Action,
			Description:  fmt.Sprintf("%s check constraint on '%s'", chkDiff.Action, tableRef),
			Dependencies: []string{tableOpID},
		}

		switch chkDiff.Action {
		case diff.MetadataDiffActionCreate:
			op.Risk = RiskLevelMedium
			op.Reversible = true
			op.RiskReason = "Adding check constraint validates existing data"
			op.Warnings = append(op.Warnings, "Check constraint creation may fail if existing data violates constraint")

		case diff.MetadataDiffActionDrop:
			op.Risk = RiskLevelLow
			op.Reversible = true
			op.RiskReason = "Dropping check constraint removes data validation"

		case diff.MetadataDiffActionAlter:
			op.Risk = RiskLevelMedium
			op.Reversible = true
			op.RiskReason = "Altering check constraint requires DROP and CREATE"
		}

		strategy.Operations = append(strategy.Operations, op)
	}
}

// convertViewChanges converts view diffs to migration operations.
func convertViewChanges(strategy *MigrationStrategy, mdiff *diff.MetadataDiff) error {
	for i, viewDiff := range mdiff.ViewChanges {
		op := &MigrationOperation{
			ID:          fmt.Sprintf("view_%d", i),
			Type:        OperationTypeView,
			Action:      viewDiff.Action,
			Description: fmt.Sprintf("%s view '%s.%s'", viewDiff.Action, viewDiff.SchemaName, viewDiff.ViewName),
		}

		switch viewDiff.Action {
		case diff.MetadataDiffActionCreate:
			op.Risk = RiskLevelNone
			op.Reversible = true
			op.RiskReason = "Creating new view has no impact on data"

		case diff.MetadataDiffActionDrop:
			op.Risk = RiskLevelMedium
			op.Reversible = true
			op.RiskReason = "Dropping view may break dependent views or applications"
			op.Warnings = append(op.Warnings, "Check for dependent objects before dropping")

		case diff.MetadataDiffActionAlter:
			op.Risk = RiskLevelLow
			op.Reversible = true
			op.RiskReason = "Altering view is safe, uses CREATE OR REPLACE"
			if viewDiff.DefinitionChanged {
				op.Warnings = append(op.Warnings, "View definition changed, may affect dependent objects or queries")
			}
		}

		strategy.Operations = append(strategy.Operations, op)
	}
	return nil
}

// convertMaterializedViewChanges converts materialized view diffs to migration operations.
func convertMaterializedViewChanges(strategy *MigrationStrategy, mdiff *diff.MetadataDiff) error {
	for i, mvDiff := range mdiff.MaterializedViewChanges {
		op := &MigrationOperation{
			ID:          fmt.Sprintf("mview_%d", i),
			Type:        OperationTypeView,
			Action:      mvDiff.Action,
			Description: fmt.Sprintf("%s materialized view '%s.%s'", mvDiff.Action, mvDiff.SchemaName, mvDiff.ViewName),
		}

		switch mvDiff.Action {
		case diff.MetadataDiffActionCreate:
			op.Risk = RiskLevelLow
			op.Reversible = true
			op.RiskReason = "Creating materialized view requires initial data population"
			op.Warnings = append(op.Warnings, "Initial population may take time on large datasets")

		case diff.MetadataDiffActionDrop:
			op.Risk = RiskLevelMedium
			op.Reversible = false
			op.RiskReason = "Dropping materialized view deletes cached data"
			op.Warnings = append(op.Warnings, "Materialized data will be lost and need to be rebuilt if recreated")

		case diff.MetadataDiffActionAlter:
			op.Risk = RiskLevelMedium
			op.Reversible = false
			op.RiskReason = "Altering materialized view requires DROP and CREATE, losing cached data"
			op.Warnings = append(op.Warnings, "Materialized view will need to be repopulated")
		}

		strategy.Operations = append(strategy.Operations, op)
	}
	return nil
}

// convertFunctionChanges converts function diffs to migration operations.
func convertFunctionChanges(strategy *MigrationStrategy, mdiff *diff.MetadataDiff) error {
	for i, funcDiff := range mdiff.FunctionChanges {
		op := &MigrationOperation{
			ID:          fmt.Sprintf("function_%d", i),
			Type:        OperationTypeFunction,
			Action:      funcDiff.Action,
			Description: fmt.Sprintf("%s function '%s'", funcDiff.Action, funcDiff.NewFunction.Name),
		}

		switch funcDiff.Action {
		case diff.MetadataDiffActionCreate:
			op.Risk = RiskLevelNone
			op.Reversible = true
			op.RiskReason = "Creating new function has no impact"

		case diff.MetadataDiffActionDrop:
			op.Risk = RiskLevelMedium
			op.Reversible = true
			op.RiskReason = "Dropping function may break dependent objects"
			op.Warnings = append(op.Warnings, "Check for dependent views, triggers, or other functions")

		case diff.MetadataDiffActionAlter:
			op.Risk = RiskLevelLow
			op.Reversible = true
			if funcDiff.CanUseAlterFunction {
				op.RiskReason = "Function can be altered in place"
			} else {
				op.RiskReason = "Function requires DROP and CREATE"
				op.Warnings = append(op.Warnings, "Function will be briefly unavailable during replacement")
			}
		}

		strategy.Operations = append(strategy.Operations, op)
	}
	return nil
}

// convertProcedureChanges converts procedure diffs to migration operations.
func convertProcedureChanges(strategy *MigrationStrategy, mdiff *diff.MetadataDiff) error {
	for i, procDiff := range mdiff.ProcedureChanges {
		op := &MigrationOperation{
			ID:          fmt.Sprintf("procedure_%d", i),
			Type:        OperationTypeProcedure,
			Action:      procDiff.Action,
			Description: fmt.Sprintf("%s procedure '%s'", procDiff.Action, procDiff.NewProcedure.Name),
		}

		switch procDiff.Action {
		case diff.MetadataDiffActionCreate:
			op.Risk = RiskLevelNone
			op.Reversible = true
			op.RiskReason = "Creating new procedure has no impact"

		case diff.MetadataDiffActionDrop:
			op.Risk = RiskLevelMedium
			op.Reversible = true
			op.RiskReason = "Dropping procedure may break dependent objects or applications"
			op.Warnings = append(op.Warnings, "Check for dependent stored procedures or application code")

		case diff.MetadataDiffActionAlter:
			op.Risk = RiskLevelLow
			op.Reversible = true
			if procDiff.CanUseAlterProcedure {
				op.RiskReason = "Procedure can be altered in place"
			} else {
				op.RiskReason = "Procedure requires DROP and CREATE"
				op.Warnings = append(op.Warnings, "Procedure will be briefly unavailable during replacement")
			}
		}

		strategy.Operations = append(strategy.Operations, op)
	}
	return nil
}

// convertEventChanges converts event diffs to migration operations.
func convertEventChanges(strategy *MigrationStrategy, mdiff *diff.MetadataDiff) error {
	for i, eventDiff := range mdiff.EventChanges {
		op := &MigrationOperation{
			ID:          fmt.Sprintf("event_%d", i),
			Type:        OperationTypeEvent,
			Action:      eventDiff.Action,
			Description: fmt.Sprintf("%s event '%s'", eventDiff.Action, eventDiff.NewEvent.Name),
		}

		switch eventDiff.Action {
		case diff.MetadataDiffActionCreate:
			op.Risk = RiskLevelLow
			op.Reversible = true
			op.RiskReason = "Creating event is safe but will start executing automatically"
			op.Warnings = append(op.Warnings, "Event will execute according to its schedule")

		case diff.MetadataDiffActionDrop:
			op.Risk = RiskLevelMedium
			op.Reversible = true
			op.RiskReason = "Dropping event stops scheduled operations"
			op.Warnings = append(op.Warnings, "Scheduled operations will no longer run")

		case diff.MetadataDiffActionAlter:
			op.Risk = RiskLevelLow
			op.Reversible = true
			op.RiskReason = "Altering event changes scheduled behavior"
		}

		strategy.Operations = append(strategy.Operations, op)
	}
	return nil
}

// orderOperations builds dependency graph and orders operations.
func orderOperations(strategy *MigrationStrategy) error {
	// Build dependency map
	opMap := make(map[string]*MigrationOperation)
	for _, op := range strategy.Operations {
		opMap[op.ID] = op
	}

	// Topological sort
	visited := make(map[string]bool)
	tempMark := make(map[string]bool)
	var ordered []*MigrationOperation

	var visit func(string) error
	visit = func(id string) error {
		if tempMark[id] {
			return errors.Errorf("circular dependency detected involving operation %s", id)
		}
		if visited[id] {
			return nil
		}

		tempMark[id] = true
		op := opMap[id]
		if op != nil {
			for _, depID := range op.Dependencies {
				if err := visit(depID); err != nil {
					return err
				}
			}
		}
		tempMark[id] = false
		visited[id] = true
		if op != nil {
			ordered = append(ordered, op)
		}
		return nil
	}

	for _, op := range strategy.Operations {
		if !visited[op.ID] {
			if err := visit(op.ID); err != nil {
				return err
			}
		}
	}

	strategy.OrderedOperations = ordered
	return nil
}

// categorizeRisks identifies high-risk and data-loss operations.
func categorizeRisks(strategy *MigrationStrategy) {
	for _, op := range strategy.Operations {
		if op.Risk == RiskLevelHigh {
			strategy.HighRiskOps = append(strategy.HighRiskOps, op)
		}
		if !op.Reversible || op.Action == diff.MetadataDiffActionDrop {
			strategy.DataLossOps = append(strategy.DataLossOps, op)
		}
	}

	// Add global warnings
	if len(strategy.HighRiskOps) > 0 {
		strategy.Warnings = append(strategy.Warnings,
			fmt.Sprintf("%d high-risk operations detected that may cause data loss", len(strategy.HighRiskOps)))
	}
	if len(strategy.DataLossOps) > 0 {
		strategy.Warnings = append(strategy.Warnings,
			fmt.Sprintf("%d operations may result in permanent data loss", len(strategy.DataLossOps)))
	}
}

// Helper functions
func getColumnName(colDiff *diff.ColumnDiff) string {
	if colDiff.NewColumn != nil {
		return colDiff.NewColumn.Name
	}
	if colDiff.OldColumn != nil {
		return colDiff.OldColumn.Name
	}
	return "unknown"
}

func getIndexName(idxDiff *diff.IndexDiff) string {
	if idxDiff.NewIndex != nil {
		return idxDiff.NewIndex.Name
	}
	if idxDiff.OldIndex != nil {
		return idxDiff.OldIndex.Name
	}
	return "unknown"
}

// SortByRisk sorts operations by risk level (high to low) for review purposes.
func SortByRisk(ops []*MigrationOperation) []*MigrationOperation {
	sorted := make([]*MigrationOperation, len(ops))
	copy(sorted, ops)

	riskOrder := map[RiskLevel]int{
		RiskLevelHigh:   0,
		RiskLevelMedium: 1,
		RiskLevelLow:    2,
		RiskLevelNone:   3,
	}

	slices.SortFunc(sorted, func(a, b *MigrationOperation) int {
		orderA := riskOrder[a.Risk]
		orderB := riskOrder[b.Risk]
		if orderA < orderB {
			return -1
		} else if orderA > orderB {
			return 1
		}
		return 0
	})

	return sorted
}
