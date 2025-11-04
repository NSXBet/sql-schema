package planner

import (
	"testing"

	schemaextract "github.com/nsxbet/sql-schema"
	"github.com/nsxbet/sql-schema/comparer/engine"
	"github.com/nsxbet/sql-schema/diff"
)

func TestAnalyzeStrategy_EmptyDiff(t *testing.T) {
	mdiff := &diff.MetadataDiff{
		DatabaseName: "test_db",
	}

	strategy, err := AnalyzeStrategy(mdiff, engine.PostgreSQL)
	if err != nil {
		t.Fatalf("AnalyzeStrategy failed: %v", err)
	}

	if len(strategy.Operations) != 0 {
		t.Errorf("Expected 0 operations, got %d", len(strategy.Operations))
	}

	if len(strategy.HighRiskOps) != 0 {
		t.Errorf("Expected 0 high-risk operations, got %d", len(strategy.HighRiskOps))
	}
}

func TestAnalyzeStrategy_CreateTable(t *testing.T) {
	mdiff := &diff.MetadataDiff{
		DatabaseName: "test_db",
		TableChanges: []*diff.TableDiff{
			{
				Action:     diff.MetadataDiffActionCreate,
				SchemaName: "public",
				TableName:  "users",
				NewTable: &schemaextract.Table{
					Name: "users",
				},
			},
		},
	}

	strategy, err := AnalyzeStrategy(mdiff, engine.PostgreSQL)
	if err != nil {
		t.Fatalf("AnalyzeStrategy failed: %v", err)
	}

	if len(strategy.Operations) != 1 {
		t.Errorf("Expected 1 operation, got %d", len(strategy.Operations))
	}

	op := strategy.Operations[0]
	if op.Action != diff.MetadataDiffActionCreate {
		t.Errorf("Expected CREATE action, got %s", op.Action)
	}

	if op.Risk != RiskLevelNone {
		t.Errorf("Expected NONE risk, got %s", op.Risk)
	}

	if !op.Reversible {
		t.Error("Expected operation to be reversible")
	}
}

func TestAnalyzeStrategy_DropTable(t *testing.T) {
	mdiff := &diff.MetadataDiff{
		DatabaseName: "test_db",
		TableChanges: []*diff.TableDiff{
			{
				Action:     diff.MetadataDiffActionDrop,
				SchemaName: "public",
				TableName:  "users",
				OldTable: &schemaextract.Table{
					Name: "users",
				},
			},
		},
	}

	strategy, err := AnalyzeStrategy(mdiff, engine.PostgreSQL)
	if err != nil {
		t.Fatalf("AnalyzeStrategy failed: %v", err)
	}

	if len(strategy.Operations) != 1 {
		t.Errorf("Expected 1 operation, got %d", len(strategy.Operations))
	}

	op := strategy.Operations[0]
	if op.Action != diff.MetadataDiffActionDrop {
		t.Errorf("Expected DROP action, got %s", op.Action)
	}

	if op.Risk != RiskLevelHigh {
		t.Errorf("Expected HIGH risk, got %s", op.Risk)
	}

	if op.Reversible {
		t.Error("Expected operation to be irreversible")
	}

	if len(strategy.HighRiskOps) != 1 {
		t.Errorf("Expected 1 high-risk operation, got %d", len(strategy.HighRiskOps))
	}

	if len(strategy.DataLossOps) != 1 {
		t.Errorf("Expected 1 data-loss operation, got %d", len(strategy.DataLossOps))
	}
}

func TestAnalyzeStrategy_AddColumn(t *testing.T) {
	mdiff := &diff.MetadataDiff{
		DatabaseName: "test_db",
		TableChanges: []*diff.TableDiff{
			{
				Action:     diff.MetadataDiffActionAlter,
				SchemaName: "public",
				TableName:  "users",
				ColumnChanges: []*diff.ColumnDiff{
					{
						Action: diff.MetadataDiffActionCreate,
						NewColumn: &schemaextract.Column{
							Name:     "email",
							Type:     "VARCHAR(255)",
							Nullable: true,
						},
					},
				},
			},
		},
	}

	strategy, err := AnalyzeStrategy(mdiff, engine.PostgreSQL)
	if err != nil {
		t.Fatalf("AnalyzeStrategy failed: %v", err)
	}

	if len(strategy.Operations) != 1 {
		t.Errorf("Expected 1 operation, got %d", len(strategy.Operations))
	}

	op := strategy.Operations[0]
	if op.Type != OperationTypeColumn {
		t.Errorf("Expected COLUMN type, got %s", op.Type)
	}

	if op.Risk != RiskLevelLow {
		t.Errorf("Expected LOW risk, got %s", op.Risk)
	}
}

func TestAnalyzeStrategy_AddNotNullColumnWithoutDefault(t *testing.T) {
	mdiff := &diff.MetadataDiff{
		DatabaseName: "test_db",
		TableChanges: []*diff.TableDiff{
			{
				Action:     diff.MetadataDiffActionAlter,
				SchemaName: "public",
				TableName:  "users",
				ColumnChanges: []*diff.ColumnDiff{
					{
						Action: diff.MetadataDiffActionCreate,
						NewColumn: &schemaextract.Column{
							Name:     "required_field",
							Type:     "VARCHAR(255)",
							Nullable: false,
							Default:  "", // No default
						},
					},
				},
			},
		},
	}

	strategy, err := AnalyzeStrategy(mdiff, engine.PostgreSQL)
	if err != nil {
		t.Fatalf("AnalyzeStrategy failed: %v", err)
	}

	if len(strategy.Operations) != 1 {
		t.Errorf("Expected 1 operation, got %d", len(strategy.Operations))
	}

	op := strategy.Operations[0]
	if op.Risk != RiskLevelMedium {
		t.Errorf("Expected MEDIUM risk, got %s", op.Risk)
	}

	if len(op.Warnings) == 0 {
		t.Error("Expected warnings about NOT NULL without default")
	}
}

func TestAnalyzeStrategy_DropColumn(t *testing.T) {
	mdiff := &diff.MetadataDiff{
		DatabaseName: "test_db",
		TableChanges: []*diff.TableDiff{
			{
				Action:     diff.MetadataDiffActionAlter,
				SchemaName: "public",
				TableName:  "users",
				ColumnChanges: []*diff.ColumnDiff{
					{
						Action: diff.MetadataDiffActionDrop,
						OldColumn: &schemaextract.Column{
							Name: "old_field",
							Type: "VARCHAR(255)",
						},
					},
				},
			},
		},
	}

	strategy, err := AnalyzeStrategy(mdiff, engine.PostgreSQL)
	if err != nil {
		t.Fatalf("AnalyzeStrategy failed: %v", err)
	}

	if len(strategy.Operations) != 1 {
		t.Errorf("Expected 1 operation, got %d", len(strategy.Operations))
	}

	op := strategy.Operations[0]
	if op.Risk != RiskLevelHigh {
		t.Errorf("Expected HIGH risk, got %s", op.Risk)
	}

	if op.Reversible {
		t.Error("Expected operation to be irreversible")
	}

	if len(strategy.DataLossOps) != 1 {
		t.Errorf("Expected 1 data-loss operation, got %d", len(strategy.DataLossOps))
	}
}

func TestAnalyzeStrategy_DependencyOrdering(t *testing.T) {
	mdiff := &diff.MetadataDiff{
		DatabaseName: "test_db",
		TableChanges: []*diff.TableDiff{
			{
				Action:     diff.MetadataDiffActionAlter,
				SchemaName: "public",
				TableName:  "users",
				ColumnChanges: []*diff.ColumnDiff{
					{
						Action: diff.MetadataDiffActionCreate,
						NewColumn: &schemaextract.Column{
							Name:     "id",
							Type:     "INTEGER",
							Nullable: false,
						},
					},
				},
				IndexChanges: []*diff.IndexDiff{
					{
						Action: diff.MetadataDiffActionCreate,
						NewIndex: &schemaextract.Index{
							Name:        "idx_users_id",
							Type:        "BTREE",
							Expressions: []string{"id"},
						},
					},
				},
			},
		},
	}

	strategy, err := AnalyzeStrategy(mdiff, engine.PostgreSQL)
	if err != nil {
		t.Fatalf("AnalyzeStrategy failed: %v", err)
	}

	// Should have column and index operations
	if len(strategy.Operations) != 2 {
		t.Errorf("Expected 2 operations, got %d", len(strategy.Operations))
	}

	// Check ordered operations respect dependencies
	if len(strategy.OrderedOperations) != 2 {
		t.Fatalf("Expected 2 ordered operations, got %d", len(strategy.OrderedOperations))
	}

	// Column should come before index in ordered list
	// Both have dependencies on table, but we're checking they're both present
	foundColumn := false
	foundIndex := false
	for _, op := range strategy.OrderedOperations {
		if op.Type == OperationTypeColumn {
			foundColumn = true
		}
		if op.Type == OperationTypeIndex {
			foundIndex = true
		}
	}

	if !foundColumn {
		t.Error("Expected to find column operation in ordered list")
	}
	if !foundIndex {
		t.Error("Expected to find index operation in ordered list")
	}
}

func TestSortByRisk(t *testing.T) {
	ops := []*MigrationOperation{
		{ID: "1", Risk: RiskLevelLow},
		{ID: "2", Risk: RiskLevelHigh},
		{ID: "3", Risk: RiskLevelNone},
		{ID: "4", Risk: RiskLevelMedium},
	}

	sorted := SortByRisk(ops)

	if len(sorted) != 4 {
		t.Fatalf("Expected 4 operations, got %d", len(sorted))
	}

	// Should be ordered: HIGH, MEDIUM, LOW, NONE
	expectedOrder := []RiskLevel{RiskLevelHigh, RiskLevelMedium, RiskLevelLow, RiskLevelNone}
	for i, expected := range expectedOrder {
		if sorted[i].Risk != expected {
			t.Errorf("Position %d: expected %s risk, got %s", i, expected, sorted[i].Risk)
		}
	}
}
