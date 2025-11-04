# Schema Snapshot Example

This example demonstrates how to use the snapshot functionality to capture, save, load, and compare database schema snapshots.

## Features Demonstrated

1. **Taking Snapshots** - Extract database schema and create snapshots with metadata
2. **Saving Snapshots** - Save snapshots in JSON or YAML format
3. **Loading Snapshots** - Load snapshots from files
4. **Validating Snapshots** - Verify snapshot integrity using checksums
5. **Comparing Snapshots** - Compare two snapshots to detect schema changes
6. **Listing Snapshots** - List all snapshots in a directory
7. **Exporting SQL** - Export snapshots as SQL DDL statements

## Running the Example

```bash
# Install dependencies
go mod download

# Run the example
go run main.go
```

## Code Walkthrough

### 1. Taking a Snapshot

```go
import (
    "database/sql"
    "github.com/nsxbet/sql-schema/comparer/engine"
    "github.com/nsxbet/sql-schema/snapshot"
)

// Connect to database
db, _ := sql.Open("postgres", "postgresql://user:password@localhost:5432/mydb")

// Take snapshot with options
snap, err := snapshot.TakeSnapshot(db, "mydb", engine.PostgreSQL, &snapshot.SnapshotOptions{
    Version:     "1.0",
    Description: "Production database snapshot before migration",
    Tags: map[string]string{
        "environment": "production",
        "purpose":     "pre-migration",
    },
    ComputeHash: true, // Compute checksums for integrity verification
})
```

### 2. Saving and Loading Snapshots

```go
// Save as JSON
err := snapshot.SaveSnapshot(snap, "snapshots/mydb_production.json")

// Save as YAML
err := snapshot.SaveSnapshot(snap, "snapshots/mydb_production.yaml")

// Load snapshot
loaded, err := snapshot.LoadSnapshot("snapshots/mydb_production.json")
```

### 3. Validating Snapshots

```go
// Validate snapshot integrity
err := snapshot.ValidateSnapshot("snapshots/mydb_production.json")
if err != nil {
    log.Printf("Validation failed: %v", err)
}
```

### 4. Comparing Snapshots

```go
// Compare two snapshots
mdiff, err := snapshot.LoadAndCompare(
    "snapshots/mydb_before.json",
    "snapshots/mydb_after.json",
    nil, // Auto-detect engine from snapshots
)

// Generate comparison report
report := formatter.FormatDiffAsText(mdiff)
fmt.Println(report)

// Analyze migration strategy
strategy, _ := planner.AnalyzeStrategy(mdiff, engine.PostgreSQL)
strategyReport := formatter.FormatStrategyAsText(strategy)
fmt.Println(strategyReport)
```

### 5. Listing Snapshots

```go
// List all snapshots in a directory
snapshots, err := snapshot.ListSnapshots("snapshots/")

for _, info := range snapshots {
    fmt.Printf("Database: %s (%s)\n", info.DatabaseName, info.DatabaseEngine)
    fmt.Printf("Timestamp: %s\n", info.Timestamp)
    fmt.Printf("Size: %d bytes\n", info.FileSize)
}
```

### 6. Exporting as SQL

```go
// Export snapshot as SQL DDL
err := snapshot.ExportSnapshotSQL(snap, "snapshots/mydb_production.sql")
```

## Use Cases

### Pre-Migration Snapshot

Take a snapshot before running database migrations to have a rollback point:

```go
// Before migration
beforeSnapshot, _ := snapshot.TakeSnapshot(db, "mydb", engine.PostgreSQL, &snapshot.SnapshotOptions{
    Description: "Pre-migration snapshot",
    Tags: map[string]string{
        "migration": "v2.0",
        "stage":     "before",
    },
    ComputeHash: true,
})
snapshot.SaveSnapshot(beforeSnapshot, "snapshots/pre_migration_v2.0.json")

// Run migrations...

// After migration
afterSnapshot, _ := snapshot.TakeSnapshot(db, "mydb", engine.PostgreSQL, &snapshot.SnapshotOptions{
    Description: "Post-migration snapshot",
    Tags: map[string]string{
        "migration": "v2.0",
        "stage":     "after",
    },
    ComputeHash: true,
})
snapshot.SaveSnapshot(afterSnapshot, "snapshots/post_migration_v2.0.json")

// Compare changes
mdiff, _ := snapshot.CompareSnapshots(beforeSnapshot, afterSnapshot, nil)
```

### Environment Comparison

Compare schemas across different environments:

```go
// Production snapshot
prodSnapshot, _ := snapshot.LoadSnapshot("snapshots/production.json")

// Staging snapshot
stagingSnapshot, _ := snapshot.LoadSnapshot("snapshots/staging.json")

// Compare
mdiff, _ := snapshot.CompareSnapshots(prodSnapshot, stagingSnapshot, nil)
```

### Schema Versioning

Track schema changes over time:

```go
// Daily snapshot
dailySnapshot, _ := snapshot.TakeSnapshot(db, "mydb", engine.PostgreSQL, &snapshot.SnapshotOptions{
    Version:     "1.0",
    Description: fmt.Sprintf("Daily snapshot %s", time.Now().Format("2006-01-02")),
    Tags: map[string]string{
        "frequency": "daily",
        "date":      time.Now().Format("2006-01-02"),
    },
    ComputeHash: true,
})

filename := fmt.Sprintf("snapshots/daily_%s.json", time.Now().Format("2006-01-02"))
snapshot.SaveSnapshot(dailySnapshot, filename)
```

## Directory Structure

```
examples/snapshot/
├── README.md           # This file
└── main.go            # Example code
```

## Related Documentation

- [Main README](../../README.md) - Project overview
- [Snapshot Package](../../snapshot/) - Snapshot utilities
- [Exporter Package](../../exporter/) - Schema export functions
- [Comparer Package](../../comparer/) - Schema comparison
