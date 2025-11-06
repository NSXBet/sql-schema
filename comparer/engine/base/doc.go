// Package defaultcomparer provides shared comparison utilities and base implementations
// for database schema comparison engines.
//
// # Overview
//
// The base package contains common comparison logic that is reused across
// MySQL and PostgreSQL comparison engines. This includes:
//   - Column comparison (type, nullable, default)
//   - Index comparison (type, uniqueness, expressions)
//   - Foreign key comparison
//   - Check constraint comparison
//   - Trigger comparison
//
// # Usage
//
// Database-specific engines typically embed or use base comparers:
//
//	type MySQLEngine struct {
//	    *base.BaseComparer
//	    // MySQL-specific fields
//	}
//
//	func (e *MySQLEngine) CompareColumns(old, new *Column) []Difference {
//	    diffs := e.BaseComparer.CompareColumns(old, new)
//	    // Add MySQL-specific column comparison
//	    return diffs
//	}
//
// # Comparison Strategy
//
// The base package uses a structural comparison approach:
//   1. Sort objects by name for consistent ordering
//   2. Compare objects pairwise
//   3. Detect additions, removals, modifications
//   4. Normalize values before comparison
//
// # Expression Normalization
//
// The package includes utilities for normalizing SQL expressions to handle
// equivalent but differently formatted expressions (e.g., whitespace, quotes).
package defaultcomparer
