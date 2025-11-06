// Package schemaextract provides comprehensive database schema extraction,
// comparison, and migration management for MySQL and PostgreSQL databases.
//
// # Overview
//
// This library enables developers to:
//   - Extract complete database schemas including tables, views, functions, triggers, and more
//   - Compare schemas to detect differences and generate migration plans
//   - Export schemas to JSON, YAML, or SQL formats
//   - Create point-in-time snapshots for schema versioning
//   - Generate human-readable schema comparison reports
//
// # Supported Databases
//
//   - MySQL 5.7+, 8.0+
//   - PostgreSQL 10+
//
// # Quick Start
//
// Extract a MySQL schema:
//
//	import (
//	    "database/sql"
//	    "github.com/NSXBet/sql-schema/extractor/mysql"
//	    _ "github.com/go-sql-driver/mysql"
//	)
//
//	db, _ := sql.Open("mysql", "user:pass@tcp(localhost:3306)/mydb")
//	schema, err := mysql.Extract(db)
//	if err != nil {
//	    log.Fatal(err)
//	}
//
// Compare two schemas:
//
//	import "github.com/NSXBet/sql-schema/comparer"
//
//	diffs, err := comparer.CompareSchemas(oldSchema, newSchema, nil)
//	if err != nil {
//	    log.Fatal(err)
//	}
//
// Export schema to JSON:
//
//	import "github.com/NSXBet/sql-schema/exporter"
//
//	err := exporter.ExportJSON(schema, "schema.json")
//	if err != nil {
//	    log.Fatal(err)
//	}
//
// # Architecture
//
// The library is organized into several key packages:
//
//   - extractor/mysql, extractor/postgres: Database-specific schema extraction
//   - comparer: Schema comparison and difference detection
//   - planner: Migration planning and strategy generation
//   - formatter: Human-readable report generation (text, markdown, JSON)
//   - exporter: Schema export to various formats
//   - snapshot: Point-in-time schema snapshots with versioning
//
// # Security Considerations
//
// All database queries use parameterized statements to prevent SQL injection.
// Database credentials should be managed securely using environment variables
// or secret management systems. Schema information should be treated as
// sensitive data.
//
// # Thread Safety
//
// All exported functions are safe for concurrent use unless otherwise documented.
//
// # Beta Status
//
// This library is currently in beta (v0.1.x). API breaking changes may occur
// between minor versions. Production use is supported but extensive testing
// is recommended.
package schemaextract
