// Package diff defines types for representing schema differences
// between two database schemas.
//
// # Overview
//
// The diff package provides structured representations of schema changes,
// including additions, removals, and modifications of database objects.
//
// # Difference Types
//
// The package defines differences for:
//   - Tables: Added, removed, renamed
//   - Columns: Type changes, nullability, defaults
//   - Indexes: Added, removed, modified
//   - Foreign keys: Added, removed, modified
//   - Check constraints: Added, removed, modified
//   - Views: Definition changes
//   - Functions/Procedures: Definition changes
//   - Triggers: Added, removed, modified
//   - Database-specific objects (sequences, enums, extensions, events)
//
// # SchemaDifference Structure
//
// Each difference includes:
//   - Type: The kind of difference (Added, Removed, Modified)
//   - Category: The schema object category (Table, Column, Index, etc.)
//   - Object: The affected object name
//   - Schema: The schema/namespace (for PostgreSQL)
//   - OldValue: Previous value (for modifications)
//   - NewValue: New value (for modifications)
//   - Message: Human-readable description
//   - Severity: Impact level (Info, Warning, Critical)
//
// # Usage
//
// Differences are typically produced by the comparer package:
//
//	diffs, _ := comparer.CompareSchemas(old, new, nil)
//	for _, diff := range diffs {
//	    fmt.Printf("[%s] %s.%s: %s\n",
//	        diff.Severity, diff.Category, diff.Object, diff.Message)
//	}
//
// # Severity Levels
//
//   - Info: Additive changes, no data loss risk
//   - Warning: Modifications that may require attention
//   - Critical: Destructive changes with potential data loss
//
// # Filtering
//
// Differences can be filtered by:
//   - Category (e.g., only table changes)
//   - Severity (e.g., only critical issues)
//   - Object name patterns (e.g., exclude temp_* tables)
//
// # Integration
//
// The diff types are used by:
//   - Comparer: Produces differences
//   - Planner: Consumes differences to generate migration plans
//   - Formatter: Formats differences for human consumption
//   - Snapshot: Compares snapshots and produces differences
package diff
