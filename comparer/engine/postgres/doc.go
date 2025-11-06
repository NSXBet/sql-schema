// Package postgres provides PostgreSQL-specific schema comparison engine.
//
// # Overview
//
// This package implements the ComparisonEngine interface for PostgreSQL
// databases, handling PostgreSQL-specific features and comparison logic.
//
// # PostgreSQL-Specific Features
//
// The PostgreSQL engine handles:
//   - Schemas (namespaces)
//   - Extensions (PostGIS, pg_trgm, etc.)
//   - Sequences and IDENTITY columns
//   - Enum types
//   - Materialized views
//   - GIN, GIST, BRIN, and other specialized indexes
//   - Partial indexes (WHERE clauses)
//   - Expression indexes
//   - Rules (rewrite rules)
//   - Foreign data wrappers (future)
//
// # Type Comparison
//
// PostgreSQL has rich type system with specific comparison rules:
//   - VARCHAR vs TEXT vs CHARACTER VARYING
//   - SERIAL vs INTEGER with sequence
//   - Array types (e.g., INTEGER[])
//   - Domain types
//   - Custom composite types
//
// The engine normalizes types and handles type aliases to detect true
// semantic changes.
//
// # Schema Awareness
//
// PostgreSQL supports multiple schemas (namespaces) within a database.
// The engine properly handles qualified names (schema.table) and compares
// objects within their schema context.
//
// # Expression Handling
//
// PostgreSQL expressions require special handling:
//   - Type casts (::type vs CAST)
//   - Function overloading
//   - Operator precedence
//   - Array and JSON operators
//
// # Compatibility
//
// Supports PostgreSQL 10+ including newer features like declarative
// partitioning, logical replication objects, and stored procedures
// (vs functions).
package postgres
