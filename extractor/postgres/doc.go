// Package postgres provides PostgreSQL-specific database schema extraction.
//
// # Overview
//
// This package extracts complete schema metadata from PostgreSQL databases
// (versions 10+) by querying pg_catalog system tables and information_schema.
//
// # Supported Features
//
// The PostgreSQL extractor captures:
//   - Multiple schemas (namespaces)
//   - Tables with storage parameters
//   - Columns including IDENTITY, generated columns
//   - Indexes (BTREE, HASH, GIN, GIST, BRIN, SP-GIST)
//   - Partial indexes (with WHERE clauses)
//   - Expression indexes
//   - Foreign keys with referential actions
//   - Check constraints
//   - Triggers
//   - Views and materialized views
//   - Functions (including stored procedures in PG 11+)
//   - Sequences
//   - Extensions (PostGIS, pg_trgm, etc.)
//   - Enum types
//   - Rules (rewrite rules)
//   - Table inheritance
//   - Partitioning (declarative partitioning)
//
// # Usage
//
// Basic extraction:
//
//	import (
//	    "database/sql"
//	    _ "github.com/lib/pq"
//	    "github.com/NSXBet/sql-schema/extractor/postgres"
//	)
//
//	db, _ := sql.Open("postgres", "postgresql://user:pass@localhost/mydb")
//	schema, err := postgres.Extract(db)
//
// Extraction with options:
//
//	schema, err := postgres.Extract(db, postgres.Options{
//	    Schemas:       []string{"public", "app"},
//	    ExcludeTables: []string{"temp_*"},
//	    IncludeSystemSchemas: false,
//	})
//
// # Schema Filtering
//
// By default, the extractor includes all non-system schemas. System schemas
// are automatically excluded:
//   - pg_catalog
//   - information_schema
//   - pg_toast
//   - pg_temp_*
//
// Use the Schemas option to explicitly specify which schemas to extract.
//
// # Database Permissions
//
// The extractor requires SELECT permissions on:
//   - pg_catalog.pg_namespace
//   - pg_catalog.pg_class
//   - pg_catalog.pg_attribute
//   - pg_catalog.pg_index
//   - pg_catalog.pg_constraint
//   - pg_catalog.pg_trigger
//   - pg_catalog.pg_proc
//   - pg_catalog.pg_type
//   - pg_catalog.pg_sequence
//   - pg_catalog.pg_extension
//   - pg_catalog.pg_enum
//   - pg_catalog.pg_rewrite
//   - information_schema.views
//   - information_schema.columns
//
// For statistics:
//   - SELECT on target tables
//
// # Version Compatibility
//
// PostgreSQL 10+:
//   - All basic features supported
//   - Declarative partitioning
//   - Identity columns
//
// PostgreSQL 11+:
//   - Stored procedures (vs functions)
//   - Hash partitioning
//
// PostgreSQL 12+:
//   - Generated columns
//   - Improved partitioning
//
// PostgreSQL 13+:
//   - Extended statistics
//   - Incremental materialized view refresh
//
// # Performance Considerations
//
// Extraction performance depends on:
//   - Number of schemas and objects
//   - Extension complexity (e.g., PostGIS objects)
//   - Network latency
//   - Server load
//
// For large databases:
//   - Filter to specific schemas
//   - Exclude statistics collection
//   - Use connection pooling
//
// # System Object Handling
//
// The package includes utilities to identify and optionally exclude:
//   - System tables (pg_*)
//   - Extension tables (spatial_ref_sys, topology, etc.)
//   - Temporary tables
//   - Toast tables
//
// # Thread Safety
//
// The Extract function is safe for concurrent use with different
// database connections. Do not share sql.DB connections across
// goroutines without proper synchronization.
package postgres
