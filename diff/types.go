package diff

import (
	schemaextract "github.com/nsxbet/sql-schema"
)

// MetadataDiffAction represents the type of change.
type MetadataDiffAction string

const (
	// MetadataDiffActionCreate indicates an object was created.
	MetadataDiffActionCreate MetadataDiffAction = "CREATE"
	// MetadataDiffActionDrop indicates an object was dropped.
	MetadataDiffActionDrop MetadataDiffAction = "DROP"
	// MetadataDiffActionAlter indicates an object was altered.
	MetadataDiffActionAlter MetadataDiffAction = "ALTER"
)

// MetadataDiff represents the complete diff between two database schemas.
type MetadataDiff struct {
	DatabaseName string

	// Top-level object changes
	SchemaChanges           []*SchemaDiff
	TableChanges            []*TableDiff
	ViewChanges             []*ViewDiff
	MaterializedViewChanges []*MaterializedViewDiff
	FunctionChanges         []*FunctionDiff
	ProcedureChanges        []*ProcedureDiff
	SequenceChanges         []*SequenceDiff
	EnumTypeChanges         []*EnumTypeDiff
	EventChanges            []*EventDiff
	ExtensionChanges        []*ExtensionDiff
	CommentChanges          []*CommentDiff
}

// SchemaDiff represents changes to a schema.
type SchemaDiff struct {
	Action     MetadataDiffAction
	SchemaName string
	OldSchema  *schemaextract.Schema
	NewSchema  *schemaextract.Schema
}

// TableDiff represents changes to a table.
type TableDiff struct {
	Action     MetadataDiffAction
	SchemaName string
	TableName  string
	OldTable   *schemaextract.Table
	NewTable   *schemaextract.Table

	// Sub-object changes (only for ALTER action)
	ColumnChanges           []*ColumnDiff
	IndexChanges            []*IndexDiff
	PrimaryKeyChanges       []*PrimaryKeyDiff
	UniqueConstraintChanges []*UniqueConstraintDiff
	ForeignKeyChanges       []*ForeignKeyDiff
	CheckConstraintChanges  []*CheckConstraintDiff
	PartitionChanges        []*PartitionDiff
	TriggerChanges          []*TriggerDiff
	RuleChanges             []*RuleDiff

	// Property changes
	CommentChanged   bool
	EngineChanged    bool
	CollationChanged bool
	CharsetChanged   bool
}

// ColumnDiff represents changes to a column.
type ColumnDiff struct {
	Action    MetadataDiffAction
	OldColumn *schemaextract.Column
	NewColumn *schemaextract.Column
}

// IndexDiff represents changes to an index.
type IndexDiff struct {
	Action   MetadataDiffAction
	OldIndex *schemaextract.Index
	NewIndex *schemaextract.Index
}

// PrimaryKeyDiff represents changes to a primary key.
type PrimaryKeyDiff struct {
	Action        MetadataDiffAction
	OldPrimaryKey *schemaextract.Index // Primary key stored as Index with Primary=true
	NewPrimaryKey *schemaextract.Index
}

// UniqueConstraintDiff represents changes to a unique constraint.
type UniqueConstraintDiff struct {
	Action        MetadataDiffAction
	OldConstraint *schemaextract.Index // Unique constraint stored as Index with Unique=true
	NewConstraint *schemaextract.Index
}

// ForeignKeyDiff represents changes to a foreign key.
type ForeignKeyDiff struct {
	Action        MetadataDiffAction
	OldForeignKey *schemaextract.ForeignKey
	NewForeignKey *schemaextract.ForeignKey
}

// CheckConstraintDiff represents changes to a check constraint.
type CheckConstraintDiff struct {
	Action        MetadataDiffAction
	OldConstraint *schemaextract.CheckConstraint
	NewConstraint *schemaextract.CheckConstraint
}

// PartitionDiff represents changes to a partition.
type PartitionDiff struct {
	Action       MetadataDiffAction
	OldPartition *schemaextract.Partition
	NewPartition *schemaextract.Partition
}

// TriggerDiff represents changes to a trigger.
type TriggerDiff struct {
	Action     MetadataDiffAction
	OldTrigger *schemaextract.Trigger
	NewTrigger *schemaextract.Trigger
}

// RuleDiff represents changes to a rule (PostgreSQL).
type RuleDiff struct {
	Action  MetadataDiffAction
	OldRule *schemaextract.Rule
	NewRule *schemaextract.Rule
}

// ViewDiff represents changes to a view.
type ViewDiff struct {
	Action     MetadataDiffAction
	SchemaName string
	ViewName   string
	OldView    *schemaextract.View
	NewView    *schemaextract.View

	// View-specific properties
	DefinitionChanged bool
}

// MaterializedViewDiff represents changes to a materialized view.
type MaterializedViewDiff struct {
	Action     MetadataDiffAction
	SchemaName string
	ViewName   string
	OldView    *schemaextract.MaterializedView
	NewView    *schemaextract.MaterializedView

	// Materialized view-specific properties
	DefinitionChanged bool
}

// FunctionDiff represents changes to a function.
type FunctionDiff struct {
	Action       MetadataDiffAction
	SchemaName   string
	FunctionName string
	OldFunction  *schemaextract.Function
	NewFunction  *schemaextract.Function

	// Function-specific properties
	SignatureChanged    bool // Requires DROP/CREATE
	BodyChanged         bool // Can use ALTER FUNCTION
	AttributesChanged   bool // Language, volatility, etc.
	CanUseAlterFunction bool // Optimization flag
}

// ProcedureDiff represents changes to a stored procedure.
type ProcedureDiff struct {
	Action        MetadataDiffAction
	SchemaName    string
	ProcedureName string
	OldProcedure  *schemaextract.Procedure
	NewProcedure  *schemaextract.Procedure

	// Procedure-specific properties
	SignatureChanged     bool
	BodyChanged          bool
	CanUseAlterProcedure bool
}

// SequenceDiff represents changes to a sequence.
type SequenceDiff struct {
	Action       MetadataDiffAction
	SchemaName   string
	SequenceName string
	OldSequence  *schemaextract.Sequence
	NewSequence  *schemaextract.Sequence

	// Sequence-specific properties
	StartChanged     bool
	IncrementChanged bool
	MinValueChanged  bool
	MaxValueChanged  bool
	CacheChanged     bool
	CycleChanged     bool
}

// EnumTypeDiff represents changes to an enum type.
type EnumTypeDiff struct {
	Action      MetadataDiffAction
	SchemaName  string
	EnumName    string
	OldEnumType *schemaextract.EnumType
	NewEnumType *schemaextract.EnumType

	// Enum-specific properties
	AddedValues   []string
	RemovedValues []string
	OrderChanged  bool
}

// EventDiff represents changes to an event (MySQL).
type EventDiff struct {
	Action    MetadataDiffAction
	EventName string
	OldEvent  *schemaextract.Event
	NewEvent  *schemaextract.Event

	// Event-specific properties
	DefinitionChanged bool
}

// ExtensionDiff represents changes to an extension (PostgreSQL).
type ExtensionDiff struct {
	Action        MetadataDiffAction
	SchemaName    string
	ExtensionName string
	OldExtension  *schemaextract.Extension
	NewExtension  *schemaextract.Extension

	// Extension-specific properties
	VersionChanged bool
}

// CommentDiff represents changes to object comments.
type CommentDiff struct {
	ObjectType string // TABLE, COLUMN, VIEW, etc.
	SchemaName string
	ObjectName string
	ColumnName string // For column comments
	OldComment string
	NewComment string
}

// IsEmpty returns true if there are no differences.
func (d *MetadataDiff) IsEmpty() bool {
	return len(d.SchemaChanges) == 0 &&
		len(d.TableChanges) == 0 &&
		len(d.ViewChanges) == 0 &&
		len(d.MaterializedViewChanges) == 0 &&
		len(d.FunctionChanges) == 0 &&
		len(d.ProcedureChanges) == 0 &&
		len(d.SequenceChanges) == 0 &&
		len(d.EnumTypeChanges) == 0 &&
		len(d.EventChanges) == 0 &&
		len(d.ExtensionChanges) == 0 &&
		len(d.CommentChanges) == 0
}

// HasTableChanges returns true if the table has any changes.
func (d *TableDiff) HasChanges() bool {
	if d.Action != MetadataDiffActionAlter {
		return true
	}

	return len(d.ColumnChanges) > 0 ||
		len(d.IndexChanges) > 0 ||
		len(d.PrimaryKeyChanges) > 0 ||
		len(d.UniqueConstraintChanges) > 0 ||
		len(d.ForeignKeyChanges) > 0 ||
		len(d.CheckConstraintChanges) > 0 ||
		len(d.PartitionChanges) > 0 ||
		len(d.TriggerChanges) > 0 ||
		len(d.RuleChanges) > 0 ||
		d.CommentChanged ||
		d.EngineChanged ||
		d.CollationChanged ||
		d.CharsetChanged
}

// CountChanges returns the total number of changes across all categories.
func (d *MetadataDiff) CountChanges() int {
	count := len(d.SchemaChanges) +
		len(d.TableChanges) +
		len(d.ViewChanges) +
		len(d.MaterializedViewChanges) +
		len(d.FunctionChanges) +
		len(d.ProcedureChanges) +
		len(d.SequenceChanges) +
		len(d.EnumTypeChanges) +
		len(d.EventChanges) +
		len(d.ExtensionChanges) +
		len(d.CommentChanges)

	// Add sub-object changes from tables
	for _, tableDiff := range d.TableChanges {
		if tableDiff.Action == MetadataDiffActionAlter {
			count += len(tableDiff.ColumnChanges) +
				len(tableDiff.IndexChanges) +
				len(tableDiff.PrimaryKeyChanges) +
				len(tableDiff.UniqueConstraintChanges) +
				len(tableDiff.ForeignKeyChanges) +
				len(tableDiff.CheckConstraintChanges) +
				len(tableDiff.PartitionChanges) +
				len(tableDiff.TriggerChanges) +
				len(tableDiff.RuleChanges)
		}
	}

	return count
}

// FunctionComparisonResult provides detailed information about function changes.
type FunctionComparisonResult struct {
	SignatureChanged    bool
	BodyChanged         bool
	AttributesChanged   bool
	CanUseAlterFunction bool
	ChangedAttributes   []string // List of attribute names that changed
}

// ViewComparisonResult provides detailed information about view changes.
type ViewComparisonResult struct {
	DefinitionChanged  bool
	RequiresRecreation bool // Some changes require DROP/CREATE instead of ALTER
	ColumnsChanged     bool
	CommentChanged     bool
}

// MaterializedViewComparisonResult provides detailed information about materialized view changes.
type MaterializedViewComparisonResult struct {
	DefinitionChanged  bool
	RequiresRecreation bool
	ColumnsChanged     bool
	CommentChanged     bool
}
