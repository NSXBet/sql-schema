package main

import (
	"database/sql"
	"fmt"
	"log"
	"os"

	_ "github.com/go-sql-driver/mysql"
	_ "github.com/lib/pq"
	schemaextract "github.com/nsxbet/sql-schema"
	"github.com/nsxbet/sql-schema/comparer/engine"
	"github.com/nsxbet/sql-schema/formatter"
	"github.com/nsxbet/sql-schema/planner"
	"github.com/nsxbet/sql-schema/snapshot"
)

func main() {
	// Example 1: Taking and saving a PostgreSQL snapshot
	fmt.Println("=== Example 1: Taking a PostgreSQL Snapshot ===")
	postgresSnapshot()

	// Example 2: Taking and saving a MySQL snapshot
	fmt.Println("\n=== Example 2: Taking a MySQL Snapshot ===")
	mysqlSnapshot()

	// Example 3: Loading and validating a snapshot
	fmt.Println("\n=== Example 3: Loading and Validating a Snapshot ===")
	loadAndValidate()

	// Example 4: Comparing two snapshots
	fmt.Println("\n=== Example 4: Comparing Two Snapshots ===")
	compareSnapshots()

	// Example 5: Listing snapshots in a directory
	fmt.Println("\n=== Example 5: Listing Snapshots ===")
	listSnapshots()

	// Example 6: Exporting snapshot as SQL DDL
	fmt.Println("\n=== Example 6: Exporting Snapshot as SQL ===")
	exportSQL()
}

func postgresSnapshot() {
	// Connect to PostgreSQL
	db, err := sql.Open("postgres", "postgresql://deeb:deeb_dev_password@localhost:5432/deeb?sslmode=disable")
	if err != nil {
		log.Printf("Failed to connect to PostgreSQL: %v", err)
		return
	}
	defer func() { _ = db.Close() }()

	// Take snapshot with options
	snap, err := snapshot.TakeSnapshot(db, "mydb", engine.PostgreSQL, &snapshot.SnapshotOptions{
		Version:     "1.0",
		Description: "Production database snapshot before migration",
		Tags: map[string]string{
			"environment": "production",
			"purpose":     "pre-migration",
		},
		HostInfo:    "localhost:5432",
		ComputeHash: true, // Compute checksums for integrity verification
	})
	if err != nil {
		log.Printf("Failed to take snapshot: %v", err)
		return
	}

	// Save snapshot as JSON
	err = snapshot.SaveSnapshot(snap, "snapshots/mydb_production.json")
	if err != nil {
		log.Printf("Failed to save snapshot: %v", err)
		return
	}

	fmt.Printf("✓ Snapshot saved: mydb_production.json\n")
	fmt.Printf("  Database: %s\n", snap.Metadata.DatabaseName)
	fmt.Printf("  Timestamp: %s\n", snap.Metadata.Timestamp.Format("2006-01-02 15:04:05"))
	fmt.Printf("  MD5: %s\n", snap.Metadata.ChecksumMD5)
	fmt.Printf("  SHA256: %s\n", snap.Metadata.ChecksumSHA256)

	// Also save as YAML
	err = snapshot.SaveSnapshot(snap, "snapshots/mydb_production.yaml")
	if err != nil {
		log.Printf("Failed to save YAML snapshot: %v", err)
		return
	}

	fmt.Printf("✓ Snapshot also saved as YAML\n")
}

func mysqlSnapshot() {
	// Connect to MySQL
	db, err := sql.Open("mysql", "user:password@tcp(localhost:3306)/mydb")
	if err != nil {
		log.Printf("Failed to connect to MySQL: %v", err)
		return
	}
	defer func() { _ = db.Close() }()

	// Take snapshot with minimal options
	snap, err := snapshot.TakeSnapshot(db, "mydb", engine.MySQL, &snapshot.SnapshotOptions{
		Description: "MySQL database snapshot",
	})
	if err != nil {
		log.Printf("Failed to take snapshot: %v", err)
		return
	}

	// Save snapshot
	err = snapshot.SaveSnapshot(snap, "snapshots/mydb_mysql.json")
	if err != nil {
		log.Printf("Failed to save snapshot: %v", err)
		return
	}

	fmt.Printf("✓ Snapshot saved: mydb_mysql.json\n")
	fmt.Printf("  Database: %s\n", snap.Metadata.DatabaseName)
	fmt.Printf("  Tables: %d\n", countTables(snap.Schema))
}

func loadAndValidate() {
	// Load a snapshot
	snap, err := snapshot.LoadSnapshot("snapshots/mydb_production.json")
	if err != nil {
		log.Printf("Failed to load snapshot: %v", err)
		return
	}

	fmt.Printf("✓ Loaded snapshot: %s\n", snap.Metadata.DatabaseName)
	fmt.Printf("  Timestamp: %s\n", snap.Metadata.Timestamp.Format("2006-01-02 15:04:05"))
	fmt.Printf("  Description: %s\n", snap.Metadata.Description)
	fmt.Printf("  Tags: %v\n", snap.Metadata.Tags)

	// Validate the snapshot
	err = snapshot.ValidateSnapshot("snapshots/mydb_production.json")
	if err != nil {
		log.Printf("Validation failed: %v", err)
		return
	}

	fmt.Printf("✓ Snapshot validation passed\n")

	// Print schema summary
	fmt.Printf("\nSchema Summary:\n")
	fmt.Printf("  Schemas: %d\n", len(snap.Schema.Schemas))
	fmt.Printf("  Tables: %d\n", countTables(snap.Schema))
	fmt.Printf("  Views: %d\n", countViews(snap.Schema))
	fmt.Printf("  Functions: %d\n", countFunctions(snap.Schema))
}

func compareSnapshots() {
	// Compare two snapshot files
	mdiff, err := snapshot.LoadAndCompare(
		"snapshots/mydb_production.json",
		"snapshots/mydb_production2.json",
		nil, // Auto-detect engine from snapshots
	)
	if err != nil {
		log.Printf("Failed to compare snapshots: %v", err)
		return
	}

	// Generate comparison report
	report := formatter.FormatDiffAsText(mdiff)
	fmt.Printf("Comparison Report:\n%s\n", report)

	// Analyze migration strategy
	eng := engine.PostgreSQL // Or determine from snapshot metadata
	strategy, err := planner.AnalyzeStrategy(mdiff, eng)
	if err != nil {
		log.Printf("Failed to analyze strategy: %v", err)
		return
	}

	// Display migration strategy
	strategyReport := formatter.FormatStrategyAsText(strategy)
	fmt.Printf("\nMigration Strategy:\n%s\n", strategyReport)

	// Save reports to files
	if err := os.WriteFile("reports/comparison.txt", []byte(report), 0o644); err != nil {
		log.Printf("Failed to write comparison report: %v", err)
	}
	if err := os.WriteFile("reports/strategy.txt", []byte(strategyReport), 0o644); err != nil {
		log.Printf("Failed to write strategy report: %v", err)
	}

	fmt.Printf("✓ Reports saved to reports/ directory\n")
}

func listSnapshots() {
	// List all snapshots in a directory
	snapshots, err := snapshot.ListSnapshots("snapshots/")
	if err != nil {
		log.Printf("Failed to list snapshots: %v", err)
		return
	}

	fmt.Printf("Found %d snapshots:\n", len(snapshots))
	for _, info := range snapshots {
		fmt.Printf("\n  File: %s\n", info.Filename)
		fmt.Printf("    Database: %s (%s)\n", info.DatabaseName, info.DatabaseEngine)
		fmt.Printf("    Timestamp: %s\n", info.Timestamp.Format("2006-01-02 15:04:05"))
		fmt.Printf("    Size: %d bytes\n", info.FileSize)
		if info.Description != "" {
			fmt.Printf("    Description: %s\n", info.Description)
		}
		if len(info.Tags) > 0 {
			fmt.Printf("    Tags: %v\n", info.Tags)
		}
	}
}

func exportSQL() {
	// Load a snapshot
	snap, err := snapshot.LoadSnapshot("snapshots/mydb_production.json")
	if err != nil {
		log.Printf("Failed to load snapshot: %v", err)
		return
	}

	// Export as SQL DDL
	err = snapshot.ExportSnapshotSQL(snap, "snapshots/mydb_production.sql")
	if err != nil {
		log.Printf("Failed to export SQL: %v", err)
		return
	}

	fmt.Printf("✓ SQL DDL exported to: mydb_production.sql\n")

	// Read and display first few lines
	data, err := os.ReadFile("snapshots/mydb_production.sql")
	if err != nil {
		log.Printf("Failed to read SQL file: %v", err)
		return
	}

	lines := splitLines(string(data), 10)
	fmt.Printf("\nFirst 10 lines of SQL DDL:\n")
	for i, line := range lines {
		fmt.Printf("  %2d: %s\n", i+1, line)
	}
}

// Helper functions

func countTables(schema *schemaextract.DatabaseSchema) int {
	count := 0
	for _, s := range schema.Schemas {
		count += len(s.Tables)
	}
	return count
}

func countViews(schema *schemaextract.DatabaseSchema) int {
	count := 0
	for _, s := range schema.Schemas {
		count += len(s.Views)
	}
	return count
}

func countFunctions(schema *schemaextract.DatabaseSchema) int {
	count := 0
	for _, s := range schema.Schemas {
		count += len(s.Functions)
	}
	return count
}

func splitLines(s string, max int) []string {
	lines := []string{}
	current := ""
	count := 0

	for _, ch := range s {
		if ch == '\n' {
			lines = append(lines, current)
			current = ""
			count++
			if count >= max {
				break
			}
		} else {
			current += string(ch)
		}
	}

	if current != "" && count < max {
		lines = append(lines, current)
	}

	return lines
}
