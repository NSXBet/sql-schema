package main

import (
	"fmt"
	"log"

	schemaextract "github.com/nsxbet/sql-schema"
	"github.com/nsxbet/sql-schema/comparer"
	"github.com/nsxbet/sql-schema/comparer/engine"
	"github.com/nsxbet/sql-schema/formatter"
	"github.com/nsxbet/sql-schema/planner"
)

func main() {
	// Example: Complete workflow from schema comparison to migration strategy
	fmt.Println("=== Schema Comparison and Migration Strategy Example ===")

	// Define old schema
	oldSchema := &schemaextract.DatabaseSchema{
		Name: "production_db",
		Schemas: []*schemaextract.Schema{
			{
				Name: "public",
				Tables: []*schemaextract.Table{
					{
						Name: "users",
						Columns: []*schemaextract.Column{
							{
								Name:     "id",
								Type:     "INTEGER",
								Nullable: false,
							},
							{
								Name:     "username",
								Type:     "VARCHAR(255)",
								Nullable: false,
							},
							{
								Name:     "created_at",
								Type:     "TIMESTAMP",
								Nullable: false,
							},
						},
						Indexes: []*schemaextract.Index{
							{
								Name:        "idx_users_username",
								Type:        "BTREE",
								Expressions: []string{"username"},
								Unique:      true,
							},
						},
					},
					{
						Name: "posts",
						Columns: []*schemaextract.Column{
							{
								Name:     "id",
								Type:     "INTEGER",
								Nullable: false,
							},
							{
								Name:     "user_id",
								Type:     "INTEGER",
								Nullable: false,
							},
							{
								Name:     "content",
								Type:     "TEXT",
								Nullable: false,
							},
						},
					},
				},
				Views: []*schemaextract.View{
					{
						Name:       "user_summary",
						Definition: "SELECT id, username FROM users",
					},
				},
			},
		},
	}

	// Define new schema (with changes)
	newSchema := &schemaextract.DatabaseSchema{
		Name: "production_db",
		Schemas: []*schemaextract.Schema{
			{
				Name: "public",
				Tables: []*schemaextract.Table{
					{
						Name: "users",
						Columns: []*schemaextract.Column{
							{
								Name:     "id",
								Type:     "INT", // Type alias: INT = INTEGER in PostgreSQL
								Nullable: false,
							},
							{
								Name:     "username",
								Type:     "VARCHAR(255)",
								Nullable: false,
							},
							{
								Name:     "email",
								Type:     "VARCHAR(255)",
								Nullable: true, // New column, nullable
							},
							{
								Name:     "created_at",
								Type:     "TIMESTAMP",
								Nullable: false,
							},
							{
								Name:     "updated_at",
								Type:     "TIMESTAMP",
								Nullable: true, // New column
							},
						},
						Indexes: []*schemaextract.Index{
							{
								Name:        "idx_users_username",
								Type:        "BTREE",
								Expressions: []string{"username"},
								Unique:      true,
							},
							{
								Name:        "idx_users_email",
								Type:        "BTREE",
								Expressions: []string{"email"},
								Unique:      false, // New index
							},
						},
					},
					// posts table removed
					{
						Name: "comments", // New table
						Columns: []*schemaextract.Column{
							{
								Name:     "id",
								Type:     "INTEGER",
								Nullable: false,
							},
							{
								Name:     "user_id",
								Type:     "INTEGER",
								Nullable: false,
							},
							{
								Name:     "post_id",
								Type:     "INTEGER",
								Nullable: false,
							},
							{
								Name:     "content",
								Type:     "TEXT",
								Nullable: false,
							},
						},
						ForeignKeys: []*schemaextract.ForeignKey{
							{
								Name:              "fk_comments_user",
								Columns:           []string{"user_id"},
								ReferencedTable:   "users",
								ReferencedColumns: []string{"id"},
								OnDelete:          "CASCADE",
							},
						},
					},
				},
				Views: []*schemaextract.View{
					{
						Name:       "user_summary",
						Definition: "SELECT id, username, email FROM users", // Definition changed
					},
				},
			},
		},
	}

	// Step 1: Compare schemas
	fmt.Println("Step 1: Comparing Schemas...")
	opts := &comparer.CompareOptions{
		Engine: engine.PostgreSQL,
	}

	mdiff, err := comparer.CompareSchemasDetailed(oldSchema, newSchema, opts)
	if err != nil {
		log.Fatalf("Failed to compare schemas: %v", err)
	}

	// Step 2: Generate text diff
	fmt.Println("\n=== Text Diff Report ===")
	textDiff := formatter.FormatDiffAsText(mdiff)
	fmt.Println(textDiff)

	// Step 3: Analyze migration strategy
	fmt.Println("\nStep 2: Analyzing Migration Strategy...")
	strategy, err := planner.AnalyzeStrategy(mdiff, engine.PostgreSQL)
	if err != nil {
		log.Fatalf("Failed to analyze migration strategy: %v", err)
	}

	// Step 4: Generate migration report
	fmt.Println("\n=== Migration Strategy Report ===")
	strategyText := formatter.FormatStrategyAsText(strategy)
	fmt.Println(strategyText)

	// Step 5: Export as JSON
	fmt.Println("\n=== JSON Export (Summary) ===")
	jsonSummary, err := formatter.FormatStrategyAsSummaryJSON(strategy, true)
	if err != nil {
		log.Fatalf("Failed to format as JSON: %v", err)
	}
	fmt.Println(jsonSummary)

	// Step 6: Export as Markdown
	fmt.Println("\n=== Markdown Report ===")
	mdReport := formatter.FormatStrategyAsMarkdown(strategy)
	fmt.Println(mdReport)

	// Summary
	fmt.Println("\n=== Summary ===")
	fmt.Printf("Total operations: %d\n", len(strategy.Operations))
	fmt.Printf("High-risk operations: %d\n", len(strategy.HighRiskOps))
	fmt.Printf("Data loss operations: %d\n", len(strategy.DataLossOps))

	if len(strategy.HighRiskOps) > 0 {
		fmt.Println("\n⚠️  WARNING: High-risk operations detected!")
		fmt.Println("Review the migration strategy report before executing.")
	}

	fmt.Println("\n=== Workflow Complete ===")
}
