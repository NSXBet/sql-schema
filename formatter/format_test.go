package formatter

import (
	"strings"
	"testing"

	schemaextract "github.com/nsxbet/sql-schema"
	"github.com/nsxbet/sql-schema/comparer/engine"
	"github.com/nsxbet/sql-schema/diff"
	"github.com/nsxbet/sql-schema/planner"
)

func TestFormatDiffAsText(t *testing.T) {
	mdiff := &diff.MetadataDiff{
		DatabaseName: "test_db",
		SchemaChanges: []*diff.SchemaDiff{
			{
				Action:     diff.MetadataDiffActionCreate,
				SchemaName: "public",
			},
		},
		TableChanges: []*diff.TableDiff{
			{
				Action:     diff.MetadataDiffActionCreate,
				SchemaName: "public",
				TableName:  "users",
			},
		},
	}

	text := FormatDiffAsText(mdiff)

	if !strings.Contains(text, "test_db") {
		t.Error("Expected database name in output")
	}
	if !strings.Contains(text, "Schema Changes") {
		t.Error("Expected Schema Changes section")
	}
	if !strings.Contains(text, "Table Changes") {
		t.Error("Expected Table Changes section")
	}
	if !strings.Contains(text, "public") {
		t.Error("Expected schema name 'public'")
	}
	if !strings.Contains(text, "users") {
		t.Error("Expected table name 'users'")
	}
}

func TestFormatDiffAsMarkdown(t *testing.T) {
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

	md := FormatDiffAsMarkdown(mdiff)

	// Check for Markdown headers
	if !strings.Contains(md, "# Database Schema Comparison") {
		t.Error("Expected H1 header")
	}
	if !strings.Contains(md, "## Summary") {
		t.Error("Expected Summary section")
	}
	if !strings.Contains(md, "## Table Changes") {
		t.Error("Expected Table Changes section")
	}

	// Check for Markdown table
	if !strings.Contains(md, "| Action | Column") {
		t.Error("Expected column change table")
	}

	// Check for data
	if !strings.Contains(md, "email") {
		t.Error("Expected column name 'email'")
	}
}

func TestFormatDiffAsJSON(t *testing.T) {
	mdiff := &diff.MetadataDiff{
		DatabaseName: "test_db",
		SchemaChanges: []*diff.SchemaDiff{
			{
				Action:     diff.MetadataDiffActionCreate,
				SchemaName: "public",
			},
		},
	}

	// Test compact JSON
	compact, err := FormatDiffAsJSON(mdiff, false)
	if err != nil {
		t.Fatalf("FormatDiffAsJSON failed: %v", err)
	}
	if !strings.Contains(compact, "test_db") {
		t.Error("Expected database name in JSON output")
	}

	// Test pretty JSON
	pretty, err := FormatDiffAsJSON(mdiff, true)
	if err != nil {
		t.Fatalf("FormatDiffAsJSON (pretty) failed: %v", err)
	}
	if !strings.Contains(pretty, "test_db") {
		t.Error("Expected database name in pretty JSON output")
	}
	// Pretty JSON should have indentation
	if !strings.Contains(pretty, "  ") {
		t.Error("Expected indentation in pretty JSON")
	}
}

func TestFormatDiffAsSummaryJSON(t *testing.T) {
	mdiff := &diff.MetadataDiff{
		DatabaseName: "test_db",
		SchemaChanges: []*diff.SchemaDiff{
			{
				Action:     diff.MetadataDiffActionCreate,
				SchemaName: "public",
			},
		},
		TableChanges: []*diff.TableDiff{
			{
				Action:     diff.MetadataDiffActionCreate,
				SchemaName: "public",
				TableName:  "users",
			},
		},
	}

	summary, err := FormatDiffAsSummaryJSON(mdiff, true)
	if err != nil {
		t.Fatalf("FormatDiffAsSummaryJSON failed: %v", err)
	}

	if !strings.Contains(summary, "test_db") {
		t.Error("Expected database name in summary")
	}
	if !strings.Contains(summary, "totalChanges") {
		t.Error("Expected totalChanges field")
	}
	if !strings.Contains(summary, "summary") {
		t.Error("Expected summary field")
	}
}

func TestFormatStrategyAsText(t *testing.T) {
	strategy := &planner.MigrationStrategy{
		Engine: engine.PostgreSQL,
		Operations: []*planner.MigrationOperation{
			{
				ID:          "op1",
				Type:        planner.OperationTypeTable,
				Action:      diff.MetadataDiffActionCreate,
				Description: "CREATE table 'users'",
				Risk:        planner.RiskLevelNone,
				Reversible:  true,
			},
			{
				ID:          "op2",
				Type:        planner.OperationTypeTable,
				Action:      diff.MetadataDiffActionDrop,
				Description: "DROP table 'old_table'",
				Risk:        planner.RiskLevelHigh,
				Reversible:  false,
				Warnings:    []string{"All data will be lost"},
			},
		},
		OrderedOperations: []*planner.MigrationOperation{},
		HighRiskOps: []*planner.MigrationOperation{
			{
				ID:          "op2",
				Type:        planner.OperationTypeTable,
				Action:      diff.MetadataDiffActionDrop,
				Description: "DROP table 'old_table'",
				Risk:        planner.RiskLevelHigh,
				Reversible:  false,
				Warnings:    []string{"All data will be lost"},
			},
		},
		Warnings: []string{"1 high-risk operation detected"},
	}

	// Populate ordered operations
	strategy.OrderedOperations = strategy.Operations

	text := FormatStrategyAsText(strategy)

	if !strings.Contains(text, "Migration Strategy Report") {
		t.Error("Expected report title")
	}
	if !strings.Contains(text, "POSTGRES") {
		t.Error("Expected engine name")
	}
	if !strings.Contains(text, "High-Risk Operations") {
		t.Error("Expected high-risk section")
	}
	if !strings.Contains(text, "Execution Order") {
		t.Error("Expected execution order section")
	}
	if !strings.Contains(text, "DROP table 'old_table'") {
		t.Error("Expected operation description")
	}
}

func TestFormatStrategyAsMarkdown(t *testing.T) {
	strategy := &planner.MigrationStrategy{
		Engine: engine.MySQL,
		Operations: []*planner.MigrationOperation{
			{
				ID:          "op1",
				Type:        planner.OperationTypeColumn,
				Action:      diff.MetadataDiffActionCreate,
				Description: "CREATE column 'email'",
				Risk:        planner.RiskLevelLow,
				Reversible:  true,
			},
		},
		OrderedOperations: []*planner.MigrationOperation{},
		HighRiskOps:       []*planner.MigrationOperation{},
		Warnings:          []string{"Test warning"},
	}

	strategy.OrderedOperations = strategy.Operations

	md := FormatStrategyAsMarkdown(strategy)

	// Check Markdown formatting
	if !strings.Contains(md, "# Migration Strategy Report") {
		t.Error("Expected H1 header")
	}
	if !strings.Contains(md, "## Summary") {
		t.Error("Expected Summary section")
	}
	if !strings.Contains(md, "| # | Action") {
		t.Error("Expected table header")
	}

	// Check data
	if !strings.Contains(md, "MYSQL") {
		t.Error("Expected engine name")
	}
	if !strings.Contains(md, "email") {
		t.Error("Expected column name")
	}
}

func TestFormatStrategyAsJSON(t *testing.T) {
	strategy := &planner.MigrationStrategy{
		Engine: engine.PostgreSQL,
		Operations: []*planner.MigrationOperation{
			{
				ID:          "op1",
				Type:        planner.OperationTypeTable,
				Action:      diff.MetadataDiffActionCreate,
				Description: "CREATE table 'users'",
				Risk:        planner.RiskLevelNone,
				Reversible:  true,
			},
		},
		OrderedOperations: []*planner.MigrationOperation{},
	}

	strategy.OrderedOperations = strategy.Operations

	jsonStr, err := FormatStrategyAsJSON(strategy, true)
	if err != nil {
		t.Fatalf("FormatStrategyAsJSON failed: %v", err)
	}

	if !strings.Contains(jsonStr, "POSTGRES") {
		t.Error("Expected engine in JSON")
	}
	if !strings.Contains(jsonStr, "CREATE table 'users'") {
		t.Error("Expected operation description in JSON")
	}
}

func TestFormatStrategyAsSummaryJSON(t *testing.T) {
	strategy := &planner.MigrationStrategy{
		Engine: engine.PostgreSQL,
		Operations: []*planner.MigrationOperation{
			{
				ID:          "op1",
				Type:        planner.OperationTypeTable,
				Action:      diff.MetadataDiffActionCreate,
				Description: "CREATE table 'users'",
				Risk:        planner.RiskLevelLow,
				Reversible:  true,
			},
			{
				ID:          "op2",
				Type:        planner.OperationTypeColumn,
				Action:      diff.MetadataDiffActionCreate,
				Description: "CREATE column 'email'",
				Risk:        planner.RiskLevelLow,
				Reversible:  true,
			},
		},
		OrderedOperations: []*planner.MigrationOperation{},
		HighRiskOps:       []*planner.MigrationOperation{},
	}

	strategy.OrderedOperations = strategy.Operations

	summary, err := FormatStrategyAsSummaryJSON(strategy, true)
	if err != nil {
		t.Fatalf("FormatStrategyAsSummaryJSON failed: %v", err)
	}

	if !strings.Contains(summary, "totalOperations") {
		t.Error("Expected totalOperations field")
	}
	if !strings.Contains(summary, "riskDistribution") {
		t.Error("Expected riskDistribution field")
	}
	if !strings.Contains(summary, "operationTypes") {
		t.Error("Expected operationTypes field")
	}
	if !strings.Contains(summary, "executionOrder") {
		t.Error("Expected executionOrder field")
	}
}
