# MySQL Schema Extractor

Complete MySQL schema metadata extraction with support for MySQL 5.7+, MySQL 8.0+, and MariaDB 10.3+.

## Features

- **Complete Schema Extraction**: All database objects and metadata
- **Version Detection**: Automatic MySQL/MariaDB version detection
- **Type Normalization**: Canonicalizes MySQL type synonyms
- **Partition Support**: All partition types (RANGE, LIST, HASH, KEY)
- **Generated Columns**: Both VIRTUAL and STORED columns
- **Storage Engines**: InnoDB, MyISAM, and others
- **Statistics**: Row counts, data sizes, index sizes

## Quick Start

```go
package main

import (
    "context"
    "database/sql"
    "fmt"
    "log"

    _ "github.com/go-sql-driver/mysql"
    "github.com/nsxbet/sql-schema/mysql"
)

func main() {
    db, err := sql.Open("mysql", "user:password@tcp(localhost:3306)/mydb")
    if err != nil {
        log.Fatal(err)
    }
    defer db.Close()

    extractor := mysql.NewExtractor(db, "mydb")

    schema, err := extractor.ExtractSchema(context.Background())
    if err != nil {
        log.Fatal(err)
    }

    fmt.Printf("Database: %s\n", schema.Name)
    fmt.Printf("Character Set: %s\n", schema.CharacterSet)
    fmt.Printf("Collation: %s\n", schema.Collation)
}
```

## API

### NewExtractor

```go
func NewExtractor(db *sql.DB, databaseName string) *Extractor
```

Creates a new MySQL schema extractor.

**Parameters:**
- `db`: Active `*sql.DB` connection to the MySQL server
- `databaseName`: Name of the database to extract

**Returns:**
- `*Extractor`: Ready-to-use extractor instance

### ListDatabases

```go
func (e *Extractor) ListDatabases(ctx context.Context) ([]string, error)
```

Returns a list of all databases on the MySQL server.

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

Extracts the complete schema for the configured database.

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
    fmt.Printf("Tables: %d\n", len(s.Tables))

    for _, table := range s.Tables {
        fmt.Printf("Table: %s (%d columns)\n", table.Name, len(table.Columns))
    }
}
```

## Extracted Objects

### Tables

Complete table metadata including:

```go
type Table struct {
    Name             string            // Table name
    Columns          []*Column         // All columns
    Indexes          []*Index          // All indexes (including PRIMARY)
    ForeignKeys      []*ForeignKey     // Foreign key constraints
    CheckConstraints []*CheckConstraint // Check constraints (MySQL 8.0.16+)
    Triggers         []*Trigger        // Associated triggers
    Partitions       []*Partition      // Table partitions
    Comment          string            // Table comment
    Engine           string            // Storage engine (InnoDB, MyISAM, etc.)
    Collation        string            // Table collation
    RowCount         int64             // Approximate row count
    DataSize         int64             // Data size in bytes
    IndexSize        int64             // Index size in bytes
    DataFree         int64             // Free space in bytes
}
```

**Example: Finding tables by storage engine**
```go
for _, table := range schema.Schemas[0].Tables {
    if table.Engine == "MyISAM" {
        fmt.Printf("Table %s uses MyISAM engine\n", table.Name)
    }
}
```

### Columns

Full column metadata with type information:

```go
type Column struct {
    Name       string       // Column name
    Type       string       // Canonical data type
    Nullable   bool         // NULL allowed
    Default    string       // Default value expression
    Comment    string       // Column comment
    Collation  string       // Column collation
    Position   int          // Ordinal position (1-based)
    Generation *Generation  // Generated column info (VIRTUAL/STORED)
}
```

**Type Canonicalization:**

MySQL type synonyms are automatically normalized:

| Input Type | Canonical Type |
|-----------|---------------|
| BOOLEAN, BOOL | TINYINT(1) |
| INTEGER | INT |
| DEC, NUMERIC, FIXED | DECIMAL |
| DOUBLE PRECISION | DOUBLE |
| REAL | DOUBLE |

**Example: Finding generated columns**
```go
for _, table := range schema.Schemas[0].Tables {
    for _, col := range table.Columns {
        if col.Generation != nil {
            fmt.Printf("Column %s.%s is generated (%s): %s\n",
                table.Name, col.Name, col.Generation.Type, col.Generation.Expression)
        }
    }
}
```

### Indexes

Index metadata including type and columns:

```go
type Index struct {
    Name     string   // Index name
    Type     string   // Index type (BTREE, HASH, FULLTEXT, SPATIAL)
    Unique   bool     // Unique constraint
    Primary  bool     // Primary key
    Columns  []string // Indexed column names
    Comment  string   // Index comment
}
```

**Example: Finding tables without primary keys**
```go
for _, table := range schema.Schemas[0].Tables {
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
```

### Foreign Keys

Foreign key relationships:

```go
type ForeignKey struct {
    Name            string   // Constraint name
    Columns         []string // Local column names
    ReferencedTable string   // Referenced table name
    ReferencedColumns []string // Referenced column names
    OnUpdate        string   // ON UPDATE action
    OnDelete        string   // ON DELETE action
}
```

### Views

View definitions and metadata:

```go
type View struct {
    Name       string    // View name
    Definition string    // View definition (SELECT statement)
    Comment    string    // View comment
    Columns    []*Column // View columns
}
```

### Functions

Stored functions:

```go
type Function struct {
    Name       string // Function name
    Definition string // Complete function definition
}
```

**Example: Listing all functions**
```go
for _, fn := range schema.Schemas[0].Functions {
    fmt.Printf("Function: %s\n", fn.Name)
    fmt.Printf("Definition:\n%s\n\n", fn.Definition)
}
```

### Procedures

Stored procedures:

```go
type Procedure struct {
    Name       string // Procedure name
    Definition string // Complete procedure definition
}
```

### Triggers

Table triggers:

```go
type Trigger struct {
    Name    string // Trigger name
    Timing  string // BEFORE or AFTER
    Event   string // INSERT, UPDATE, DELETE
    Body    string // Trigger body
    Comment string // Trigger comment
}
```

### Events

Scheduled events:

```go
type Event struct {
    Name       string // Event name
    Definition string // Complete event definition
}
```

### Partitions

Table partition information:

```go
type Partition struct {
    Name        string // Partition name
    Type        string // RANGE, LIST, HASH, KEY
    Expression  string // Partitioning expression
    Description string // Partition value description
    Comment     string // Partition comment
}
```

**Supported Partition Types:**
- RANGE
- LIST
- HASH
- KEY
- RANGE COLUMNS
- LIST COLUMNS

**Example: Finding partitioned tables**
```go
for _, table := range schema.Schemas[0].Tables {
    if len(table.Partitions) > 0 {
        fmt.Printf("Table %s is partitioned (%s) with %d partitions\n",
            table.Name, table.Partitions[0].Type, len(table.Partitions))
    }
}
```

## Version Support

### MySQL 5.7

Supports all features except:
- Check constraints (added in 8.0.16)

### MySQL 8.0+

Full support for all features including:
- Check constraints
- Invisible columns
- Role-based access control metadata

### MariaDB 10.3+

Compatible with MySQL 5.7 feature set.

## Connection Strings

### Basic Connection

```go
db, err := sql.Open("mysql", "user:password@tcp(host:3306)/database")
```

### With SSL/TLS

```go
db, err := sql.Open("mysql", "user:password@tcp(host:3306)/database?tls=true")
```

### With Timeout

```go
db, err := sql.Open("mysql", "user:password@tcp(host:3306)/database?timeout=30s&readTimeout=60s")
```

### With Character Set

```go
db, err := sql.Open("mysql", "user:password@tcp(host:3306)/database?charset=utf8mb4&collation=utf8mb4_unicode_ci")
```

### Full Example with Configuration

```go
import (
    "database/sql"
    "time"
)

db, err := sql.Open("mysql",
    "user:password@tcp(localhost:3306)/mydb"+
    "?timeout=30s"+
    "&readTimeout=60s"+
    "&writeTimeout=60s"+
    "&parseTime=true"+
    "&charset=utf8mb4"+
    "&collation=utf8mb4_unicode_ci")

if err != nil {
    log.Fatal(err)
}

// Configure connection pool
db.SetMaxOpenConns(10)
db.SetMaxIdleConns(5)
db.SetConnMaxLifetime(time.Hour)
```

## Required Permissions

The MySQL user needs the following privileges:

```sql
-- Minimum required permissions
GRANT SELECT ON mysql.* TO 'schema_user'@'%';
GRANT SELECT ON information_schema.* TO 'schema_user'@'%';
GRANT SELECT ON your_database.* TO 'schema_user'@'%';

-- For listing all databases
GRANT SHOW DATABASES ON *.* TO 'schema_user'@'%';
```

## Common Use Cases

### Find Large Tables

```go
schema, _ := extractor.ExtractSchema(ctx)

for _, table := range schema.Schemas[0].Tables {
    sizeMB := float64(table.DataSize + table.IndexSize) / 1024 / 1024
    if sizeMB > 1000 { // Tables larger than 1GB
        fmt.Printf("Large table: %s (%.2f MB)\n", table.Name, sizeMB)
    }
}
```

### Find Tables Without Indexes

```go
for _, table := range schema.Schemas[0].Tables {
    if len(table.Indexes) == 0 {
        fmt.Printf("No indexes: %s\n", table.Name)
    } else if len(table.Indexes) == 1 && table.Indexes[0].Primary {
        fmt.Printf("Only primary key: %s\n", table.Name)
    }
}
```

### List Foreign Key Relationships

```go
for _, table := range schema.Schemas[0].Tables {
    for _, fk := range table.ForeignKeys {
        fmt.Printf("%s.%s → %s.%s (ON DELETE %s)\n",
            table.Name,
            strings.Join(fk.Columns, ","),
            fk.ReferencedTable,
            strings.Join(fk.ReferencedColumns, ","),
            fk.OnDelete)
    }
}
```

### Analyze Character Sets and Collations

```go
schema, _ := extractor.ExtractSchema(ctx)

fmt.Printf("Database default: %s / %s\n", schema.CharacterSet, schema.Collation)

for _, table := range schema.Schemas[0].Tables {
    if table.Collation != schema.Collation {
        fmt.Printf("Table %s uses different collation: %s\n", table.Name, table.Collation)
    }
}
```

### Export Table Statistics

```go
import "encoding/csv"

file, _ := os.Create("table-stats.csv")
writer := csv.NewWriter(file)
defer writer.Flush()

writer.Write([]string{"Table", "Rows", "Data Size (MB)", "Index Size (MB)", "Total (MB)"})

for _, table := range schema.Schemas[0].Tables {
    dataMB := float64(table.DataSize) / 1024 / 1024
    indexMB := float64(table.IndexSize) / 1024 / 1024
    totalMB := dataMB + indexMB

    writer.Write([]string{
        table.Name,
        fmt.Sprintf("%d", table.RowCount),
        fmt.Sprintf("%.2f", dataMB),
        fmt.Sprintf("%.2f", indexMB),
        fmt.Sprintf("%.2f", totalMB),
    })
}
```

## Troubleshooting

### Error: "Access denied for user"

Your user doesn't have sufficient permissions.

**Solution:**
```sql
GRANT SELECT ON mysql.* TO 'your_user'@'%';
GRANT SELECT ON information_schema.* TO 'your_user'@'%';
GRANT SHOW DATABASES ON *.* TO 'your_user'@'%';
FLUSH PRIVILEGES;
```

### Error: "dial tcp: i/o timeout"

Connection timeout to MySQL server.

**Solution:**
- Check firewall rules
- Verify MySQL is listening on the correct port
- Increase timeout in connection string: `?timeout=60s`

### Error: "Too many connections"

MySQL has reached `max_connections` limit.

**Solution:**
- Configure connection pool limits:
```go
db.SetMaxOpenConns(10)
db.SetMaxIdleConns(5)
```
- Or increase MySQL `max_connections` setting

### Error: "Table doesn't exist in engine"

Table definition exists but data files are missing.

**Solution:**
- This indicates corrupted or missing table files
- Use `CHECK TABLE` to diagnose
- Restore from backup if necessary

## Performance Considerations

### Extraction Speed

Typical extraction times for different database sizes:

| Tables | Views | Functions | Extraction Time |
|--------|-------|-----------|----------------|
| 10 | 5 | 10 | ~50ms |
| 100 | 20 | 50 | ~200ms |
| 500 | 50 | 100 | ~1s |
| 1000 | 100 | 200 | ~2s |

### Memory Usage

Memory usage scales with:
- Number of tables
- Number of columns per table
- Size of view/function/procedure definitions

For databases with 1000+ tables, expect ~50-100MB memory usage during extraction.

### Optimization Tips

1. **Use connection pooling** to reuse connections
2. **Set appropriate timeouts** for large databases
3. **Extract during off-peak hours** if database is under heavy load
4. **Consider filtering** if you only need specific objects

## Examples

See the [examples](../examples/mysql) directory for complete working examples.

## Related Documentation

- [Main Package README](../README.md)
- [PostgreSQL Extractor](../postgres/README.md)
- [Type Definitions](../types.go)
