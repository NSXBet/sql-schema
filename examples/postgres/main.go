package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"os"

	_ "github.com/lib/pq"
	"github.com/nsxbet/sql-schema/extractor/postgres"
)

func main() {
	// Get connection parameters from environment variables
	// Example: export PG_DSN="postgres://user:password@localhost:5432/dbname?sslmode=disable"
	dsn := os.Getenv("PG_DSN")
	if dsn == "" {
		log.Fatal("Please set PG_DSN environment variable (e.g., postgres://user:password@localhost:5432/dbname?sslmode=disable)")
	}

	// Parse database name from DSN
	dbName := extractDatabaseName(dsn)
	if dbName == "" {
		log.Fatal("Could not extract database name from DSN")
	}

	fmt.Printf("Connecting to PostgreSQL database: %s\n", dbName)

	// Open database connection
	db, err := sql.Open("postgres", dsn)
	if err != nil {
		log.Fatalf("Failed to open database: %v", err)
	}
	defer db.Close()

	// Test connection
	if err := db.Ping(); err != nil {
		log.Fatalf("Failed to ping database: %v", err)
	}
	fmt.Println("✓ Connected to PostgreSQL database")

	// Create extractor
	extractor := postgres.NewExtractor(db, dbName)

	// List databases
	fmt.Println("\n--- Listing Databases ---")
	databases, err := extractor.ListDatabases(context.Background())
	if err != nil {
		log.Fatalf("Failed to list databases: %v", err)
	}
	fmt.Printf("Found %d databases:\n", len(databases))
	for _, dbName := range databases {
		fmt.Printf("  - %s\n", dbName)
	}

	// Extract schema
	fmt.Printf("\n--- Extracting Schema for '%s' ---\n", dbName)
	schema, err := extractor.ExtractSchema(context.Background())
	if err != nil {
		log.Fatalf("Failed to extract schema: %v", err)
	}

	// Display summary
	fmt.Println("\n=== Schema Summary ===")
	fmt.Printf("Database: %s\n", schema.Name)
	fmt.Printf("Search Path: %v\n", schema.SearchPath)
	fmt.Printf("Schemas: %d\n", len(schema.Schemas))

	for _, s := range schema.Schemas {
		fmt.Printf("\nSchema: %s\n", s.Name)
		fmt.Printf("  Tables: %d\n", len(s.Tables))
		fmt.Printf("  Views: %d\n", len(s.Views))
		fmt.Printf("  Materialized Views: %d\n", len(s.MaterializedViews))
		fmt.Printf("  Functions: %d\n", len(s.Functions))
		fmt.Printf("  Procedures: %d\n", len(s.Procedures))
		fmt.Printf("  Sequences: %d\n", len(s.Sequences))
		fmt.Printf("  Extensions: %d\n", len(s.Extensions))
		fmt.Printf("  Enum Types: %d\n", len(s.EnumTypes))

		// Show table details
		if len(s.Tables) > 0 {
			fmt.Println("\n  Tables:")
			for _, table := range s.Tables {
				fmt.Printf("    - %s (%d columns, %d indexes, %d foreign keys)\n",
					table.Name, len(table.Columns), len(table.Indexes), len(table.ForeignKeys))
				if table.DataSize > 0 {
					fmt.Printf("      Data Size: %d bytes, Index Size: %d bytes\n",
						table.DataSize, table.IndexSize)
				}
				if table.Comment != "" {
					fmt.Printf("      Comment: %s\n", table.Comment)
				}

				// Show columns
				if len(table.Columns) > 0 && len(table.Columns) <= 10 {
					fmt.Println("      Columns:")
					for _, col := range table.Columns {
						nullable := "NOT NULL"
						if col.Nullable {
							nullable = "NULL"
						}
						fmt.Printf("        - %s %s %s", col.Name, col.Type, nullable)
						if col.Default != "" {
							fmt.Printf(" DEFAULT %s", col.Default)
						}
						if col.Generation != nil {
							fmt.Printf(" GENERATED ALWAYS AS (%s) %s",
								col.Generation.Expression, col.Generation.Type)
						}
						fmt.Println()
					}
				}

				// Show partitions
				if len(table.Partitions) > 0 {
					fmt.Printf("      Partitions: %d\n", len(table.Partitions))
					for _, part := range table.Partitions {
						fmt.Printf("        - %s (%s)\n", part.Name, part.Type)
					}
				}
			}
		}

		// Show view details
		if len(s.Views) > 0 {
			fmt.Println("\n  Views:")
			for _, view := range s.Views {
				fmt.Printf("    - %s (%d columns)\n", view.Name, len(view.Columns))
				if view.Comment != "" {
					fmt.Printf("      Comment: %s\n", view.Comment)
				}
			}
		}

		// Show materialized view details
		if len(s.MaterializedViews) > 0 {
			fmt.Println("\n  Materialized Views:")
			for _, mv := range s.MaterializedViews {
				fmt.Printf("    - %s (%d columns)\n", mv.Name, len(mv.Columns))
				if mv.Comment != "" {
					fmt.Printf("      Comment: %s\n", mv.Comment)
				}
			}
		}

		// Show sequence details
		if len(s.Sequences) > 0 {
			fmt.Println("\n  Sequences:")
			for _, seq := range s.Sequences {
				fmt.Printf("    - %s (type: %s, start: %d, increment: %d)\n",
					seq.Name, seq.DataType, seq.Start, seq.Increment)
				if seq.Comment != "" {
					fmt.Printf("      Comment: %s\n", seq.Comment)
				}
			}
		}

		// Show extension details
		if len(s.Extensions) > 0 {
			fmt.Println("\n  Extensions:")
			for _, ext := range s.Extensions {
				fmt.Printf("    - %s (version: %s)\n", ext.Name, ext.Version)
				if ext.Comment != "" {
					fmt.Printf("      Comment: %s\n", ext.Comment)
				}
			}
		}

		// Show enum type details
		if len(s.EnumTypes) > 0 {
			fmt.Println("\n  Enum Types:")
			for _, enum := range s.EnumTypes {
				fmt.Printf("    - %s: %v\n", enum.Name, enum.Values)
			}
		}

		// Show function details
		if len(s.Functions) > 0 {
			fmt.Println("\n  Functions:")
			for _, fn := range s.Functions {
				fmt.Printf("    - %s\n", fn.Name)
				if fn.Comment != "" {
					fmt.Printf("      Comment: %s\n", fn.Comment)
				}
			}
		}

		// Show procedure details
		if len(s.Procedures) > 0 {
			fmt.Println("\n  Procedures:")
			for _, proc := range s.Procedures {
				fmt.Printf("    - %s\n", proc.Name)
				if proc.Comment != "" {
					fmt.Printf("      Comment: %s\n", proc.Comment)
				}
			}
		}
	}

	// Export to JSON
	fmt.Println("\n--- Exporting to JSON ---")
	jsonData, err := json.MarshalIndent(schema, "", "  ")
	if err != nil {
		log.Fatalf("Failed to marshal JSON: %v", err)
	}

	jsonFile := "postgres_schema.json"
	if err := os.WriteFile(jsonFile, jsonData, 0o644); err != nil {
		log.Fatalf("Failed to write JSON file: %v", err)
	}
	fmt.Printf("✓ Schema exported to %s (%d bytes)\n", jsonFile, len(jsonData))

	fmt.Println("\n=== Extraction Complete ===")
}

func extractDatabaseName(dsn string) string {
	// Parse DSN format: postgres://user:password@host:port/dbname?params
	// Find the last / before ?
	slashIdx := -1
	questionIdx := -1

	for i := len(dsn) - 1; i >= 0; i-- {
		if dsn[i] == '?' && questionIdx == -1 {
			questionIdx = i
		}
		if dsn[i] == '/' && slashIdx == -1 {
			slashIdx = i
			break
		}
	}

	if slashIdx < 0 {
		return ""
	}

	dbName := dsn[slashIdx+1:]
	if questionIdx > slashIdx {
		dbName = dsn[slashIdx+1 : questionIdx]
	}

	return dbName
}
