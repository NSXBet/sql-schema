# SQL Schema

[![Go Version](https://img.shields.io/badge/Go-1.23%2B-blue.svg)](https://golang.org/)
[![License](https://img.shields.io/badge/License-MIT-green.svg)](LICENSE)
[![Version](https://img.shields.io/badge/version-v0.1.0--beta-orange.svg)](CHANGELOG.md)
[![Tests](https://img.shields.io/badge/tests-passing-brightgreen.svg)](https://github.com/nsxbet/sql-schema)

A comprehensive Go package for database schema extraction, comparison, and migration planning for MySQL and PostgreSQL databases.

> **Status:** Beta (v0.1.0) - Production-ready core functionality with active development. Feedback welcome!

## Features

### Schema Extraction
- **Complete Schema Extraction**: Extract all database objects including tables, views, functions, procedures, triggers, sequences, and more
- **Plain Go Structs**: No protobuf dependencies - uses idiomatic Go structs with JSON/YAML tags
- **Database Support**:
  - MySQL 5.7+ and 8.0+
  - PostgreSQL 10+
- **Comprehensive Metadata**: Captures all schema details including:
  - Tables with columns, indexes, foreign keys, check constraints, partitions
  - Views and materialized views (PostgreSQL)
  - Functions and stored procedures
  - Triggers
  - Sequences (PostgreSQL)
  - Extensions (PostgreSQL)
  - Enum types (PostgreSQL)
  - Events (MySQL)
  - Generated columns
  - Table statistics (row counts, sizes)
- **System Object Filtering**: Automatically filters out system objects and built-in functions
- **Transaction-based Extraction**: PostgreSQL extraction uses a single read-only transaction for consistency
- **Type Normalization**: MySQL type synonyms are canonicalized for consistency

### Schema Comparison & Migration Planning
- **Engine-Aware Comparison**: Intelligent comparison with PostgreSQL and MySQL-specific logic
- **Type Alias Recognition**: Understands equivalent types (INTEGER=INT=INT4, VARCHAR=CHARACTER VARYING, etc.)
- **11+ Object Types**: Compare tables, views, functions, procedures, sequences, enums, events, extensions, and more
- **Semantic Expression Comparison**: Normalizes and compares expressions semantically
- **Risk Assessment**: Automatic risk analysis (NONE, LOW, MEDIUM, HIGH) for migration operations
- **Dependency Ordering**: Topological sort ensures correct execution order
- **Multiple Output Formats**: Generate reports in Text, Markdown, or JSON
- **Auto-Import**: Engines register automatically - no manual imports needed

### Schema Snapshots
- **Point-in-Time Captures**: Take snapshots of database schemas with rich metadata
- **Multiple Formats**: Save/load snapshots in JSON or YAML format
- **Integrity Verification**: Optional MD5 and SHA256 checksums for snapshot validation
- **Metadata Tracking**: Track timestamp, version, database engine, tags, and descriptions
- **Snapshot Comparison**: Compare snapshots to detect schema changes over time
- **SQL Export**: Export snapshots as SQL DDL statements (CREATE TABLE, etc.)
- **Snapshot Management**: List, load, validate, and organize snapshot files
- **Use Cases**:
  - Pre/post-migration snapshots for rollback capability
  - Environment comparison (production vs staging)
  - Schema versioning and change tracking
  - Documentation and audit trails

## Installation

```bash
go get github.com/nsxbet/sql-schema
```

## Quick Start

### MySQL Example

```go
package main

import (
    "context"
    "database/sql"
    "encoding/json"
    "fmt"
    "log"

    _ "github.com/go-sql-driver/mysql"
    "github.com/nsxbet/sql-schema/mysql"
)

func main() {
    // Open database connection
    db, err := sql.Open("mysql", "user:password@tcp(localhost:3306)/mydb")
    if err != nil {
        log.Fatal(err)
    }
    defer db.Close()

    // Create extractor
    extractor := mysql.NewExtractor(db, "mydb")

    // List all databases
    databases, err := extractor.ListDatabases(context.Background())
    if err != nil {
        log.Fatal(err)
    }
    fmt.Printf("Found %d databases\n", len(databases))

    // Extract complete schema
    schema, err := extractor.ExtractSchema(context.Background())
    if err != nil {
        log.Fatal(err)
    }

    // Schema is ready to use
    fmt.Printf("Database: %s\n", schema.Name)
    fmt.Printf("Character Set: %s\n", schema.CharacterSet)
    fmt.Printf("Collation: %s\n", schema.Collation)

    for _, s := range schema.Schemas {
        fmt.Printf("Tables: %d, Views: %d, Functions: %d\n",
            len(s.Tables), len(s.Views), len(s.Functions))
    }

    // Export to JSON
    jsonData, _ := json.MarshalIndent(schema, "", "  ")
    fmt.Println(string(jsonData))
}
```

### PostgreSQL Example

```go
package main

import (
    "context"
    "database/sql"
    "encoding/json"
    "fmt"
    "log"

    _ "github.com/lib/pq"
    "github.com/nsxbet/sql-schema/postgres"
)

func main() {
    // Open database connection
    db, err := sql.Open("postgres", "postgres://user:password@localhost/mydb?sslmode=disable")
    if err != nil {
        log.Fatal(err)
    }
    defer db.Close()

    // Create extractor
    extractor := postgres.NewExtractor(db, "mydb")

    // List all databases (excludes templates and system databases)
    databases, err := extractor.ListDatabases(context.Background())
    if err != nil {
        log.Fatal(err)
    }
    fmt.Printf("Found %d databases\n", len(databases))

    // Extract complete schema
    schema, err := extractor.ExtractSchema(context.Background())
    if err != nil {
        log.Fatal(err)
    }

    // Schema is ready to use
    fmt.Printf("Database: %s\n", schema.Name)
    fmt.Printf("Search Path: %v\n", schema.SearchPath)

    for _, s := range schema.Schemas {
        fmt.Printf("Schema: %s\n", s.Name)
        fmt.Printf("  Tables: %d\n", len(s.Tables))
        fmt.Printf("  Views: %d\n", len(s.Views))
        fmt.Printf("  Materialized Views: %d\n", len(s.MaterializedViews))
        fmt.Printf("  Functions: %d\n", len(s.Functions))
        fmt.Printf("  Procedures: %d\n", len(s.Procedures))
        fmt.Printf("  Sequences: %d\n", len(s.Sequences))
        fmt.Printf("  Extensions: %d\n", len(s.Extensions))
        fmt.Printf("  Enum Types: %d\n", len(s.EnumTypes))
    }

    // Export to JSON
    jsonData, _ := json.MarshalIndent(schema, "", "  ")
    fmt.Println(string(jsonData))
}
```

## API Reference

### Core Types

#### DatabaseSchema

The root structure representing a complete database schema.

```go
type DatabaseSchema struct {
    Name         string    // Database name
    CharacterSet string    // MySQL only: default character set
    Collation    string    // MySQL only: default collation
    SearchPath   []string  // PostgreSQL only: search path
    Schemas      []*Schema // List of schemas (always 1 for MySQL, multiple for PostgreSQL)
}
```

#### Schema

Represents a schema (namespace) within a database.

```go
type Schema struct {
    Name               string
    Tables             []*Table
    Views              []*View
    MaterializedViews  []*MaterializedView  // PostgreSQL only
    Functions          []*Function
    Procedures         []*Procedure
    Triggers           []*Trigger
    Sequences          []*Sequence          // PostgreSQL only
    Extensions         []*Extension         // PostgreSQL only
    EnumTypes          []*EnumType          // PostgreSQL only
    Events             []*Event             // MySQL only
}
```

#### Table

Complete table metadata including columns, indexes, constraints, and statistics.

```go
type Table struct {
    Name             string
    Columns          []*Column
    Indexes          []*Index
    ForeignKeys      []*ForeignKey
    CheckConstraints []*CheckConstraint
    Triggers         []*Trigger
    Partitions       []*Partition
    Comment          string
    Engine           string  // MySQL only
    Collation        string  // MySQL only
    RowCount         int64
    DataSize         int64
    IndexSize        int64
    DataFree         int64   // MySQL only
}
```

See [types.go](types.go) for complete type definitions.

### MySQL API

```go
package mysql

// NewExtractor creates a new MySQL schema extractor
func NewExtractor(db *sql.DB, databaseName string) *Extractor

// ListDatabases returns a list of all databases
func (e *Extractor) ListDatabases(ctx context.Context) ([]string, error)

// ExtractSchema extracts the complete schema for the configured database
func (e *Extractor) ExtractSchema(ctx context.Context) (*schemaextract.DatabaseSchema, error)
```

### PostgreSQL API

```go
package postgres

// NewExtractor creates a new PostgreSQL schema extractor
func NewExtractor(db *sql.DB, databaseName string) *Extractor

// ListDatabases returns a list of all user databases (excludes templates and system databases)
func (e *Extractor) ListDatabases(ctx context.Context) ([]string, error)

// ExtractSchema extracts the complete schema for the configured database
// Uses a single read-only transaction for consistency
func (e *Extractor) ExtractSchema(ctx context.Context) (*schemaextract.DatabaseSchema, error)
```

### Schema Comparison API

The `util` package provides comprehensive schema comparison with automatic engine registration:

```go
package util

import (
    "github.com/nsxbet/sql-schema/util"
    "github.com/nsxbet/sql-schema/util/engine"
)

// Note: Engine comparers (PostgreSQL, MySQL, etc.) are automatically registered
// when you import the util package. No need for blank imports!

// CompareOptions configures schema comparison behavior
type CompareOptions struct {
    Engine          engine.Engine  // Database engine (PostgreSQL, MySQL, etc.)
    IgnoreComments  bool           // Ignore comment differences
    IgnoreCharset   bool           // Ignore charset differences
    IgnoreCollation bool           // Ignore collation differences
}

// CompareSchemasDetailed compares two schemas and returns detailed differences
func CompareSchemasDetailed(
    oldSchema *DatabaseSchema,
    newSchema *DatabaseSchema,
    opts *CompareOptions,
) (*diff.MetadataDiff, error)
```

**Example Usage:**

```go
import (
    "github.com/nsxbet/sql-schema/util"
    "github.com/nsxbet/sql-schema/util/engine"
    "github.com/nsxbet/sql-schema/util/migration"
    "github.com/nsxbet/sql-schema/util/format"
)

// Compare schemas (engines auto-registered)
opts := &util.CompareOptions{
    Engine: engine.PostgreSQL,
}

diff, err := util.CompareSchemasDetailed(oldSchema, newSchema, opts)
if err != nil {
    log.Fatal(err)
}

// Analyze migration strategy
strategy, _ := migration.AnalyzeStrategy(diff, engine.PostgreSQL)

// Generate outputs in multiple formats
textReport := format.FormatDiffAsText(diff)
mdReport := format.FormatStrategyAsMarkdown(strategy)
jsonSummary, _ := format.FormatStrategyAsSummaryJSON(strategy, true)
```

**Supported Features:**
- ✅ All database object types (11+)
- ✅ PostgreSQL and MySQL with type alias recognition
- ✅ Semantic expression comparison
- ✅ Migration risk assessment
- ✅ Dependency ordering
- ✅ Multiple output formats (Text, Markdown, JSON)

See [examples/complete_workflow_example.go](examples/complete_workflow_example.go) for a complete example.

## Database Support

### MySQL

**Supported Versions:**
- MySQL 5.7+
- MySQL 8.0+
- MariaDB 10.3+

**Extracted Objects:**
- Tables with all metadata
- Views
- Functions
- Stored procedures
- Triggers
- Events
- Partitions (all types: RANGE, LIST, HASH, KEY)
- Generated columns (VIRTUAL/STORED)

**Features:**
- Type synonym canonicalization (e.g., BOOLEAN → TINYINT(1))
- Character set and collation detection
- Storage engine information
- Table statistics (row count, data size, index size, data free)

### PostgreSQL

**Supported Versions:**
- PostgreSQL 10+
- PostgreSQL 11+ (for procedures)

**Extracted Objects:**
- Multiple schemas per database
- Tables with all metadata
- Views
- Materialized views
- Functions
- Procedures (PostgreSQL 11+)
- Triggers
- Sequences
- Extensions
- Enum types
- Partitions (PostgreSQL 10+)
- Generated columns

**Features:**
- System object filtering (900+ built-in functions filtered)
- Cloud provider object filtering (AWS RDS, GCP CloudSQL, AlloyDB)
- Extension dependency tracking
- Transaction-based extraction for consistency
- Search path support

## Configuration

### Connection Management

This package accepts `*sql.DB` connections, giving you full control over:
- Connection pooling
- Timeouts
- SSL/TLS configuration
- Connection string parameters

Example with custom configuration:

```go
import "database/sql"

// MySQL with custom settings
db, err := sql.Open("mysql", "user:password@tcp(localhost:3306)/mydb?timeout=30s&readTimeout=60s")
db.SetMaxOpenConns(10)
db.SetMaxIdleConns(5)
db.SetConnMaxLifetime(time.Hour)

// PostgreSQL with custom settings
db, err := sql.Open("postgres", "postgres://user:password@localhost/mydb?sslmode=require&connect_timeout=30")
db.SetMaxOpenConns(10)
```

### Context Support

All extraction methods accept `context.Context` for cancellation and timeout control:

```go
ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
defer cancel()

schema, err := extractor.ExtractSchema(ctx)
```

## Use Cases

### Schema Documentation

Generate comprehensive documentation of your database schema:

```go
schema, _ := extractor.ExtractSchema(ctx)

for _, s := range schema.Schemas {
    for _, table := range s.Tables {
        fmt.Printf("## Table: %s\n", table.Name)
        if table.Comment != "" {
            fmt.Printf("%s\n\n", table.Comment)
        }

        fmt.Println("### Columns")
        for _, col := range table.Columns {
            nullable := "NOT NULL"
            if col.Nullable {
                nullable = "NULL"
            }
            fmt.Printf("- **%s** (%s) %s", col.Name, col.Type, nullable)
            if col.Comment != "" {
                fmt.Printf(" - %s", col.Comment)
            }
            fmt.Println()
        }
        fmt.Println()
    }
}
```

### Schema Comparison

Compare schemas between environments:

```go
prodSchema, _ := prodExtractor.ExtractSchema(ctx)
devSchema, _ := devExtractor.ExtractSchema(ctx)

// Compare table counts
prodTables := len(prodSchema.Schemas[0].Tables)
devTables := len(devSchema.Schemas[0].Tables)

if prodTables != devTables {
    fmt.Printf("Table count mismatch: prod=%d, dev=%d\n", prodTables, devTables)
}
```

### Migration Planning

Identify schema changes for migration planning:

```go
schema, _ := extractor.ExtractSchema(ctx)

// Find tables without primary keys
for _, s := range schema.Schemas {
    for _, table := range s.Tables {
        hasPK := false
        for _, idx := range table.Indexes {
            if idx.Primary {
                hasPK = true
                break
            }
        }
        if !hasPK {
            fmt.Printf("Table %s has no primary key\n", table.Name)
        }
    }
}
```

### Schema Backup

Export complete schema as JSON/YAML for backup or version control:

```go
import (
    "encoding/json"
    "os"
    "gopkg.in/yaml.v3"
)

schema, _ := extractor.ExtractSchema(ctx)

// Export as JSON
jsonFile, _ := os.Create("schema-backup.json")
encoder := json.NewEncoder(jsonFile)
encoder.SetIndent("", "  ")
encoder.Encode(schema)

// Export as YAML
yamlFile, _ := os.Create("schema-backup.yaml")
yamlEncoder := yaml.NewEncoder(yamlFile)
yamlEncoder.Encode(schema)
```

## Examples

See the [examples](examples/) directory for complete working examples:

- [examples/mysql](examples/mysql) - MySQL schema extraction example
- [examples/postgres](examples/postgres) - PostgreSQL schema extraction example
- [examples/snapshot](examples/snapshot) - Schema snapshot management example

Each example includes:
- Connection setup
- Database listing
- Schema extraction
- Summary output
- JSON export

### Snapshot Example

```go
package main

import (
    "database/sql"
    "github.com/nsxbet/sql-schema/comparer/engine"
    "github.com/nsxbet/sql-schema/snapshot"
)

func main() {
    db, _ := sql.Open("postgres", "postgresql://user:password@localhost:5432/mydb")
    defer db.Close()

    // Take a snapshot with metadata
    snap, _ := snapshot.TakeSnapshot(db, "mydb", engine.PostgreSQL, &snapshot.SnapshotOptions{
        Version:     "1.0",
        Description: "Production database snapshot before migration",
        Tags: map[string]string{
            "environment": "production",
            "purpose":     "pre-migration",
        },
        ComputeHash: true, // Verify integrity
    })

    // Save snapshot
    snapshot.SaveSnapshot(snap, "snapshots/production_before.json")

    // Load and compare snapshots
    mdiff, _ := snapshot.LoadAndCompare(
        "snapshots/production_before.json",
        "snapshots/production_after.json",
        nil, // Auto-detect engine
    )

    // Export as SQL DDL
    snapshot.ExportSnapshotSQL(snap, "snapshots/schema.sql")
}
```

See [examples/snapshot](examples/snapshot) for more comprehensive examples.

## Testing

Run the integration tests (requires Docker):

```bash
# Run all tests
go test ./...

# Run MySQL tests only
go test ./mysql/...

# Run PostgreSQL tests only
go test ./postgres/...

# Run with verbose output
go test -v ./...
```

The tests use [testcontainers-go](https://golang.testcontainers.org/) to automatically spin up MySQL and PostgreSQL containers, create test schemas, and verify extraction functionality.

## Performance

### Optimization Tips

1. **Use Connection Pooling**: Configure `sql.DB` with appropriate pool settings
2. **Set Timeouts**: Use context with timeout for large databases
3. **Filter Early**: The package already filters system objects automatically
4. **Transaction Size**: PostgreSQL extraction runs in a single transaction - ensure your database can handle read-only transactions for the duration

### Benchmarks

Typical extraction times (local environment):

| Database Size | MySQL 8.0 | PostgreSQL 15 |
|--------------|-----------|---------------|
| Small (10 tables) | ~50ms | ~100ms |
| Medium (100 tables) | ~200ms | ~400ms |
| Large (1000 tables) | ~2s | ~4s |

*Note: Times vary based on database configuration, hardware, and number of objects.*

## Troubleshooting

### MySQL: "You do not have the SUPER privilege"

When creating functions/procedures, you might encounter:
```
Error 1419 (HY000): You do not have the SUPER privilege and binary logging is enabled
```

**Solution**: Set `log_bin_trust_function_creators=1` in MySQL configuration or:
```sql
SET GLOBAL log_bin_trust_function_creators = 1;
```

### PostgreSQL: "permission denied for table pg_proc"

The user needs read access to system catalogs.

**Solution**: Grant necessary permissions:
```sql
GRANT SELECT ON ALL TABLES IN SCHEMA pg_catalog TO your_user;
```

### Connection Timeout

For large databases, extraction may take longer than default timeout.

**Solution**: Increase timeout in context:
```go
ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
defer cancel()
```

### Memory Usage

For databases with thousands of tables, extraction may use significant memory.

**Solution**: Consider extracting specific schemas or tables rather than the entire database.

## Dependencies

### Core Dependencies

- `database/sql` - Standard library database interface
- `context` - Standard library context support

### Database Drivers (peer dependencies)

- MySQL: `github.com/go-sql-driver/mysql`
- PostgreSQL: `github.com/lib/pq`

### Utilities

- `github.com/blang/semver/v4` - Version parsing
- `golang.org/x/text` - Character encoding

## Contributing

This package is part of the [Bytebase](https://github.com/bytebase/bytebase) project.

### Development Setup

1. Clone the repository:
```bash
git clone https://github.com/bytebase/bytebase.git
cd bytebase/sql-schema
```

2. Install dependencies:
```bash
go mod download
```

3. Run tests:
```bash
go test ./...
```

### Code Style

- Follow [Google Go Style Guide](https://google.github.io/styleguide/go/)
- Run `gofmt -w .` before committing
- Ensure all tests pass
- Add tests for new functionality

## License

This package is part of Bytebase and follows the same license.

## Related Projects

- [Bytebase](https://github.com/bytebase/bytebase) - Database CI/CD for DevOps teams
- [sqlparser](https://github.com/bytebase/bytebase/tree/main/backend/plugin/parser) - SQL parsers for multiple databases

## Support

- [GitHub Issues](https://github.com/bytebase/bytebase/issues) - Bug reports and feature requests
- [Discord](https://discord.gg/bytebase) - Community chat
- [Documentation](https://www.bytebase.com/docs) - Bytebase documentation
