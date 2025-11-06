// Package mysql provides MySQL-specific database schema extraction.
//
// # Overview
//
// This package extracts complete schema metadata from MySQL databases
// (versions 5.7+, 8.0+) by querying information_schema and system tables.
//
// # Supported Features
//
// The MySQL extractor captures:
//   - Tables with storage engines, character sets, collations
//   - Columns including AUTO_INCREMENT, generated columns
//   - Indexes (BTREE, HASH, FULLTEXT, SPATIAL)
//   - Foreign keys with referential actions
//   - Check constraints (MySQL 8.0+)
//   - Triggers with SQL_MODE settings
//   - Views with definitions
//   - Stored functions and procedures
//   - Scheduled events
//   - Partitioning information
//   - Table statistics (row count, data size)
//
// # Usage
//
// Basic extraction:
//
//	import (
//	    "database/sql"
//	    _ "github.com/go-sql-driver/mysql"
//	    "github.com/NSXBet/sql-schema/extractor/mysql"
//	)
//
//	db, _ := sql.Open("mysql", "user:pass@tcp(localhost:3306)/mydb")
//	schema, err := mysql.Extract(db)
//
// Extraction with options:
//
//	schema, err := mysql.Extract(db, mysql.Options{
//	    ExcludeTables:     []string{"temp_*", "cache_*"},
//	    IncludeStatistics: true,
//	})
//
// # Database Permissions
//
// The extractor requires SELECT permissions on:
//   - information_schema.TABLES
//   - information_schema.COLUMNS
//   - information_schema.STATISTICS
//   - information_schema.KEY_COLUMN_USAGE
//   - information_schema.TABLE_CONSTRAINTS
//   - information_schema.TRIGGERS
//   - information_schema.VIEWS
//   - information_schema.ROUTINES
//   - information_schema.EVENTS
//   - information_schema.PARTITIONS
//   - mysql.proc (for older MySQL versions)
//
// For statistics, additional permissions may be required:
//   - SELECT on target tables (for SHOW TABLE STATUS)
//
// # Version Compatibility
//
// MySQL 5.7:
//   - Basic features fully supported
//   - Check constraints not available
//   - Invisible columns not available
//
// MySQL 8.0+:
//   - All features supported
//   - Check constraints extracted
//   - Invisible columns detected
//   - Functional indexes supported
//
// # Performance Considerations
//
// Extraction performance depends on:
//   - Number of tables and objects
//   - Network latency
//   - Server load
//
// For large databases (1000+ tables):
//   - Consider excluding statistics
//   - Use table filtering
//   - Run during low-traffic periods
//
// # Thread Safety
//
// The Extract function is safe for concurrent use with different
// database connections. Do not share sql.DB connections across
// goroutines without proper synchronization.
package mysql
