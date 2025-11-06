// Package snapshot provides point-in-time schema snapshots with versioning
// and comparison capabilities.
//
// # Overview
//
// The snapshot package wraps DatabaseSchema objects with metadata for
// tracking and management. It enables:
//   - Creating timestamped schema snapshots
//   - Storing snapshots in JSON/YAML format
//   - Loading snapshots for comparison
//   - Calculating checksums (MD5, SHA256)
//   - Tagging snapshots with custom metadata
//
// # Usage
//
// Create a snapshot:
//
//	snapshot := snapshot.Create(schema, snapshot.Options{
//	    Tags: map[string]string{
//	        "environment": "production",
//	        "version":     "v2.5.0",
//	    },
//	    Description: "Pre-migration snapshot",
//	})
//
// Save snapshot to file:
//
//	err := snapshot.Save("snapshots/prod-2024-01-15.json")
//
// Load and compare snapshots:
//
//	old, _ := snapshot.Load("snapshots/prod-2024-01-15.json")
//	new, _ := snapshot.Load("snapshots/prod-2024-01-22.json")
//	report, _ := snapshot.CompareSnapshots(old, new, nil)
//
// # Snapshot Metadata
//
// Each snapshot includes:
//   - Timestamp: When the snapshot was created
//   - Version: Snapshot format version
//   - DatabaseEngine: "mysql" or "postgres"
//   - Tags: User-defined key-value pairs
//   - Checksums: MD5 and SHA256 hashes
//
// # Use Cases
//
//   - Schema versioning and history tracking
//   - Pre/post migration validation
//   - Multi-environment schema comparison
//   - Disaster recovery and rollback planning
//   - Compliance and audit trails
//
// # File Format
//
// Snapshots are stored as JSON or YAML with the following structure:
//
//	{
//	  "metadata": {
//	    "timestamp": "2024-01-15T10:30:00Z",
//	    "version": "1.0",
//	    "databaseEngine": "postgres",
//	    "tags": {"environment": "production"}
//	  },
//	  "schema": { /* DatabaseSchema */ }
//	}
package snapshot
