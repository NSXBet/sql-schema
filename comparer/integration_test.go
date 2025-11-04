package comparer

import (
	"strings"
	"testing"

	schemaextract "github.com/nsxbet/sql-schema"
	"github.com/nsxbet/sql-schema/comparer/engine"
	"github.com/nsxbet/sql-schema/diff"
	"github.com/nsxbet/sql-schema/formatter"
	"github.com/nsxbet/sql-schema/planner"
)

// TestEndToEndWorkflow tests the complete workflow from schema comparison to output formatting.
func TestEndToEndWorkflow(t *testing.T) {
	// Define schemas
	oldSchema := &schemaextract.DatabaseSchema{
		Name: "test_db",
		Schemas: []*schemaextract.Schema{
			{
				Name: "public",
				Tables: []*schemaextract.Table{
					{
						Name: "users",
						Columns: []*schemaextract.Column{
							{Name: "id", Type: "INTEGER", Nullable: false},
							{Name: "name", Type: "VARCHAR(255)", Nullable: false},
						},
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
				Tables: []*schemaextract.Table{
					{
						Name: "users",
						Columns: []*schemaextract.Column{
							{Name: "id", Type: "INT", Nullable: false}, // Type alias
							{Name: "name", Type: "VARCHAR(255)", Nullable: false},
							{Name: "email", Type: "VARCHAR(255)", Nullable: true}, // New column
						},
					},
				},
			},
		},
	}

	// Step 1: Compare schemas
	opts := &CompareOptions{Engine: engine.PostgreSQL}
	mdiff, err := CompareSchemasDetailed(oldSchema, newSchema, opts)
	if err != nil {
		t.Fatalf("CompareSchemasDetailed failed: %v", err)
	}

	// Verify comparison results
	if len(mdiff.TableChanges) != 1 {
		t.Errorf("Expected 1 table change, got %d", len(mdiff.TableChanges))
	}
	if mdiff.TableChanges[0].Action != diff.MetadataDiffActionAlter {
		t.Errorf("Expected ALTER action, got %s", mdiff.TableChanges[0].Action)
	}
	if len(mdiff.TableChanges[0].ColumnChanges) != 1 {
		t.Errorf("Expected 1 column change, got %d", len(mdiff.TableChanges[0].ColumnChanges))
	}

	// Step 2: Analyze migration strategy
	strategy, err := planner.AnalyzeStrategy(mdiff, engine.PostgreSQL)
	if err != nil {
		t.Fatalf("AnalyzeStrategy failed: %v", err)
	}

	if len(strategy.Operations) != 1 {
		t.Errorf("Expected 1 operation, got %d", len(strategy.Operations))
	}

	// Step 3: Generate text output
	textOutput := formatter.FormatDiffAsText(mdiff)
	if !strings.Contains(textOutput, "test_db") {
		t.Error("Text output should contain database name")
	}
	if !strings.Contains(textOutput, "email") {
		t.Error("Text output should contain new column name")
	}

	// Step 4: Generate Markdown output
	mdOutput := formatter.FormatDiffAsMarkdown(mdiff)
	if !strings.Contains(mdOutput, "# Database Schema Comparison") {
		t.Error("Markdown output should have H1 header")
	}

	// Step 5: Generate JSON output
	jsonOutput, err := formatter.FormatDiffAsJSON(mdiff, true)
	if err != nil {
		t.Fatalf("FormatDiffAsJSON failed: %v", err)
	}
	if !strings.Contains(jsonOutput, "test_db") {
		t.Error("JSON output should contain database name")
	}

	// Step 6: Generate migration strategy output
	strategyText := formatter.FormatStrategyAsText(strategy)
	if !strings.Contains(strategyText, "Migration Strategy Report") {
		t.Error("Strategy text should have report header")
	}
}

// TestPostgreSQLTypeAliases tests that PostgreSQL type aliases are recognized as equivalent.
func TestPostgreSQLTypeAliases(t *testing.T) {
	testCases := []struct {
		name  string
		type1 string
		type2 string
	}{
		{"INTEGER vs INT", "INTEGER", "INT"},
		{"INTEGER vs INT4", "INTEGER", "INT4"},
		{"BIGINT vs INT8", "BIGINT", "INT8"},
		{"CHARACTER VARYING vs VARCHAR", "CHARACTER VARYING", "VARCHAR"},
		{"TIMESTAMP WITH TIME ZONE vs TIMESTAMPTZ", "TIMESTAMP WITH TIME ZONE", "TIMESTAMPTZ"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			oldSchema := &schemaextract.DatabaseSchema{
				Name: "test_db",
				Schemas: []*schemaextract.Schema{
					{
						Name: "public",
						Tables: []*schemaextract.Table{
							{
								Name: "test_table",
								Columns: []*schemaextract.Column{
									{Name: "col", Type: tc.type1, Nullable: false},
								},
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
						Tables: []*schemaextract.Table{
							{
								Name: "test_table",
								Columns: []*schemaextract.Column{
									{Name: "col", Type: tc.type2, Nullable: false},
								},
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

			// Should not detect any changes (types are equivalent)
			if len(mdiff.TableChanges) != 0 {
				t.Errorf("Type aliases %s and %s should be equivalent, but changes detected", tc.type1, tc.type2)
			}
		})
	}
}

// TestMySQLTypeAliases tests that MySQL type aliases are recognized as equivalent.
func TestMySQLTypeAliases(t *testing.T) {
	testCases := []struct {
		name  string
		type1 string
		type2 string
	}{
		{"INT vs INTEGER", "INT", "INTEGER"},
		{"BOOL vs BOOLEAN", "BOOL", "BOOLEAN"},
		{"BOOL vs TINYINT(1)", "BOOL", "TINYINT(1)"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			oldSchema := &schemaextract.DatabaseSchema{
				Name: "test_db",
				Schemas: []*schemaextract.Schema{
					{
						Name: "",
						Tables: []*schemaextract.Table{
							{
								Name: "test_table",
								Columns: []*schemaextract.Column{
									{Name: "col", Type: tc.type1, Nullable: false},
								},
							},
						},
					},
				},
			}

			newSchema := &schemaextract.DatabaseSchema{
				Name: "test_db",
				Schemas: []*schemaextract.Schema{
					{
						Name: "",
						Tables: []*schemaextract.Table{
							{
								Name: "test_table",
								Columns: []*schemaextract.Column{
									{Name: "col", Type: tc.type2, Nullable: false},
								},
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

			// Should not detect any changes (types are equivalent)
			if len(mdiff.TableChanges) != 0 {
				t.Errorf("Type aliases %s and %s should be equivalent, but changes detected", tc.type1, tc.type2)
			}
		})
	}
}

// TestComplexSchemaComparison tests a complex schema with multiple object types.
func TestComplexSchemaComparison(t *testing.T) {
	oldSchema := &schemaextract.DatabaseSchema{
		Name: "complex_db",
		Schemas: []*schemaextract.Schema{
			{
				Name: "public",
				Tables: []*schemaextract.Table{
					{
						Name: "users",
						Columns: []*schemaextract.Column{
							{Name: "id", Type: "INTEGER", Nullable: false},
						},
						Indexes: []*schemaextract.Index{
							{Name: "idx_id", Type: "BTREE", Expressions: []string{"id"}},
						},
					},
				},
				Views: []*schemaextract.View{
					{Name: "user_view", Definition: "SELECT * FROM users"},
				},
				Functions: []*schemaextract.Function{
					{Name: "get_user", Definition: "CREATE FUNCTION get_user(id INT) RETURNS TEXT AS $$ BEGIN RETURN 'user'; END; $$ LANGUAGE plpgsql;"},
				},
				Sequences: []*schemaextract.Sequence{
					{Name: "user_id_seq", Start: 1, Increment: 1},
				},
			},
		},
	}

	newSchema := &schemaextract.DatabaseSchema{
		Name: "complex_db",
		Schemas: []*schemaextract.Schema{
			{
				Name: "public",
				Tables: []*schemaextract.Table{
					{
						Name: "users",
						Columns: []*schemaextract.Column{
							{Name: "id", Type: "INTEGER", Nullable: false},
							{Name: "name", Type: "VARCHAR(255)", Nullable: true}, // New column
						},
						Indexes: []*schemaextract.Index{
							{Name: "idx_id", Type: "BTREE", Expressions: []string{"id"}},
							{Name: "idx_name", Type: "BTREE", Expressions: []string{"name"}}, // New index
						},
					},
					{
						Name: "posts", // New table
						Columns: []*schemaextract.Column{
							{Name: "id", Type: "INTEGER", Nullable: false},
						},
					},
				},
				Views: []*schemaextract.View{
					{Name: "user_view", Definition: "SELECT id, name FROM users"}, // Changed definition
				},
				Functions: []*schemaextract.Function{
					{Name: "get_user", Definition: "CREATE FUNCTION get_user(id INT) RETURNS TEXT AS $$ BEGIN RETURN 'user'; END; $$ LANGUAGE plpgsql;"},
					{Name: "get_post", Definition: "CREATE FUNCTION get_post(id INT) RETURNS TEXT AS $$ BEGIN RETURN 'post'; END; $$ LANGUAGE plpgsql;"}, // New function
				},
				Sequences: []*schemaextract.Sequence{
					{Name: "user_id_seq", Start: 1, Increment: 1},
					{Name: "post_id_seq", Start: 1, Increment: 1}, // New sequence
				},
			},
		},
	}

	opts := &CompareOptions{Engine: engine.PostgreSQL}
	mdiff, err := CompareSchemasDetailed(oldSchema, newSchema, opts)
	if err != nil {
		t.Fatalf("CompareSchemasDetailed failed: %v", err)
	}

	// Verify all types of changes are detected
	if len(mdiff.TableChanges) != 2 { // 1 altered (users), 1 created (posts)
		t.Errorf("Expected 2 table changes, got %d", len(mdiff.TableChanges))
	}

	if len(mdiff.ViewChanges) != 1 { // 1 altered
		t.Errorf("Expected 1 view change, got %d", len(mdiff.ViewChanges))
	}

	if len(mdiff.FunctionChanges) != 1 { // 1 created
		t.Errorf("Expected 1 function change, got %d", len(mdiff.FunctionChanges))
	}

	if len(mdiff.SequenceChanges) != 1 { // 1 created
		t.Errorf("Expected 1 sequence change, got %d", len(mdiff.SequenceChanges))
	}

	// Verify migration strategy
	strategy, err := planner.AnalyzeStrategy(mdiff, engine.PostgreSQL)
	if err != nil {
		t.Fatalf("AnalyzeStrategy failed: %v", err)
	}

	if len(strategy.Operations) < 4 {
		t.Errorf("Expected at least 4 operations, got %d", len(strategy.Operations))
	}

	// Verify execution order respects dependencies
	if len(strategy.OrderedOperations) != len(strategy.Operations) {
		t.Error("All operations should be in ordered list")
	}
}
