// Package engine provides database-specific comparison engines for MySQL
// and PostgreSQL schema comparison.
//
// # Overview
//
// This package defines the ComparisonEngine interface and provides concrete
// implementations for MySQL and PostgreSQL. Each engine understands the
// specific nuances of its target database system.
//
// # Architecture
//
// The engine package follows a pluggable architecture:
//   - Interfaces define the contract for comparison engines
//   - Base package provides shared comparison utilities
//   - MySQL and PostgreSQL packages provide database-specific implementations
//   - Registry manages engine selection based on database type
//
// # Creating Custom Engines
//
// To create a custom comparison engine:
//
//	type CustomEngine struct {
//	    // Implement ComparisonEngine interface
//	}
//
//	func (e *CustomEngine) CompareSchemas(old, new *schemaextract.DatabaseSchema) ([]*diff.SchemaDifference, error) {
//	    // Custom comparison logic
//	}
//
//	// Register the engine
//	engine.RegisterEngine("custom", &CustomEngine{})
//
// # Database-Specific Handling
//
// Each engine handles database-specific features:
//   - MySQL: ENGINE, AUTO_INCREMENT, FULLTEXT indexes, events
//   - PostgreSQL: Extensions, sequences, enum types, materialized views
//
// # Thread Safety
//
// Engine instances are safe for concurrent use. The registry is thread-safe.
package engine
