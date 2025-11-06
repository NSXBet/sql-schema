// Package mysql provides MySQL-specific schema comparison engine.
//
// # Overview
//
// This package implements the ComparisonEngine interface for MySQL databases,
// handling MySQL-specific features and comparison logic.
//
// # MySQL-Specific Features
//
// The MySQL engine handles:
//   - Storage engines (InnoDB, MyISAM, etc.)
//   - Character sets and collations
//   - AUTO_INCREMENT columns
//   - FULLTEXT and SPATIAL indexes
//   - Triggers with SQL_MODE settings
//   - Scheduled events
//   - Partitioning (RANGE, LIST, HASH, KEY)
//   - Generated columns (VIRTUAL, STORED)
//
// # Type Comparison
//
// MySQL has specific rules for type equivalence:
//   - VARCHAR(255) vs TINYTEXT
//   - INT vs INTEGER
//   - BOOL vs TINYINT(1)
//
// The engine normalizes types before comparison to detect semantic changes
// rather than syntactic differences.
//
// # Expression Handling
//
// MySQL expressions are parsed and normalized to handle:
//   - Function name case sensitivity
//   - Whitespace variations
//   - Quote style differences
//
// # Compatibility
//
// Supports MySQL 5.7+ and 8.0+, including version-specific features like
// invisible columns and functional indexes.
package mysql
