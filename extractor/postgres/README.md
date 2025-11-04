# PostgreSQL Schema Extractor

Complete PostgreSQL schema metadata extraction with support for PostgreSQL 10+.

## Features

- **Multiple Schemas**: Support for multiple schemas per database
- **Transaction-based Extraction**: Single read-only transaction for consistency
- **System Object Filtering**: Filters 900+ built-in PostgreSQL functions
- **Cloud Provider Support**: Filters AWS RDS, GCP CloudSQL, AlloyDB objects
- **Extension Tracking**: Tracks extension dependencies
- **Complete Metadata**: All PostgreSQL-specific objects and features
- **Version-aware**: Detects PostgreSQL version and adjusts extraction accordingly

## Quick Start

```go
package main

import (
    "context"
    "database/sql"
    "fmt"
    "log"

    _ "github.com/lib/pq"
    "github.com/nsxbet/sql-schema/postgres"
)

func main() {
    db, err := sql.Open("postgres",
        "postgres://user:password@localhost/mydb?sslmode=disable")
    if err != nil {
        log.Fatal(err)
    }
    defer db.Close()

    extractor := postgres.NewExtractor(db, "mydb")

    schema, err := extractor.ExtractSchema(context.Background())
    if err != nil {
        log.Fatal(err)
    }

    fmt.Printf("Database: %s\n", schema.Name)
    fmt.Printf("Search Path: %v\n", schema.SearchPath)

    for _, s := range schema.Schemas {
        fmt.Printf("Schema: %s\n", s.Name)
        fmt.Printf("  Tables: %d\n", len(s.Tables))
        fmt.Printf("  Views: %d\n", len(s.Views))
        fmt.Printf("  Functions: %d\n", len(s.Functions))
    }
}
```

## API

### NewExtractor

```go
func NewExtractor(db *sql.DB, databaseName string) *Extractor
```

Creates a new PostgreSQL schema extractor.

**Parameters:**
- `db`: Active `*sql.DB` connection to the PostgreSQL server
- `databaseName`: Name of the database to extract

**Returns:**
- `*Extractor`: Ready-to-use extractor instance

### ListDatabases

```go
func (e *Extractor) ListDatabases(ctx context.Context) ([]string, error)
```

Returns a list of all user databases, excluding:
- Template databases (template0, template1)
- System databases (postgres)
- Cloud provider databases (rdsadmin, cloudsqladmin, etc.)

**Returns:**
- `[]string`: List of database names
- `error`: Any error encountered during listing

**Example:**
```go
databases, err := extractor.ListDatabases(ctx)
if err != nil {
    log.Fatal(err)
}

for _, dbName := range databases {
    fmt.Println(dbName)
}
```

### ExtractSchema

```go
func (e *Extractor) ExtractSchema(ctx context.Context) (*schemaextract.DatabaseSchema, error)
```

Extracts the complete schema for the configured database using a single read-only transaction for consistency.

**Returns:**
- `*schemaextract.DatabaseSchema`: Complete schema metadata
- `error`: Any error encountered during extraction

**Example:**
```go
schema, err := extractor.ExtractSchema(ctx)
if err != nil {
    log.Fatal(err)
}

// Access schema data
for _, s := range schema.Schemas {
    fmt.Printf("Schema: %s has %d tables\n", s.Name, len(s.Tables))

    for _, table := range s.Tables {
        fmt.Printf("  Table: %s (%d columns)\n", table.Name, len(table.Columns))
    }
}
```

## Extracted Objects

### Multiple Schemas

PostgreSQL supports multiple schemas (namespaces) per database:

```go
type Schema struct {
    Name               string
    Tables             []*Table
    Views              []*View
    MaterializedViews  []*MaterializedView
    Functions          []*Function
    Procedures         []*Procedure
    Triggers           []*Trigger
    Sequences          []*Sequence
    Extensions         []*Extension
    EnumTypes          []*EnumType
}
```

**Example: Listing all schemas**
```go
for _, s := range schema.Schemas {
    fmt.Printf("Schema: %s\n", s.Name)
    fmt.Printf("  Tables: %d\n", len(s.Tables))
    fmt.Printf("  Views: %d\n", len(s.Views))
    fmt.Printf("  Functions: %d\n", len(s.Functions))
}
```

### Tables

Complete table metadata:

```go
type Table struct {
    Name             string
    Columns          []*Column
    Indexes          []*Index
    ForeignKeys      []*ForeignKey
    CheckConstraints []*CheckConstraint
    Triggers         []*Trigger
    Partitions       []*Partition    // PostgreSQL 10+
    Comment          string
    RowCount         int64            // Approximate from pg_class
    DataSize         int64            // pg_relation_size
    IndexSize        int64            // pg_indexes_size
}
```

**Example: Finding large tables**
```go
for _, s := range schema.Schemas {
    for _, table := range s.Tables {
        totalMB := float64(table.DataSize+table.IndexSize) / 1024 / 1024
        if totalMB > 1000 {
            fmt.Printf("Large table: %s.%s (%.2f MB)\n", s.Name, table.Name, totalMB)
        }
    }
}
```

### Columns

Full column metadata with PostgreSQL-specific features:

```go
type Column struct {
    Name       string
    Type       string       // PostgreSQL data type
    Nullable   bool
    Default    string       // Default value expression
    Comment    string
    Collation  string
    Position   int          // Ordinal position (1-based)
    Generation *Generation  // Generated column (PostgreSQL 12+)
}
```

**PostgreSQL-specific types:**
- Array types: `integer[]`, `text[]`
- JSON types: `json`, `jsonb`
- Network types: `inet`, `cidr`, `macaddr`
- Geometric types: `point`, `line`, `polygon`
- Range types: `int4range`, `tsrange`, `daterange`
- Custom enum types
- Composite types

**Example: Finding JSONB columns**
```go
for _, s := range schema.Schemas {
    for _, table := range s.Tables {
        for _, col := range table.Columns {
            if col.Type == "jsonb" {
                fmt.Printf("JSONB column: %s.%s.%s\n", s.Name, table.Name, col.Name)
            }
        }
    }
}
```

### Indexes

Index metadata with PostgreSQL-specific features:

```go
type Index struct {
    Name     string
    Type     string   // btree, hash, gist, gin, brin, spgist
    Unique   bool
    Primary  bool
    Columns  []string
    Comment  string
}
```

**PostgreSQL Index Types:**
- **btree**: Default, supports ordering and range queries
- **hash**: Equality comparisons only
- **gist**: Generalized search tree (geometry, full-text)
- **gin**: Generalized inverted index (arrays, JSONB, full-text)
- **brin**: Block range index (large tables with natural ordering)
- **spgist**: Space-partitioned GiST (non-balanced structures)

**Example: Finding specialized indexes**
```go
for _, s := range schema.Schemas {
    for _, table := range s.Tables {
        for _, idx := range table.Indexes {
            if idx.Type == "gin" {
                fmt.Printf("GIN index: %s.%s.%s\n", s.Name, table.Name, idx.Name)
            }
        }
    }
}
```

### Views

Regular views:

```go
type View struct {
    Name       string
    Definition string    // View definition (SELECT statement)
    Comment    string
    Columns    []*Column
}
```

### Materialized Views

Materialized views with refresh capability:

```go
type MaterializedView struct {
    Name       string
    Definition string    // View definition
    Comment    string
    Columns    []*Column
}
```

**Example: Listing materialized views**
```go
for _, s := range schema.Schemas {
    for _, mv := range s.MaterializedViews {
        fmt.Printf("Materialized view: %s.%s\n", s.Name, mv.Name)
        fmt.Printf("  Definition: %s\n", mv.Definition)
    }
}
```

### Functions

PL/pgSQL and other language functions:

```go
type Function struct {
    Name       string
    Definition string    // Complete function definition (CREATE FUNCTION ...)
    Comment    string
}
```

**System Function Filtering:**

The extractor automatically filters out 900+ built-in PostgreSQL functions including:
- Mathematical functions (abs, ceil, floor, etc.)
- String functions (concat, substring, etc.)
- Date/time functions (now, date_trunc, etc.)
- Aggregate functions (sum, avg, count, etc.)
- Window functions
- System information functions

**Example: Listing user-defined functions**
```go
for _, s := range schema.Schemas {
    for _, fn := range s.Functions {
        fmt.Printf("Function: %s.%s\n", s.Name, fn.Name)
    }
}
```

### Procedures

Stored procedures (PostgreSQL 11+):

```go
type Procedure struct {
    Name       string
    Definition string    // Complete procedure definition
    Comment    string
}
```

**Note:** Procedures require PostgreSQL 11+. On older versions, this list will be empty.

### Sequences

Sequence generators:

```go
type Sequence struct {
    Name      string
    DataType  string  // bigint, integer, smallint
    Start     int64   // Start value
    Increment int64   // Increment by
    MinValue  int64   // Minimum value
    MaxValue  int64   // Maximum value
    Cache     int64   // Cache size
    Cycle     bool    // Cycle when reaching max/min
    Comment   string
}
```

**Example: Listing sequences**
```go
for _, s := range schema.Schemas {
    for _, seq := range s.Sequences {
        fmt.Printf("Sequence: %s.%s (start: %d, increment: %d)\n",
            s.Name, seq.Name, seq.Start, seq.Increment)
    }
}
```

### Extensions

Installed PostgreSQL extensions:

```go
type Extension struct {
    Name    string
    Version string
    Comment string
}
```

**Common Extensions:**
- postgis - Spatial and geographic objects
- pg_trgm - Trigram matching for text search
- uuid-ossp - UUID generation
- hstore - Key-value store
- citext - Case-insensitive text type
- pgcrypto - Cryptographic functions

**Example: Listing extensions**
```go
for _, s := range schema.Schemas {
    for _, ext := range s.Extensions {
        fmt.Printf("Extension: %s (version: %s)\n", ext.Name, ext.Version)
    }
}
```

### Enum Types

Custom enumeration types:

```go
type EnumType struct {
    Name   string
    Values []string  // Ordered list of enum values
}
```

**Example: Listing enum types**
```go
for _, s := range schema.Schemas {
    for _, enum := range s.EnumTypes {
        fmt.Printf("Enum: %s.%s = %v\n", s.Name, enum.Name, enum.Values)
    }
}
```

### Triggers

Table triggers:

```go
type Trigger struct {
    Name    string
    Timing  string  // BEFORE, AFTER, INSTEAD OF
    Event   string  // INSERT, UPDATE, DELETE, TRUNCATE
    Body    string  // Trigger function call
    Comment string
}
```

### Partitions

Table partitions (PostgreSQL 10+):

```go
type Partition struct {
    Name        string
    Type        string      // RANGE, LIST, HASH
    Expression  string      // Partitioning expression
    Description string      // Partition bounds
    Comment     string
}
```

**Supported Partition Types:**
- RANGE - Partition by value ranges
- LIST - Partition by value lists
- HASH - Partition by hash of values

## System Object Filtering

The extractor automatically filters out:

### System Schemas
- pg_catalog
- information_schema
- pg_toast
- pg_temp_*
- pg_toast_temp_*

### Cloud Provider Schemas
- **AWS RDS**: rdsadmin
- **GCP CloudSQL**: cloudsqladmin, cloudsqlsuperuser
- **AlloyDB**: alloydbsuperuser, alloydbadmin

### Extension Schemas
- TimescaleDB: _timescaledb_*
- Citus: citus, citus_internal
- Partman: partman
- pg_cron: cron
- And many more...

### System Functions

900+ built-in functions are filtered including all functions from:
- pg_catalog schema
- Common extensions (postgis, timescaledb, etc.)
- Internal functions

## Version Support

### PostgreSQL 10+

Full support for:
- Table partitions (RANGE, LIST, HASH)
- Generated columns (PostgreSQL 12+)
- Procedures (PostgreSQL 11+)
- All standard objects

### PostgreSQL 9.6 and Earlier

Not officially supported, but may work with reduced functionality:
- No partition extraction
- No procedures
- No generated columns

## Connection Strings

### Basic Connection

```go
db, err := sql.Open("postgres", "postgres://user:password@localhost/mydb")
```

### With SSL

```go
db, err := sql.Open("postgres",
    "postgres://user:password@localhost/mydb?sslmode=require")
```

**SSL Modes:**
- `disable` - No SSL
- `require` - Require SSL (no verification)
- `verify-ca` - Verify CA certificate
- `verify-full` - Verify CA and hostname

### With Timeout

```go
db, err := sql.Open("postgres",
    "postgres://user:password@localhost/mydb?connect_timeout=30")
```

### With Application Name

```go
db, err := sql.Open("postgres",
    "postgres://user:password@localhost/mydb?application_name=schema_extractor")
```

### Full Example with Configuration

```go
import (
    "database/sql"
    "time"
)

connStr := "postgres://user:password@localhost:5432/mydb" +
    "?sslmode=require" +
    "&connect_timeout=30" +
    "&application_name=schema_extractor"

db, err := sql.Open("postgres", connStr)
if err != nil {
    log.Fatal(err)
}

// Configure connection pool
db.SetMaxOpenConns(10)
db.SetMaxIdleConns(5)
db.SetConnMaxLifetime(time.Hour)
```

## Required Permissions

The PostgreSQL user needs SELECT permission on system catalogs:

```sql
-- Grant access to system catalogs
GRANT SELECT ON ALL TABLES IN SCHEMA pg_catalog TO schema_user;

-- Grant access to user schemas
GRANT USAGE ON SCHEMA public TO schema_user;
GRANT SELECT ON ALL TABLES IN SCHEMA public TO schema_user;

-- For listing all databases
GRANT CONNECT ON DATABASE template1 TO schema_user;
```

## Common Use Cases

### Find Tables by Schema

```go
schema, _ := extractor.ExtractSchema(ctx)

publicTables := 0
for _, s := range schema.Schemas {
    if s.Name == "public" {
        publicTables = len(s.Tables)
        break
    }
}

fmt.Printf("Public schema has %d tables\n", publicTables)
```

### List All Partitioned Tables

```go
for _, s := range schema.Schemas {
    for _, table := range s.Tables {
        if len(table.Partitions) > 0 {
            fmt.Printf("Partitioned: %s.%s (%s, %d partitions)\n",
                s.Name, table.Name,
                table.Partitions[0].Type, len(table.Partitions))
        }
    }
}
```

### Find Foreign Key Relationships

```go
for _, s := range schema.Schemas {
    for _, table := range s.Tables {
        for _, fk := range table.ForeignKeys {
            fmt.Printf("%s.%s.%s → %s.%s (ON DELETE %s)\n",
                s.Name, table.Name,
                strings.Join(fk.Columns, ","),
                fk.ReferencedTable,
                strings.Join(fk.ReferencedColumns, ","),
                fk.OnDelete)
        }
    }
}
```

### Analyze Extensions

```go
for _, s := range schema.Schemas {
    for _, ext := range s.Extensions {
        fmt.Printf("Extension: %s v%s\n", ext.Name, ext.Version)
        if ext.Comment != "" {
            fmt.Printf("  Description: %s\n", ext.Comment)
        }
    }
}
```

### Find Materialized Views That Need Refresh

```go
// List all materialized views
for _, s := range schema.Schemas {
    for _, mv := range s.MaterializedViews {
        fmt.Printf("Materialized view: %s.%s\n", s.Name, mv.Name)
        fmt.Println("  Run: REFRESH MATERIALIZED VIEW " + s.Name + "." + mv.Name)
    }
}
```

### Export Schema Structure as Markdown

```go
for _, s := range schema.Schemas {
    fmt.Printf("# Schema: %s\n\n", s.Name)

    fmt.Println("## Tables\n")
    for _, table := range s.Tables {
        fmt.Printf("### %s\n\n", table.Name)
        if table.Comment != "" {
            fmt.Printf("%s\n\n", table.Comment)
        }

        fmt.Println("| Column | Type | Nullable | Default |")
        fmt.Println("|--------|------|----------|---------|")
        for _, col := range table.Columns {
            nullable := "NO"
            if col.Nullable {
                nullable = "YES"
            }
            fmt.Printf("| %s | %s | %s | %s |\n",
                col.Name, col.Type, nullable, col.Default)
        }
        fmt.Println()
    }
}
```

## Troubleshooting

### Error: "permission denied for table pg_proc"

Your user doesn't have access to system catalogs.

**Solution:**
```sql
GRANT SELECT ON ALL TABLES IN SCHEMA pg_catalog TO your_user;
```

### Error: "database does not exist"

The specified database name doesn't exist.

**Solution:**
```sql
-- List all databases
SELECT datname FROM pg_database WHERE datistemplate = false;
```

### Error: "connection refused"

Cannot connect to PostgreSQL server.

**Solution:**
- Check PostgreSQL is running: `pg_ctl status`
- Verify postgresql.conf has `listen_addresses = '*'` or appropriate IP
- Check pg_hba.conf allows connections from your host
- Verify firewall rules allow port 5432

### Error: "SSL is required"

Server requires SSL but connection doesn't use it.

**Solution:**
```go
db, err := sql.Open("postgres",
    "postgres://user:password@localhost/mydb?sslmode=require")
```

### Slow Extraction

Extraction takes a long time for large databases.

**Causes:**
- Many tables (1000+)
- Large view/function definitions
- Heavy database load

**Solutions:**
- Extract during off-peak hours
- Increase timeout: `?connect_timeout=120`
- Use connection pooling
- Consider extracting specific schemas only

## Performance Considerations

### Transaction Consistency

All extraction happens in a single read-only transaction, ensuring a consistent snapshot even while the database is being modified.

### Extraction Speed

Typical extraction times:

| Objects | Extraction Time |
|---------|----------------|
| 100 tables, 50 functions | ~100ms |
| 500 tables, 100 functions | ~400ms |
| 1000 tables, 500 functions | ~2s |
| 5000 tables, 1000 functions | ~10s |

### Memory Usage

Memory usage scales with database size:
- 100 tables: ~10MB
- 1000 tables: ~50MB
- 5000 tables: ~200MB

## Examples

See the [examples](../examples/postgres) directory for complete working examples.

## Related Documentation

- [Main Package README](../README.md)
- [MySQL Extractor](../mysql/README.md)
- [Type Definitions](../types.go)
- [System Objects Filter](system_objects.go)
