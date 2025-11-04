package comparer

import (
	"slices"

	"github.com/pkg/errors"

	schemaextract "github.com/nsxbet/sql-schema"
	"github.com/nsxbet/sql-schema/comparer/engine"
	"github.com/nsxbet/sql-schema/diff"
)

// CompareOptions configures schema comparison behavior.
type CompareOptions struct {
	// Engine specifies which database engine to use for comparison.
	Engine engine.Engine

	// IgnoreComments ignores comment differences.
	IgnoreComments bool

	// IgnoreCharset ignores charset differences (MySQL).
	IgnoreCharset bool

	// IgnoreCollation ignores collation differences.
	IgnoreCollation bool
}

// DefaultCompareOptions returns default comparison options.
func DefaultCompareOptions(eng engine.Engine) *CompareOptions {
	return &CompareOptions{
		Engine:          eng,
		IgnoreComments:  false,
		IgnoreCharset:   false,
		IgnoreCollation: false,
	}
}

// CompareSchemasDetailed compares two database schemas and returns detailed differences.
// This is the main entry point for engine-aware schema comparison.
func CompareSchemasDetailed(oldSchema, newSchema *schemaextract.DatabaseSchema, opts *CompareOptions) (*diff.MetadataDiff, error) {
	if opts == nil {
		opts = DefaultCompareOptions(engine.PostgreSQL)
	}

	// Get the engine-specific comparer
	comparer, err := engine.Get(opts.Engine)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to get comparer for engine %s", opts.Engine)
	}

	result := &diff.MetadataDiff{
		DatabaseName: newSchema.Name,
	}

	// Compare schemas (namespaces)
	result.SchemaChanges = compareSchemasLevel(oldSchema, newSchema)

	// Compare all object types within each schema
	for _, oldSch := range oldSchema.Schemas {
		newSch := findSchema(newSchema.Schemas, oldSch.Name)
		if newSch == nil {
			// Schema was dropped - all objects in it are dropped
			continue
		}

		// Compare tables
		tableChanges, err := compareTablesDetailed(oldSch.Tables, newSch.Tables, oldSch.Name, comparer, opts)
		if err != nil {
			return nil, errors.Wrap(err, "failed to compare tables")
		}
		result.TableChanges = append(result.TableChanges, tableChanges...)

		// Compare views
		viewChanges, err := compareViewsDetailed(oldSch.Views, newSch.Views, oldSch.Name, comparer, opts)
		if err != nil {
			return nil, errors.Wrap(err, "failed to compare views")
		}
		result.ViewChanges = append(result.ViewChanges, viewChanges...)

		// Compare materialized views
		mvChanges, err := compareMaterializedViews(oldSch.MaterializedViews, newSch.MaterializedViews, oldSch.Name, comparer, opts)
		if err != nil {
			return nil, errors.Wrap(err, "failed to compare materialized views")
		}
		result.MaterializedViewChanges = append(result.MaterializedViewChanges, mvChanges...)

		// Compare functions
		functionChanges, err := compareFunctionsDetailed(oldSch.Functions, newSch.Functions, oldSch.Name, comparer, opts)
		if err != nil {
			return nil, errors.Wrap(err, "failed to compare functions")
		}
		result.FunctionChanges = append(result.FunctionChanges, functionChanges...)

		// Compare procedures
		procedureChanges := compareProcedures(oldSch.Procedures, newSch.Procedures, oldSch.Name, comparer, opts)
		result.ProcedureChanges = append(result.ProcedureChanges, procedureChanges...)

		// Compare sequences
		sequenceChanges := compareSequences(oldSch.Sequences, newSch.Sequences, oldSch.Name, comparer, opts)
		result.SequenceChanges = append(result.SequenceChanges, sequenceChanges...)

		// Compare enum types (PostgreSQL)
		enumChanges := compareEnumTypes(oldSch.EnumTypes, newSch.EnumTypes, oldSch.Name, opts)
		result.EnumTypeChanges = append(result.EnumTypeChanges, enumChanges...)

		// Compare events (MySQL)
		eventChanges := compareEvents(oldSch.Events, newSch.Events, opts)
		result.EventChanges = append(result.EventChanges, eventChanges...)

		// Compare extensions (PostgreSQL)
		extensionChanges := compareExtensions(oldSch.Extensions, newSch.Extensions, oldSch.Name, opts)
		result.ExtensionChanges = append(result.ExtensionChanges, extensionChanges...)
	}

	// Check for new schemas
	for _, newSch := range newSchema.Schemas {
		oldSch := findSchema(oldSchema.Schemas, newSch.Name)
		if oldSch == nil {
			// New schema - all objects in it are new
			result.SchemaChanges = append(result.SchemaChanges, &diff.SchemaDiff{
				Action:     diff.MetadataDiffActionCreate,
				SchemaName: newSch.Name,
				NewSchema:  newSch,
			})

			// Add all new objects
			for _, table := range newSch.Tables {
				result.TableChanges = append(result.TableChanges, &diff.TableDiff{
					Action:     diff.MetadataDiffActionCreate,
					SchemaName: newSch.Name,
					TableName:  table.Name,
					NewTable:   table,
				})
			}

			// Similar for other object types...
		}
	}

	// Sort diffs for deterministic output
	sortMetadataDiff(result)

	return result, nil
}

// compareSchemasLevel compares schema-level changes (CREATE/DROP schemas).
func compareSchemasLevel(oldDB, newDB *schemaextract.DatabaseSchema) []*diff.SchemaDiff {
	var diffs []*diff.SchemaDiff

	// Check for dropped schemas
	for _, oldSch := range oldDB.Schemas {
		if findSchema(newDB.Schemas, oldSch.Name) == nil {
			diffs = append(diffs, &diff.SchemaDiff{
				Action:     diff.MetadataDiffActionDrop,
				SchemaName: oldSch.Name,
				OldSchema:  oldSch,
			})
		}
	}

	// Check for new schemas
	for _, newSch := range newDB.Schemas {
		if findSchema(oldDB.Schemas, newSch.Name) == nil {
			diffs = append(diffs, &diff.SchemaDiff{
				Action:     diff.MetadataDiffActionCreate,
				SchemaName: newSch.Name,
				NewSchema:  newSch,
			})
		}
	}

	return diffs
}

// compareTables compares tables between old and new schemas.
func compareTablesDetailed(oldTables, newTables []*schemaextract.Table, schemaName string, comparer engine.Comparer, opts *CompareOptions) ([]*diff.TableDiff, error) {
	var diffs []*diff.TableDiff

	// Build maps for efficient lookup
	oldMap := make(map[string]*schemaextract.Table)
	for _, t := range oldTables {
		oldMap[t.Name] = t
	}

	newMap := make(map[string]*schemaextract.Table)
	for _, t := range newTables {
		newMap[t.Name] = t
	}

	// Check for dropped tables
	for name, oldTable := range oldMap {
		if _, exists := newMap[name]; !exists {
			diffs = append(diffs, &diff.TableDiff{
				Action:     diff.MetadataDiffActionDrop,
				SchemaName: schemaName,
				TableName:  name,
				OldTable:   oldTable,
			})
		}
	}

	// Check for new and modified tables
	for name, newTable := range newMap {
		oldTable, exists := oldMap[name]
		if !exists {
			// New table
			diffs = append(diffs, &diff.TableDiff{
				Action:     diff.MetadataDiffActionCreate,
				SchemaName: schemaName,
				TableName:  name,
				NewTable:   newTable,
			})
		} else {
			// Compare table details
			tableDiff, err := compareTableDetails(oldTable, newTable, schemaName, comparer, opts)
			if err != nil {
				return nil, err
			}
			if tableDiff != nil && tableDiff.HasChanges() {
				diffs = append(diffs, tableDiff)
			}
		}
	}

	return diffs, nil
}

// compareTableDetails compares two tables in detail.
func compareTableDetails(oldTable, newTable *schemaextract.Table, schemaName string, comparer engine.Comparer, opts *CompareOptions) (*diff.TableDiff, error) {
	diff := &diff.TableDiff{
		Action:     diff.MetadataDiffActionAlter,
		SchemaName: schemaName,
		TableName:  newTable.Name,
		OldTable:   oldTable,
		NewTable:   newTable,
	}

	// Compare columns
	columnDiffs, err := compareColumns(oldTable.Columns, newTable.Columns, comparer.Column())
	if err != nil {
		return nil, err
	}
	diff.ColumnChanges = columnDiffs

	// Compare indexes
	indexDiffs, err := compareIndexes(oldTable.Indexes, newTable.Indexes, comparer.Index())
	if err != nil {
		return nil, err
	}
	diff.IndexChanges, diff.PrimaryKeyChanges, diff.UniqueConstraintChanges = splitIndexDiffs(indexDiffs)

	// Compare foreign keys
	diff.ForeignKeyChanges = compareForeignKeys(oldTable.ForeignKeys, newTable.ForeignKeys)

	// Compare check constraints
	diff.CheckConstraintChanges = compareCheckConstraints(oldTable.CheckConstraints, newTable.CheckConstraints)

	// Compare triggers
	diff.TriggerChanges = compareTriggers(oldTable.Triggers, newTable.Triggers)

	// Compare partitions
	diff.PartitionChanges = comparePartitions(oldTable.Partitions, newTable.Partitions)

	// Compare rules (PostgreSQL)
	diff.RuleChanges = compareRules(oldTable.Rules, newTable.Rules)

	// Compare table-level properties
	if !opts.IgnoreComments && oldTable.Comment != newTable.Comment {
		diff.CommentChanged = true
	}
	if !opts.IgnoreCollation && oldTable.Collation != newTable.Collation {
		diff.CollationChanged = true
	}
	if !opts.IgnoreCharset && oldTable.Charset != newTable.Charset {
		diff.CharsetChanged = true
	}
	if oldTable.Engine != newTable.Engine {
		diff.EngineChanged = true
	}

	return diff, nil
}

// compareColumns compares columns using the engine-specific column comparer.
func compareColumns(oldColumns, newColumns []*schemaextract.Column, comparer engine.ColumnComparer) ([]*diff.ColumnDiff, error) {
	var diffs []*diff.ColumnDiff

	// Build maps
	oldMap := make(map[string]*schemaextract.Column)
	for _, c := range oldColumns {
		oldMap[c.Name] = c
	}

	newMap := make(map[string]*schemaextract.Column)
	for _, c := range newColumns {
		newMap[c.Name] = c
	}

	// Check for dropped columns
	for name, oldCol := range oldMap {
		if _, exists := newMap[name]; !exists {
			diff, err := comparer.CompareColumns(oldCol, nil)
			if err != nil {
				return nil, err
			}
			if diff != nil {
				diffs = append(diffs, diff)
			}
		}
	}

	// Check for new and modified columns
	for name, newCol := range newMap {
		oldCol, exists := oldMap[name]
		var diff *diff.ColumnDiff
		var err error

		if !exists {
			diff, err = comparer.CompareColumns(nil, newCol)
		} else {
			diff, err = comparer.CompareColumns(oldCol, newCol)
		}

		if err != nil {
			return nil, err
		}
		if diff != nil {
			diffs = append(diffs, diff)
		}
	}

	return diffs, nil
}

// compareIndexes compares indexes using the engine-specific index comparer.
func compareIndexes(oldIndexes, newIndexes []*schemaextract.Index, comparer engine.IndexComparer) ([]*diff.IndexDiff, error) {
	var diffs []*diff.IndexDiff

	// Build maps
	oldMap := make(map[string]*schemaextract.Index)
	for _, idx := range oldIndexes {
		oldMap[idx.Name] = idx
	}

	newMap := make(map[string]*schemaextract.Index)
	for _, idx := range newIndexes {
		newMap[idx.Name] = idx
	}

	// Check for dropped indexes
	for name, oldIdx := range oldMap {
		if _, exists := newMap[name]; !exists {
			diff, err := comparer.CompareIndexes(oldIdx, nil)
			if err != nil {
				return nil, err
			}
			if diff != nil {
				diffs = append(diffs, diff)
			}
		}
	}

	// Check for new and modified indexes
	for name, newIdx := range newMap {
		oldIdx, exists := oldMap[name]
		var diff *diff.IndexDiff
		var err error

		if !exists {
			diff, err = comparer.CompareIndexes(nil, newIdx)
		} else {
			diff, err = comparer.CompareIndexes(oldIdx, newIdx)
		}

		if err != nil {
			return nil, err
		}
		if diff != nil {
			diffs = append(diffs, diff)
		}
	}

	return diffs, nil
}

// splitIndexDiffs separates index diffs into regular indexes, primary keys, and unique constraints.
func splitIndexDiffs(indexDiffs []*diff.IndexDiff) ([]*diff.IndexDiff, []*diff.PrimaryKeyDiff, []*diff.UniqueConstraintDiff) {
	var indexes []*diff.IndexDiff
	var primaryKeys []*diff.PrimaryKeyDiff
	var uniqueConstraints []*diff.UniqueConstraintDiff

	for _, idxDiff := range indexDiffs {
		// Check if it's a primary key
		isPrimaryKey := (idxDiff.OldIndex != nil && idxDiff.OldIndex.Primary) ||
			(idxDiff.NewIndex != nil && idxDiff.NewIndex.Primary)

		if isPrimaryKey {
			primaryKeys = append(primaryKeys, &diff.PrimaryKeyDiff{
				Action:        idxDiff.Action,
				OldPrimaryKey: idxDiff.OldIndex,
				NewPrimaryKey: idxDiff.NewIndex,
			})
			continue
		}

		// Check if it's a unique constraint (but not primary key)
		isUnique := (idxDiff.OldIndex != nil && idxDiff.OldIndex.Unique) ||
			(idxDiff.NewIndex != nil && idxDiff.NewIndex.Unique)

		if isUnique {
			uniqueConstraints = append(uniqueConstraints, &diff.UniqueConstraintDiff{
				Action:        idxDiff.Action,
				OldConstraint: idxDiff.OldIndex,
				NewConstraint: idxDiff.NewIndex,
			})
			continue
		}

		// Regular index
		indexes = append(indexes, idxDiff)
	}

	return indexes, primaryKeys, uniqueConstraints
}

// Helper functions for comparing other object types...

func compareForeignKeys(oldFKs, newFKs []*schemaextract.ForeignKey) []*diff.ForeignKeyDiff {
	var diffs []*diff.ForeignKeyDiff

	oldMap := make(map[string]*schemaextract.ForeignKey)
	for _, fk := range oldFKs {
		oldMap[fk.Name] = fk
	}

	newMap := make(map[string]*schemaextract.ForeignKey)
	for _, fk := range newFKs {
		newMap[fk.Name] = fk
	}

	// Dropped FKs
	for name, oldFK := range oldMap {
		if _, exists := newMap[name]; !exists {
			diffs = append(diffs, &diff.ForeignKeyDiff{
				Action:        diff.MetadataDiffActionDrop,
				OldForeignKey: oldFK,
			})
		}
	}

	// New and modified FKs
	for name, newFK := range newMap {
		oldFK, exists := oldMap[name]
		if !exists {
			diffs = append(diffs, &diff.ForeignKeyDiff{
				Action:        diff.MetadataDiffActionCreate,
				NewForeignKey: newFK,
			})
		} else if !foreignKeysEqual(oldFK, newFK) {
			diffs = append(diffs, &diff.ForeignKeyDiff{
				Action:        diff.MetadataDiffActionAlter,
				OldForeignKey: oldFK,
				NewForeignKey: newFK,
			})
		}
	}

	return diffs
}

func foreignKeysEqual(fk1, fk2 *schemaextract.ForeignKey) bool {
	if fk1.ReferencedTable != fk2.ReferencedTable {
		return false
	}
	if len(fk1.Columns) != len(fk2.Columns) {
		return false
	}
	for i := range fk1.Columns {
		if fk1.Columns[i] != fk2.Columns[i] {
			return false
		}
	}
	if len(fk1.ReferencedColumns) != len(fk2.ReferencedColumns) {
		return false
	}
	for i := range fk1.ReferencedColumns {
		if fk1.ReferencedColumns[i] != fk2.ReferencedColumns[i] {
			return false
		}
	}
	if fk1.OnDelete != fk2.OnDelete || fk1.OnUpdate != fk2.OnUpdate {
		return false
	}
	return true
}

func compareCheckConstraints(oldCCs, newCCs []*schemaextract.CheckConstraint) []*diff.CheckConstraintDiff {
	var diffs []*diff.CheckConstraintDiff

	oldMap := make(map[string]*schemaextract.CheckConstraint)
	for _, cc := range oldCCs {
		oldMap[cc.Name] = cc
	}

	newMap := make(map[string]*schemaextract.CheckConstraint)
	for _, cc := range newCCs {
		newMap[cc.Name] = cc
	}

	for name, oldCC := range oldMap {
		if _, exists := newMap[name]; !exists {
			diffs = append(diffs, &diff.CheckConstraintDiff{
				Action:        diff.MetadataDiffActionDrop,
				OldConstraint: oldCC,
			})
		}
	}

	for name, newCC := range newMap {
		oldCC, exists := oldMap[name]
		if !exists {
			diffs = append(diffs, &diff.CheckConstraintDiff{
				Action:        diff.MetadataDiffActionCreate,
				NewConstraint: newCC,
			})
		} else if oldCC.Expression != newCC.Expression {
			diffs = append(diffs, &diff.CheckConstraintDiff{
				Action:        diff.MetadataDiffActionAlter,
				OldConstraint: oldCC,
				NewConstraint: newCC,
			})
		}
	}

	return diffs
}

func compareTriggers(oldTriggers, newTriggers []*schemaextract.Trigger) []*diff.TriggerDiff {
	var diffs []*diff.TriggerDiff

	oldMap := make(map[string]*schemaextract.Trigger)
	for _, t := range oldTriggers {
		oldMap[t.Name] = t
	}

	newMap := make(map[string]*schemaextract.Trigger)
	for _, t := range newTriggers {
		newMap[t.Name] = t
	}

	for name, oldTrigger := range oldMap {
		if _, exists := newMap[name]; !exists {
			diffs = append(diffs, &diff.TriggerDiff{
				Action:     diff.MetadataDiffActionDrop,
				OldTrigger: oldTrigger,
			})
		}
	}

	for name, newTrigger := range newMap {
		oldTrigger, exists := oldMap[name]
		if !exists {
			diffs = append(diffs, &diff.TriggerDiff{
				Action:     diff.MetadataDiffActionCreate,
				NewTrigger: newTrigger,
			})
		} else if oldTrigger.Body != newTrigger.Body {
			diffs = append(diffs, &diff.TriggerDiff{
				Action:     diff.MetadataDiffActionAlter,
				OldTrigger: oldTrigger,
				NewTrigger: newTrigger,
			})
		}
	}

	return diffs
}

func comparePartitions(oldPartitions, newPartitions []*schemaextract.Partition) []*diff.PartitionDiff {
	// Simplified partition comparison
	var diffs []*diff.PartitionDiff

	oldMap := make(map[string]*schemaextract.Partition)
	for _, p := range oldPartitions {
		oldMap[p.Name] = p
	}

	newMap := make(map[string]*schemaextract.Partition)
	for _, p := range newPartitions {
		newMap[p.Name] = p
	}

	for name, oldP := range oldMap {
		if _, exists := newMap[name]; !exists {
			diffs = append(diffs, &diff.PartitionDiff{
				Action:       diff.MetadataDiffActionDrop,
				OldPartition: oldP,
			})
		}
	}

	for name, newP := range newMap {
		oldP, exists := oldMap[name]
		if !exists {
			diffs = append(diffs, &diff.PartitionDiff{
				Action:       diff.MetadataDiffActionCreate,
				NewPartition: newP,
			})
		} else if !partitionsEqual(oldP, newP) {
			diffs = append(diffs, &diff.PartitionDiff{
				Action:       diff.MetadataDiffActionAlter,
				OldPartition: oldP,
				NewPartition: newP,
			})
		}
	}

	return diffs
}

func partitionsEqual(p1, p2 *schemaextract.Partition) bool {
	return p1.Type == p2.Type &&
		p1.Expression == p2.Expression &&
		p1.Value == p2.Value
}

func compareRules(oldRules, newRules []*schemaextract.Rule) []*diff.RuleDiff {
	var diffs []*diff.RuleDiff

	oldMap := make(map[string]*schemaextract.Rule)
	for _, r := range oldRules {
		oldMap[r.Name] = r
	}

	newMap := make(map[string]*schemaextract.Rule)
	for _, r := range newRules {
		newMap[r.Name] = r
	}

	for name, oldRule := range oldMap {
		if _, exists := newMap[name]; !exists {
			diffs = append(diffs, &diff.RuleDiff{
				Action:  diff.MetadataDiffActionDrop,
				OldRule: oldRule,
			})
		}
	}

	for name, newRule := range newMap {
		oldRule, exists := oldMap[name]
		if !exists {
			diffs = append(diffs, &diff.RuleDiff{
				Action:  diff.MetadataDiffActionCreate,
				NewRule: newRule,
			})
		} else if oldRule.Definition != newRule.Definition {
			diffs = append(diffs, &diff.RuleDiff{
				Action:  diff.MetadataDiffActionAlter,
				OldRule: oldRule,
				NewRule: newRule,
			})
		}
	}

	return diffs
}

// compareViews compares views between old and new schemas.
func compareViewsDetailed(oldViews, newViews []*schemaextract.View, schemaName string, comparer engine.Comparer, opts *CompareOptions) ([]*diff.ViewDiff, error) {
	var diffs []*diff.ViewDiff

	// Build maps
	oldMap := make(map[string]*schemaextract.View)
	for _, v := range oldViews {
		oldMap[v.Name] = v
	}

	newMap := make(map[string]*schemaextract.View)
	for _, v := range newViews {
		newMap[v.Name] = v
	}

	// Check for dropped views
	for name, oldView := range oldMap {
		if _, exists := newMap[name]; !exists {
			diffs = append(diffs, &diff.ViewDiff{
				Action:     diff.MetadataDiffActionDrop,
				SchemaName: schemaName,
				ViewName:   name,
				OldView:    oldView,
			})
		}
	}

	// Check for new and modified views
	for name, newView := range newMap {
		oldView, exists := oldMap[name]
		if !exists {
			diffs = append(diffs, &diff.ViewDiff{
				Action:     diff.MetadataDiffActionCreate,
				SchemaName: schemaName,
				ViewName:   name,
				NewView:    newView,
			})
		} else {
			// Compare view details
			result, err := comparer.View().CompareView(oldView, newView)
			if err != nil {
				return nil, err
			}

			if result.DefinitionChanged || result.CommentChanged {
				diffs = append(diffs, &diff.ViewDiff{
					Action:            diff.MetadataDiffActionAlter,
					SchemaName:        schemaName,
					ViewName:          name,
					OldView:           oldView,
					NewView:           newView,
					DefinitionChanged: result.DefinitionChanged,
				})
			}
		}
	}

	return diffs, nil
}

func compareMaterializedViews(oldMVs, newMVs []*schemaextract.MaterializedView, schemaName string, comparer engine.Comparer, opts *CompareOptions) ([]*diff.MaterializedViewDiff, error) {
	var diffs []*diff.MaterializedViewDiff

	// Build maps
	oldMap := make(map[string]*schemaextract.MaterializedView)
	for _, mv := range oldMVs {
		oldMap[mv.Name] = mv
	}

	newMap := make(map[string]*schemaextract.MaterializedView)
	for _, mv := range newMVs {
		newMap[mv.Name] = mv
	}

	// Check for dropped materialized views
	for name, oldMV := range oldMap {
		if _, exists := newMap[name]; !exists {
			diffs = append(diffs, &diff.MaterializedViewDiff{
				Action:     diff.MetadataDiffActionDrop,
				SchemaName: schemaName,
				ViewName:   name,
				OldView:    oldMV,
			})
		}
	}

	// Check for new and modified materialized views
	for name, newMV := range newMap {
		oldMV, exists := oldMap[name]
		if !exists {
			diffs = append(diffs, &diff.MaterializedViewDiff{
				Action:     diff.MetadataDiffActionCreate,
				SchemaName: schemaName,
				ViewName:   name,
				NewView:    newMV,
			})
		} else {
			// Compare materialized view details
			result, err := comparer.View().CompareMaterializedView(oldMV, newMV)
			if err != nil {
				return nil, err
			}

			if result.DefinitionChanged || result.CommentChanged {
				diffs = append(diffs, &diff.MaterializedViewDiff{
					Action:            diff.MetadataDiffActionAlter,
					SchemaName:        schemaName,
					ViewName:          name,
					OldView:           oldMV,
					NewView:           newMV,
					DefinitionChanged: result.DefinitionChanged,
				})
			}
		}
	}

	return diffs, nil
}

func compareFunctionsDetailed(oldFuncs, newFuncs []*schemaextract.Function, schemaName string, comparer engine.Comparer, opts *CompareOptions) ([]*diff.FunctionDiff, error) {
	var diffs []*diff.FunctionDiff

	// Group functions by signature to handle overloading
	oldMap := groupFunctionsBySignature(oldFuncs, comparer.Function())
	newMap := groupFunctionsBySignature(newFuncs, comparer.Function())

	// Check for dropped functions
	for sig, oldFunc := range oldMap {
		if _, exists := newMap[sig]; !exists {
			diffs = append(diffs, &diff.FunctionDiff{
				Action:       diff.MetadataDiffActionDrop,
				SchemaName:   schemaName,
				FunctionName: oldFunc.Name,
				OldFunction:  oldFunc,
			})
		}
	}

	// Check for new and modified functions
	for sig, newFunc := range newMap {
		oldFunc, exists := oldMap[sig]
		if !exists {
			diffs = append(diffs, &diff.FunctionDiff{
				Action:       diff.MetadataDiffActionCreate,
				SchemaName:   schemaName,
				FunctionName: newFunc.Name,
				NewFunction:  newFunc,
			})
		} else if !comparer.Function().Equal(oldFunc, newFunc) {
			// Compare function details
			result, err := comparer.Function().CompareDetailed(oldFunc, newFunc)
			if err != nil {
				return nil, err
			}

			diffs = append(diffs, &diff.FunctionDiff{
				Action:              diff.MetadataDiffActionAlter,
				SchemaName:          schemaName,
				FunctionName:        newFunc.Name,
				OldFunction:         oldFunc,
				NewFunction:         newFunc,
				SignatureChanged:    result.SignatureChanged,
				BodyChanged:         result.BodyChanged,
				AttributesChanged:   result.AttributesChanged,
				CanUseAlterFunction: result.CanUseAlterFunction,
			})
		}
	}

	return diffs, nil
}

// groupFunctionsBySignature groups functions by their signature (name + parameters).
func groupFunctionsBySignature(functions []*schemaextract.Function, comparer engine.FunctionComparer) map[string]*schemaextract.Function {
	result := make(map[string]*schemaextract.Function)
	for _, fn := range functions {
		sig := comparer.GetSignature(fn)
		result[sig] = fn
	}
	return result
}

func compareProcedures(oldProcs, newProcs []*schemaextract.Procedure, schemaName string, comparer engine.Comparer, opts *CompareOptions) []*diff.ProcedureDiff {
	var diffs []*diff.ProcedureDiff

	// Build maps
	oldMap := make(map[string]*schemaextract.Procedure)
	for _, p := range oldProcs {
		oldMap[p.Name] = p
	}

	newMap := make(map[string]*schemaextract.Procedure)
	for _, p := range newProcs {
		newMap[p.Name] = p
	}

	// Check for dropped procedures
	for name, oldProc := range oldMap {
		if _, exists := newMap[name]; !exists {
			diffs = append(diffs, &diff.ProcedureDiff{
				Action:        diff.MetadataDiffActionDrop,
				SchemaName:    schemaName,
				ProcedureName: name,
				OldProcedure:  oldProc,
			})
		}
	}

	// Check for new and modified procedures
	for name, newProc := range newMap {
		oldProc, exists := oldMap[name]
		if !exists {
			diffs = append(diffs, &diff.ProcedureDiff{
				Action:        diff.MetadataDiffActionCreate,
				SchemaName:    schemaName,
				ProcedureName: name,
				NewProcedure:  newProc,
			})
		} else {
			// Compare procedure details
			result, err := comparer.Procedure().CompareProcedures(oldProc, newProc)
			if err != nil {
				continue // Skip on error
			}

			if result != nil && result.Action == diff.MetadataDiffActionAlter {
				diffs = append(diffs, &diff.ProcedureDiff{
					Action:               diff.MetadataDiffActionAlter,
					SchemaName:           schemaName,
					ProcedureName:        name,
					OldProcedure:         oldProc,
					NewProcedure:         newProc,
					SignatureChanged:     result.SignatureChanged,
					BodyChanged:          result.BodyChanged,
					CanUseAlterProcedure: result.CanUseAlterProcedure,
				})
			}
		}
	}

	return diffs
}

func compareSequences(oldSeqs, newSeqs []*schemaextract.Sequence, schemaName string, comparer engine.Comparer, opts *CompareOptions) []*diff.SequenceDiff {
	var diffs []*diff.SequenceDiff

	// Build maps
	oldMap := make(map[string]*schemaextract.Sequence)
	for _, s := range oldSeqs {
		oldMap[s.Name] = s
	}

	newMap := make(map[string]*schemaextract.Sequence)
	for _, s := range newSeqs {
		newMap[s.Name] = s
	}

	// Check for dropped sequences
	for name, oldSeq := range oldMap {
		if _, exists := newMap[name]; !exists {
			diffs = append(diffs, &diff.SequenceDiff{
				Action:       diff.MetadataDiffActionDrop,
				SchemaName:   schemaName,
				SequenceName: name,
				OldSequence:  oldSeq,
			})
		}
	}

	// Check for new and modified sequences
	for name, newSeq := range newMap {
		oldSeq, exists := oldMap[name]
		if !exists {
			diffs = append(diffs, &diff.SequenceDiff{
				Action:       diff.MetadataDiffActionCreate,
				SchemaName:   schemaName,
				SequenceName: name,
				NewSequence:  newSeq,
			})
		} else {
			// Compare sequence details
			result, err := comparer.Sequence().CompareSequences(oldSeq, newSeq)
			if err != nil {
				continue // Skip on error
			}

			if result != nil && result.Action == diff.MetadataDiffActionAlter {
				diffs = append(diffs, result)
				result.SchemaName = schemaName
				result.SequenceName = name
			}
		}
	}

	return diffs
}

func compareEnumTypes(oldEnums, newEnums []*schemaextract.EnumType, schemaName string, opts *CompareOptions) []*diff.EnumTypeDiff {
	var diffs []*diff.EnumTypeDiff

	// Build maps
	oldMap := make(map[string]*schemaextract.EnumType)
	for _, e := range oldEnums {
		oldMap[e.Name] = e
	}

	newMap := make(map[string]*schemaextract.EnumType)
	for _, e := range newEnums {
		newMap[e.Name] = e
	}

	// Check for dropped enum types
	for name, oldEnum := range oldMap {
		if _, exists := newMap[name]; !exists {
			diffs = append(diffs, &diff.EnumTypeDiff{
				Action:      diff.MetadataDiffActionDrop,
				SchemaName:  schemaName,
				EnumName:    name,
				OldEnumType: oldEnum,
			})
		}
	}

	// Check for new and modified enum types
	for name, newEnum := range newMap {
		oldEnum, exists := oldMap[name]
		if !exists {
			diffs = append(diffs, &diff.EnumTypeDiff{
				Action:      diff.MetadataDiffActionCreate,
				SchemaName:  schemaName,
				EnumName:    name,
				NewEnumType: newEnum,
			})
		} else {
			// Compare enum values
			added, removed, orderChanged := compareEnumValues(oldEnum.Values, newEnum.Values)
			if len(added) > 0 || len(removed) > 0 || orderChanged {
				diffs = append(diffs, &diff.EnumTypeDiff{
					Action:        diff.MetadataDiffActionAlter,
					SchemaName:    schemaName,
					EnumName:      name,
					OldEnumType:   oldEnum,
					NewEnumType:   newEnum,
					AddedValues:   added,
					RemovedValues: removed,
					OrderChanged:  orderChanged,
				})
			}
		}
	}

	return diffs
}

// compareEnumValues compares two lists of enum values.
func compareEnumValues(oldValues, newValues []string) (added, removed []string, orderChanged bool) {
	oldSet := make(map[string]bool)
	for _, v := range oldValues {
		oldSet[v] = true
	}

	newSet := make(map[string]bool)
	for _, v := range newValues {
		newSet[v] = true
	}

	// Find added values
	for _, v := range newValues {
		if !oldSet[v] {
			added = append(added, v)
		}
	}

	// Find removed values
	for _, v := range oldValues {
		if !newSet[v] {
			removed = append(removed, v)
		}
	}

	// Check if order changed (only for values that exist in both)
	if len(added) == 0 && len(removed) == 0 {
		if len(oldValues) != len(newValues) {
			orderChanged = true
		} else {
			for i := range oldValues {
				if oldValues[i] != newValues[i] {
					orderChanged = true
					break
				}
			}
		}
	}

	return
}

func compareEvents(oldEvents, newEvents []*schemaextract.Event, opts *CompareOptions) []*diff.EventDiff {
	var diffs []*diff.EventDiff

	// Build maps
	oldMap := make(map[string]*schemaextract.Event)
	for _, e := range oldEvents {
		oldMap[e.Name] = e
	}

	newMap := make(map[string]*schemaextract.Event)
	for _, e := range newEvents {
		newMap[e.Name] = e
	}

	// Check for dropped events
	for name, oldEvent := range oldMap {
		if _, exists := newMap[name]; !exists {
			diffs = append(diffs, &diff.EventDiff{
				Action:    diff.MetadataDiffActionDrop,
				EventName: name,
				OldEvent:  oldEvent,
			})
		}
	}

	// Check for new and modified events
	for name, newEvent := range newMap {
		oldEvent, exists := oldMap[name]
		if !exists {
			diffs = append(diffs, &diff.EventDiff{
				Action:    diff.MetadataDiffActionCreate,
				EventName: name,
				NewEvent:  newEvent,
			})
		} else {
			// Compare event definitions
			if oldEvent.Definition != newEvent.Definition {
				diffs = append(diffs, &diff.EventDiff{
					Action:            diff.MetadataDiffActionAlter,
					EventName:         name,
					OldEvent:          oldEvent,
					NewEvent:          newEvent,
					DefinitionChanged: true,
				})
			}
		}
	}

	return diffs
}

func compareExtensions(oldExts, newExts []*schemaextract.Extension, schemaName string, opts *CompareOptions) []*diff.ExtensionDiff {
	var diffs []*diff.ExtensionDiff

	// Build maps
	oldMap := make(map[string]*schemaextract.Extension)
	for _, e := range oldExts {
		oldMap[e.Name] = e
	}

	newMap := make(map[string]*schemaextract.Extension)
	for _, e := range newExts {
		newMap[e.Name] = e
	}

	// Check for dropped extensions
	for name, oldExt := range oldMap {
		if _, exists := newMap[name]; !exists {
			diffs = append(diffs, &diff.ExtensionDiff{
				Action:        diff.MetadataDiffActionDrop,
				SchemaName:    schemaName,
				ExtensionName: name,
				OldExtension:  oldExt,
			})
		}
	}

	// Check for new and modified extensions
	for name, newExt := range newMap {
		oldExt, exists := oldMap[name]
		if !exists {
			diffs = append(diffs, &diff.ExtensionDiff{
				Action:        diff.MetadataDiffActionCreate,
				SchemaName:    schemaName,
				ExtensionName: name,
				NewExtension:  newExt,
			})
		} else {
			// Compare extension versions
			if oldExt.Version != newExt.Version {
				diffs = append(diffs, &diff.ExtensionDiff{
					Action:         diff.MetadataDiffActionAlter,
					SchemaName:     schemaName,
					ExtensionName:  name,
					OldExtension:   oldExt,
					NewExtension:   newExt,
					VersionChanged: true,
				})
			}
		}
	}

	return diffs
}

// Helper functions

func findSchema(schemas []*schemaextract.Schema, name string) *schemaextract.Schema {
	for _, s := range schemas {
		if s.Name == name {
			return s
		}
	}
	return nil
}

// sortMetadataDiff sorts all diffs for deterministic output.
func sortMetadataDiff(mdiff *diff.MetadataDiff) {
	slices.SortFunc(mdiff.SchemaChanges, func(a, b *diff.SchemaDiff) int {
		if a.SchemaName < b.SchemaName {
			return -1
		} else if a.SchemaName > b.SchemaName {
			return 1
		}
		return 0
	})
	slices.SortFunc(mdiff.TableChanges, func(a, b *diff.TableDiff) int {
		if a.SchemaName != b.SchemaName {
			if a.SchemaName < b.SchemaName {
				return -1
			}
			return 1
		}
		if a.TableName < b.TableName {
			return -1
		} else if a.TableName > b.TableName {
			return 1
		}
		return 0
	})
	// Sort other diff types similarly...
}
