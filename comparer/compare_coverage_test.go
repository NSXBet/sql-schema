package comparer

import (
	"testing"

	schemaextract "github.com/nsxbet/sql-schema"
	"github.com/nsxbet/sql-schema/comparer/engine"
	"github.com/nsxbet/sql-schema/diff"
)

// TestDefaultCompareOptions tests the default options constructor.
func TestDefaultCompareOptions(t *testing.T) {
	opts := DefaultCompareOptions(engine.PostgreSQL)

	if opts.Engine != engine.PostgreSQL {
		t.Errorf("Expected engine PostgreSQL, got %s", opts.Engine)
	}
	if opts.IgnoreComments {
		t.Error("Expected IgnoreComments to be false by default")
	}
	if opts.IgnoreCharset {
		t.Error("Expected IgnoreCharset to be false by default")
	}
	if opts.IgnoreCollation {
		t.Error("Expected IgnoreCollation to be false by default")
	}
}

// TestCompareForeignKeys tests foreign key comparison with various scenarios.
func TestCompareForeignKeys(t *testing.T) {
	tests := []struct {
		name     string
		oldFKs   []*schemaextract.ForeignKey
		newFKs   []*schemaextract.ForeignKey
		expected int // number of diffs
	}{
		{
			name:     "no foreign keys",
			oldFKs:   []*schemaextract.ForeignKey{},
			newFKs:   []*schemaextract.ForeignKey{},
			expected: 0,
		},
		{
			name:   "add foreign key",
			oldFKs: []*schemaextract.ForeignKey{},
			newFKs: []*schemaextract.ForeignKey{
				{
					Name:              "fk_user_id",
					Columns:           []string{"user_id"},
					ReferencedTable:   "users",
					ReferencedColumns: []string{"id"},
				},
			},
			expected: 1,
		},
		{
			name: "drop foreign key",
			oldFKs: []*schemaextract.ForeignKey{
				{
					Name:              "fk_user_id",
					Columns:           []string{"user_id"},
					ReferencedTable:   "users",
					ReferencedColumns: []string{"id"},
				},
			},
			newFKs:   []*schemaextract.ForeignKey{},
			expected: 1,
		},
		{
			name: "identical foreign keys",
			oldFKs: []*schemaextract.ForeignKey{
				{
					Name:              "fk_user_id",
					Columns:           []string{"user_id"},
					ReferencedTable:   "users",
					ReferencedColumns: []string{"id"},
					OnDelete:          "CASCADE",
					OnUpdate:          "CASCADE",
				},
			},
			newFKs: []*schemaextract.ForeignKey{
				{
					Name:              "fk_user_id",
					Columns:           []string{"user_id"},
					ReferencedTable:   "users",
					ReferencedColumns: []string{"id"},
					OnDelete:          "CASCADE",
					OnUpdate:          "CASCADE",
				},
			},
			expected: 0,
		},
		{
			name: "modified foreign key - different columns",
			oldFKs: []*schemaextract.ForeignKey{
				{
					Name:              "fk_user",
					Columns:           []string{"user_id"},
					ReferencedTable:   "users",
					ReferencedColumns: []string{"id"},
				},
			},
			newFKs: []*schemaextract.ForeignKey{
				{
					Name:              "fk_user",
					Columns:           []string{"user_id", "tenant_id"},
					ReferencedTable:   "users",
					ReferencedColumns: []string{"id", "tenant_id"},
				},
			},
			expected: 1, // ALTER
		},
		{
			name: "modified foreign key - different actions",
			oldFKs: []*schemaextract.ForeignKey{
				{
					Name:              "fk_user_id",
					Columns:           []string{"user_id"},
					ReferencedTable:   "users",
					ReferencedColumns: []string{"id"},
					OnDelete:          "CASCADE",
				},
			},
			newFKs: []*schemaextract.ForeignKey{
				{
					Name:              "fk_user_id",
					Columns:           []string{"user_id"},
					ReferencedTable:   "users",
					ReferencedColumns: []string{"id"},
					OnDelete:          "SET NULL",
				},
			},
			expected: 1, // ALTER
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			diffs := compareForeignKeys(tt.oldFKs, tt.newFKs)

			if len(diffs) != tt.expected {
				t.Errorf("Expected %d diffs, got %d", tt.expected, len(diffs))
			}
		})
	}
}

// TestComparePartitions tests partition comparison.
func TestComparePartitions(t *testing.T) {
	tests := []struct {
		name     string
		oldParts []*schemaextract.Partition
		newParts []*schemaextract.Partition
		expected int
	}{
		{
			name:     "no partitions",
			oldParts: []*schemaextract.Partition{},
			newParts: []*schemaextract.Partition{},
			expected: 0,
		},
		{
			name:     "add partition",
			oldParts: []*schemaextract.Partition{},
			newParts: []*schemaextract.Partition{
				{
					Name:       "p2024",
					Expression: "2024",
				},
			},
			expected: 1,
		},
		{
			name: "drop partition",
			oldParts: []*schemaextract.Partition{
				{
					Name:       "p2023",
					Expression: "2023",
				},
			},
			newParts: []*schemaextract.Partition{},
			expected: 1,
		},
		{
			name: "identical partitions",
			oldParts: []*schemaextract.Partition{
				{
					Name:       "p2024",
					Expression: "2024",
					Subpartitions: []*schemaextract.Partition{
						{Name: "sp1", Expression: "HASH"},
					},
				},
			},
			newParts: []*schemaextract.Partition{
				{
					Name:       "p2024",
					Expression: "2024",
					Subpartitions: []*schemaextract.Partition{
						{Name: "sp1", Expression: "HASH"},
					},
				},
			},
			expected: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			diffs := comparePartitions(tt.oldParts, tt.newParts)

			if len(diffs) != tt.expected {
				t.Errorf("Expected %d diffs, got %d", tt.expected, len(diffs))
			}
		})
	}
}

// TestCompareTriggers tests trigger comparison.
func TestCompareTriggers(t *testing.T) {
	tests := []struct {
		name        string
		oldTriggers []*schemaextract.Trigger
		newTriggers []*schemaextract.Trigger
		expected    int
	}{
		{
			name:        "no triggers",
			oldTriggers: []*schemaextract.Trigger{},
			newTriggers: []*schemaextract.Trigger{},
			expected:    0,
		},
		{
			name:        "add trigger",
			oldTriggers: []*schemaextract.Trigger{},
			newTriggers: []*schemaextract.Trigger{
				{
					Name:   "audit_trigger",
					Timing: "BEFORE",
					Event:  "INSERT",
					Body:   "EXECUTE FUNCTION audit_log()",
				},
			},
			expected: 1,
		},
		{
			name: "drop trigger",
			oldTriggers: []*schemaextract.Trigger{
				{
					Name:   "audit_trigger",
					Timing: "BEFORE",
					Event:  "INSERT",
					Body:   "EXECUTE FUNCTION audit_log()",
				},
			},
			newTriggers: []*schemaextract.Trigger{},
			expected:    1,
		},
		{
			name: "identical triggers",
			oldTriggers: []*schemaextract.Trigger{
				{
					Name:   "audit_trigger",
					Timing: "AFTER",
					Event:  "UPDATE",
					Body:   "EXECUTE FUNCTION audit()",
				},
			},
			newTriggers: []*schemaextract.Trigger{
				{
					Name:   "audit_trigger",
					Timing: "AFTER",
					Event:  "UPDATE",
					Body:   "EXECUTE FUNCTION audit()",
				},
			},
			expected: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			diffs := compareTriggers(tt.oldTriggers, tt.newTriggers)

			if len(diffs) != tt.expected {
				t.Errorf("Expected %d diffs, got %d", tt.expected, len(diffs))
			}
		})
	}
}

// TestCompareCheckConstraints tests check constraint comparison.
func TestCompareCheckConstraints(t *testing.T) {
	tests := []struct {
		name     string
		oldCCs   []*schemaextract.CheckConstraint
		newCCs   []*schemaextract.CheckConstraint
		expected int
	}{
		{
			name:     "no constraints",
			oldCCs:   []*schemaextract.CheckConstraint{},
			newCCs:   []*schemaextract.CheckConstraint{},
			expected: 0,
		},
		{
			name:   "add constraint",
			oldCCs: []*schemaextract.CheckConstraint{},
			newCCs: []*schemaextract.CheckConstraint{
				{
					Name:       "check_age",
					Expression: "age >= 18",
				},
			},
			expected: 1,
		},
		{
			name: "drop constraint",
			oldCCs: []*schemaextract.CheckConstraint{
				{
					Name:       "check_age",
					Expression: "age >= 18",
				},
			},
			newCCs:   []*schemaextract.CheckConstraint{},
			expected: 1,
		},
		{
			name: "identical constraints",
			oldCCs: []*schemaextract.CheckConstraint{
				{
					Name:       "check_positive",
					Expression: "value > 0",
				},
			},
			newCCs: []*schemaextract.CheckConstraint{
				{
					Name:       "check_positive",
					Expression: "value > 0",
				},
			},
			expected: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			diffs := compareCheckConstraints(tt.oldCCs, tt.newCCs)

			if len(diffs) != tt.expected {
				t.Errorf("Expected %d diffs, got %d", tt.expected, len(diffs))
			}
		})
	}
}

// TestCompareMaterializedViews tests materialized view comparison.
func TestCompareMaterializedViews(t *testing.T) {
	oldSchema := &schemaextract.DatabaseSchema{
		Name: "test_db",
		Schemas: []*schemaextract.Schema{
			{
				Name: "public",
				MaterializedViews: []*schemaextract.MaterializedView{
					{
						Name:       "mv_summary",
						Definition: "SELECT COUNT(*) FROM users",
					},
				},
			},
		},
	}

	newSchema := &schemaextract.DatabaseSchema{
		Name: "test_db",
		Schemas: []*schemaextract.Schema{
			{
				Name: "public",
				MaterializedViews: []*schemaextract.MaterializedView{
					{
						Name:       "mv_summary",
						Definition: "SELECT COUNT(*) as total FROM users",
					},
					{
						Name:       "mv_new",
						Definition: "SELECT * FROM orders",
					},
				},
			},
		},
	}

	opts := &CompareOptions{Engine: engine.PostgreSQL}
	mdiff, err := CompareSchemasDetailed(oldSchema, newSchema, opts)
	if err != nil {
		t.Fatalf("CompareSchemasDetailed failed: %v", err)
	}

	if len(mdiff.MaterializedViewChanges) != 2 {
		t.Errorf("Expected 2 materialized view changes, got %d", len(mdiff.MaterializedViewChanges))
	}

	// Check for ALTER and CREATE actions
	foundAlter := false
	foundCreate := false
	for _, mvDiff := range mdiff.MaterializedViewChanges {
		if mvDiff.Action == diff.MetadataDiffActionAlter && mvDiff.ViewName == "mv_summary" {
			foundAlter = true
		}
		if mvDiff.Action == diff.MetadataDiffActionCreate && mvDiff.ViewName == "mv_new" {
			foundCreate = true
		}
	}

	if !foundAlter {
		t.Error("Expected ALTER action for mv_summary")
	}
	if !foundCreate {
		t.Error("Expected CREATE action for mv_new")
	}
}

// TestCompareProcedures tests procedure comparison.
func TestCompareProcedures(t *testing.T) {
	oldSchema := &schemaextract.DatabaseSchema{
		Name: "test_db",
		Schemas: []*schemaextract.Schema{
			{
				Name: "public",
				Procedures: []*schemaextract.Procedure{
					{
						Name:       "update_user",
						Definition: "BEGIN UPDATE users SET updated_at = NOW(); END",
					},
				},
			},
		},
	}

	newSchema := &schemaextract.DatabaseSchema{
		Name: "test_db",
		Schemas: []*schemaextract.Schema{
			{
				Name: "public",
				Procedures: []*schemaextract.Procedure{
					{
						Name:       "update_user",
						Definition: "BEGIN UPDATE users SET updated_at = CURRENT_TIMESTAMP; END",
					},
				},
			},
		},
	}

	opts := &CompareOptions{Engine: engine.MySQL}
	mdiff, err := CompareSchemasDetailed(oldSchema, newSchema, opts)
	if err != nil {
		t.Fatalf("CompareSchemasDetailed failed: %v", err)
	}

	if len(mdiff.ProcedureChanges) != 1 {
		t.Errorf("Expected 1 procedure change, got %d", len(mdiff.ProcedureChanges))
	}

	if len(mdiff.ProcedureChanges) > 0 && mdiff.ProcedureChanges[0].Action != diff.MetadataDiffActionAlter {
		t.Errorf("Expected ALTER action, got %s", mdiff.ProcedureChanges[0].Action)
	}
}

// TestCompareEnumTypes tests enum type comparison including values.
func TestCompareEnumTypes(t *testing.T) {
	oldSchema := &schemaextract.DatabaseSchema{
		Name: "test_db",
		Schemas: []*schemaextract.Schema{
			{
				Name: "public",
				EnumTypes: []*schemaextract.EnumType{
					{
						Name:   "status",
						Values: []string{"pending", "active"},
					},
				},
			},
		},
	}

	newSchema := &schemaextract.DatabaseSchema{
		Name: "test_db",
		Schemas: []*schemaextract.Schema{
			{
				Name: "public",
				EnumTypes: []*schemaextract.EnumType{
					{
						Name:   "status",
						Values: []string{"pending", "active", "completed"},
					},
				},
			},
		},
	}

	opts := &CompareOptions{Engine: engine.PostgreSQL}
	mdiff, err := CompareSchemasDetailed(oldSchema, newSchema, opts)
	if err != nil {
		t.Fatalf("CompareSchemasDetailed failed: %v", err)
	}

	if len(mdiff.EnumTypeChanges) != 1 {
		t.Errorf("Expected 1 enum type change, got %d", len(mdiff.EnumTypeChanges))
	}

	if len(mdiff.EnumTypeChanges) > 0 {
		enumDiff := mdiff.EnumTypeChanges[0]
		if enumDiff.Action != diff.MetadataDiffActionAlter {
			t.Errorf("Expected ALTER action, got %s", enumDiff.Action)
		}
		if len(enumDiff.AddedValues) != 1 || enumDiff.AddedValues[0] != "completed" {
			t.Errorf("Expected added value 'completed', got %v", enumDiff.AddedValues)
		}
	}
}

// TestCompareEvents tests MySQL event comparison.
func TestCompareEvents(t *testing.T) {
	oldSchema := &schemaextract.DatabaseSchema{
		Name: "test_db",
		Schemas: []*schemaextract.Schema{
			{
				Name: "public",
				Events: []*schemaextract.Event{
					{
						Name:       "cleanup_old_data",
						Definition: "DELETE FROM logs WHERE created_at < NOW() - INTERVAL 30 DAY",
					},
				},
			},
		},
	}

	newSchema := &schemaextract.DatabaseSchema{
		Name: "test_db",
		Schemas: []*schemaextract.Schema{
			{
				Name: "public",
				Events: []*schemaextract.Event{
					{
						Name:       "cleanup_old_data",
						Definition: "DELETE FROM logs WHERE created_at < NOW() - INTERVAL 90 DAY",
					},
				},
			},
		},
	}

	opts := &CompareOptions{Engine: engine.MySQL}
	mdiff, err := CompareSchemasDetailed(oldSchema, newSchema, opts)
	if err != nil {
		t.Fatalf("CompareSchemasDetailed failed: %v", err)
	}

	if len(mdiff.EventChanges) != 1 {
		t.Errorf("Expected 1 event change, got %d", len(mdiff.EventChanges))
	}

	if len(mdiff.EventChanges) > 0 && mdiff.EventChanges[0].Action != diff.MetadataDiffActionAlter {
		t.Errorf("Expected ALTER action, got %s", mdiff.EventChanges[0].Action)
	}
}

// TestCompareExtensions tests PostgreSQL extension comparison.
func TestCompareExtensions(t *testing.T) {
	oldSchema := &schemaextract.DatabaseSchema{
		Name: "test_db",
		Schemas: []*schemaextract.Schema{
			{
				Name: "public",
				Extensions: []*schemaextract.Extension{
					{
						Name:    "uuid-ossp",
						Version: "1.1",
					},
				},
			},
		},
	}

	newSchema := &schemaextract.DatabaseSchema{
		Name: "test_db",
		Schemas: []*schemaextract.Schema{
			{
				Name: "public",
				Extensions: []*schemaextract.Extension{
					{
						Name:    "uuid-ossp",
						Version: "1.2",
					},
					{
						Name:    "pg_trgm",
						Version: "1.6",
					},
				},
			},
		},
	}

	opts := &CompareOptions{Engine: engine.PostgreSQL}
	mdiff, err := CompareSchemasDetailed(oldSchema, newSchema, opts)
	if err != nil {
		t.Fatalf("CompareSchemasDetailed failed: %v", err)
	}

	if len(mdiff.ExtensionChanges) != 2 {
		t.Errorf("Expected 2 extension changes, got %d", len(mdiff.ExtensionChanges))
	}

	foundAlter := false
	foundCreate := false
	for _, extDiff := range mdiff.ExtensionChanges {
		if extDiff.Action == diff.MetadataDiffActionAlter && extDiff.ExtensionName == "uuid-ossp" {
			foundAlter = true
		}
		if extDiff.Action == diff.MetadataDiffActionCreate && extDiff.ExtensionName == "pg_trgm" {
			foundCreate = true
		}
	}

	if !foundAlter {
		t.Error("Expected ALTER action for uuid-ossp")
	}
	if !foundCreate {
		t.Error("Expected CREATE action for pg_trgm")
	}
}

// TestCompareRules tests PostgreSQL rule comparison.
func TestCompareRules(t *testing.T) {
	tests := []struct {
		name     string
		oldRules []*schemaextract.Rule
		newRules []*schemaextract.Rule
		expected int
	}{
		{
			name:     "no rules",
			oldRules: []*schemaextract.Rule{},
			newRules: []*schemaextract.Rule{},
			expected: 0,
		},
		{
			name:     "add rule",
			oldRules: []*schemaextract.Rule{},
			newRules: []*schemaextract.Rule{
				{
					Name:       "log_updates",
					Definition: "CREATE RULE log_updates AS ON UPDATE DO ...",
				},
			},
			expected: 1,
		},
		{
			name: "drop rule",
			oldRules: []*schemaextract.Rule{
				{
					Name:       "log_updates",
					Definition: "CREATE RULE log_updates AS ON UPDATE DO ...",
				},
			},
			newRules: []*schemaextract.Rule{},
			expected: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			diffs := compareRules(tt.oldRules, tt.newRules)

			if len(diffs) != tt.expected {
				t.Errorf("Expected %d diffs, got %d", tt.expected, len(diffs))
			}
		})
	}
}
