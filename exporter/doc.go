// Package exporter provides schema export functionality to various
// formats including JSON, YAML, and SQL.
//
// # Overview
//
// The exporter package enables saving DatabaseSchema objects in different
// formats for documentation, version control, or migration purposes.
//
// # Supported Formats
//
//   - JSON: Compact or pretty-printed JSON
//   - YAML: Human-readable YAML format
//   - SQL: DDL statements to recreate the schema
//
// # Usage
//
// Export to JSON:
//
//	err := exporter.ExportJSON(schema, "schema.json", &exporter.JSONOptions{
//	    PrettyPrint: true,
//	})
//
// Export to YAML:
//
//	err := exporter.ExportYAML(schema, "schema.yaml", nil)
//
// Export to SQL:
//
//	err := exporter.ExportSQL(schema, "schema.sql", &exporter.SQLOptions{
//	    Dialect:         exporter.DialectMySQL,
//	    IncludeDrops:    false,
//	    IncludeComments: true,
//	})
//
// # JSON Export
//
// JSON exports preserve all schema metadata and are suitable for:
//   - Machine consumption
//   - API responses
//   - Data interchange
//   - Compact storage
//
// Options:
//   - PrettyPrint: Indent with spaces (default: true)
//   - Indent: Indentation string (default: "  ")
//
// # YAML Export
//
// YAML exports are more human-readable and are suitable for:
//   - Version control (Git)
//   - Configuration files
//   - Documentation
//   - Manual review
//
// YAML output is always pretty-printed with consistent formatting.
//
// # SQL Export
//
// SQL exports generate DDL statements to recreate the schema:
//
//	CREATE TABLE users (
//	  id INT PRIMARY KEY,
//	  username VARCHAR(50) NOT NULL,
//	  email VARCHAR(100) UNIQUE
//	);
//
// SQL export options:
//   - Dialect: MySQL or PostgreSQL syntax
//   - IncludeDrops: Add DROP statements before CREATE
//   - IncludeComments: Add SQL comments with metadata
//   - IfNotExists: Use CREATE IF NOT EXISTS
//
// # Snapshot Export
//
// The package also supports exporting Snapshot objects which include
// metadata alongside the schema:
//
//	err := exporter.ExportSnapshot(snapshot, "snapshot.json")
//
// # File Handling
//
// All export functions:
//   - Create parent directories if needed
//   - Set appropriate file permissions (0644)
//   - Return detailed errors on failure
//   - Support both absolute and relative paths
package exporter
