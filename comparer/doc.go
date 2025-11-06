// Package comparer provides schema comparison and difference detection
// for MySQL and PostgreSQL databases.
//
// # Overview
//
// The comparer package analyzes two DatabaseSchema objects and produces a
// detailed list of differences, including:
//   - Added, removed, and modified tables
//   - Column changes (type, constraints, defaults)
//   - Index additions, removals, and modifications
//   - Foreign key constraint changes
//   - View, function, procedure changes
//   - Database-specific objects (triggers, events, sequences, etc.)
//
// # Usage
//
// Basic comparison:
//
//	diffs, err := comparer.CompareSchemas(oldSchema, newSchema, nil)
//	if err != nil {
//	    return err
//	}
//
//	for _, diff := range diffs {
//	    fmt.Printf("%s: %s\n", diff.Type, diff.Message)
//	}
//
// Comparison with options:
//
//	options := &comparer.CompareOptions{
//	    IgnoreTables: []string{"temp_*", "cache_*"},
//	    CompareData:  false,
//	}
//	diffs, err := comparer.CompareSchemas(oldSchema, newSchema, options)
//
// # Comparison Engines
//
// The package uses database-specific comparison engines (MySQL and PostgreSQL)
// that understand the nuances of each database system. These engines handle:
//   - Type compatibility (e.g., VARCHAR vs TEXT)
//   - Default value formatting
//   - Expression normalization
//   - Database-specific features
//
// # Integration Tests
//
// The package includes comprehensive integration tests that verify comparison
// accuracy against real MySQL and PostgreSQL databases using testcontainers.
package comparer
