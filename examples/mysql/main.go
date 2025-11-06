package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"os"

	_ "github.com/go-sql-driver/mysql"
	"github.com/nsxbet/sql-schema/extractor/mysql"
)

func main() {
	// Get connection parameters from environment variables
	// Example: export MYSQL_DSN="user:password@tcp(localhost:3306)/dbname"
	dsn := os.Getenv("MYSQL_DSN")
	if dsn == "" {
		log.Fatal("Please set MYSQL_DSN environment variable (e.g., user:password@tcp(localhost:3306)/dbname)")
	}

	// Parse database name from DSN
	// Format: user:password@tcp(host:port)/dbname
	dbName := extractDatabaseName(dsn)
	if dbName == "" {
		log.Fatal("Could not extract database name from DSN")
	}

	fmt.Printf("Connecting to MySQL database: %s\n", dbName)

	// Open database connection
	db, err := sql.Open("mysql", dsn)
	if err != nil {
		log.Fatalf("Failed to open database: %v", err)
	}
	defer db.Close()

	// Test connection
	if err := db.Ping(); err != nil {
		log.Fatalf("Failed to ping database: %v", err)
	}
	fmt.Println("✓ Connected to MySQL database")

	// Create extractor
	extractor := mysql.NewExtractor(db, dbName)

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
	fmt.Printf("Character Set: %s\n", schema.CharacterSet)
	fmt.Printf("Collation: %s\n", schema.Collation)
	fmt.Printf("Schemas: %d\n", len(schema.Schemas))

	for _, s := range schema.Schemas {
		fmt.Printf("\nSchema: %s (unnamed schema in MySQL)\n", s.Name)
		fmt.Printf("  Tables: %d\n", len(s.Tables))
		fmt.Printf("  Views: %d\n", len(s.Views))
		fmt.Printf("  Functions: %d\n", len(s.Functions))
		fmt.Printf("  Procedures: %d\n", len(s.Procedures))
		fmt.Printf("  Events: %d\n", len(s.Events))

		// Show table details
		if len(s.Tables) > 0 {
			fmt.Println("\n  Tables:")
			for _, table := range s.Tables {
				fmt.Printf("    - %s (%d columns, %d indexes, %d foreign keys)\n",
					table.Name, len(table.Columns), len(table.Indexes), len(table.ForeignKeys))
				if table.Engine != "" {
					fmt.Printf("      Engine: %s, Rows: %d, Size: %d bytes\n",
						table.Engine, table.RowCount, table.DataSize)
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
						fmt.Println()
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

	jsonFile := "mysql_schema.json"
	if err := os.WriteFile(jsonFile, jsonData, 0o644); err != nil {
		log.Fatalf("Failed to write JSON file: %v", err)
	}
	fmt.Printf("✓ Schema exported to %s (%d bytes)\n", jsonFile, len(jsonData))

	fmt.Println("\n=== Extraction Complete ===")
}

func extractDatabaseName(dsn string) string {
	// Parse DSN format: user:password@tcp(host:port)/dbname
	// or user:password@tcp(host:port)/dbname?params
	parts := splitDSN(dsn)
	if len(parts) >= 2 {
		dbPart := parts[1]
		// Remove query parameters
		if idx := findChar(dbPart, '?'); idx >= 0 {
			dbPart = dbPart[:idx]
		}
		return dbPart
	}
	return ""
}

func splitDSN(dsn string) []string {
	idx := findChar(dsn, '/')
	if idx < 0 {
		return []string{dsn}
	}
	return []string{dsn[:idx], dsn[idx+1:]}
}

func findChar(s string, ch rune) int {
	for i, c := range s {
		if c == ch {
			return i
		}
	}
	return -1
}
