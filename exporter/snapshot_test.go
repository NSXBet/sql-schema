package exporter

import (
	"bytes"
	"strings"
	"testing"
	"time"

	schemaextract "github.com/nsxbet/sql-schema"
)

func TestCreateSnapshot(t *testing.T) {
	schema := &schemaextract.DatabaseSchema{
		Name: "test_db",
		Schemas: []*schemaextract.Schema{
			{
				Name: "public",
				Tables: []*schemaextract.Table{
					{
						Name: "users",
						Columns: []*schemaextract.Column{
							{Name: "id", Type: "INTEGER", Nullable: false},
							{Name: "name", Type: "VARCHAR(255)", Nullable: false},
						},
					},
				},
			},
		},
	}

	metadata := schemaextract.SnapshotMetadata{
		Timestamp:      time.Now(),
		Version:        "1.0",
		DatabaseName:   "test_db",
		DatabaseEngine: "postgres",
		Description:    "Test snapshot",
		Tags: map[string]string{
			"environment": "test",
		},
	}

	snapshot := CreateSnapshot(schema, metadata)

	if snapshot == nil {
		t.Fatal("CreateSnapshot returned nil")
	}
	if snapshot.Schema != schema {
		t.Error("Snapshot schema doesn't match input schema")
	}
	if snapshot.Metadata.DatabaseName != "test_db" {
		t.Error("Snapshot metadata doesn't match input metadata")
	}
	if snapshot.Metadata.Tags["environment"] != "test" {
		t.Error("Snapshot tags don't match")
	}
}

func TestExportSnapshot(t *testing.T) {
	snapshot := &schemaextract.Snapshot{
		Metadata: schemaextract.SnapshotMetadata{
			Timestamp:      time.Date(2024, 1, 1, 12, 0, 0, 0, time.UTC),
			Version:        "1.0",
			DatabaseName:   "test_db",
			DatabaseEngine: "postgres",
		},
		Schema: &schemaextract.DatabaseSchema{
			Name: "test_db",
			Schemas: []*schemaextract.Schema{
				{
					Name: "public",
					Tables: []*schemaextract.Table{
						{
							Name: "users",
							Columns: []*schemaextract.Column{
								{Name: "id", Type: "INTEGER", Nullable: false},
							},
						},
					},
				},
			},
		},
	}

	var buf bytes.Buffer
	err := ExportSnapshot(snapshot, &buf)
	if err != nil {
		t.Fatalf("ExportSnapshot failed: %v", err)
	}

	output := buf.String()
	if !strings.Contains(output, "test_db") {
		t.Error("Expected database name in output")
	}
	if !strings.Contains(output, "metadata") {
		t.Error("Expected metadata in output")
	}
	if !strings.Contains(output, "schema") {
		t.Error("Expected schema in output")
	}
	if !strings.Contains(output, "postgres") {
		t.Error("Expected engine in output")
	}
}

func TestExportSnapshotYAML(t *testing.T) {
	snapshot := &schemaextract.Snapshot{
		Metadata: schemaextract.SnapshotMetadata{
			Timestamp:      time.Date(2024, 1, 1, 12, 0, 0, 0, time.UTC),
			Version:        "1.0",
			DatabaseName:   "test_db",
			DatabaseEngine: "mysql",
		},
		Schema: &schemaextract.DatabaseSchema{
			Name: "test_db",
			Schemas: []*schemaextract.Schema{
				{
					Name: "",
					Tables: []*schemaextract.Table{
						{
							Name: "users",
							Columns: []*schemaextract.Column{
								{Name: "id", Type: "INT", Nullable: false},
							},
						},
					},
				},
			},
		},
	}

	var buf bytes.Buffer
	err := ExportSnapshotYAML(snapshot, &buf)
	if err != nil {
		t.Fatalf("ExportSnapshotYAML failed: %v", err)
	}

	output := buf.String()
	if !strings.Contains(output, "test_db") {
		t.Error("Expected database name in YAML output")
	}
	if !strings.Contains(output, "metadata:") {
		t.Error("Expected metadata in YAML output")
	}
	if !strings.Contains(output, "schema:") {
		t.Error("Expected schema in YAML output")
	}
	if !strings.Contains(output, "mysql") {
		t.Error("Expected engine in YAML output")
	}
}

func TestImportSnapshot(t *testing.T) {
	jsonData := `{
  "metadata": {
    "timestamp": "2024-01-01T12:00:00Z",
    "version": "1.0",
    "databaseName": "test_db",
    "databaseEngine": "postgres",
    "description": "Test snapshot",
    "tags": {
      "environment": "test"
    }
  },
  "schema": {
    "name": "test_db",
    "schemas": [
      {
        "name": "public",
        "tables": [
          {
            "name": "users",
            "columns": [
              {
                "name": "id",
                "position": 1,
                "type": "INTEGER",
                "nullable": false
              }
            ]
          }
        ]
      }
    ]
  }
}`

	buf := bytes.NewBufferString(jsonData)
	snapshot, err := ImportSnapshot(buf)
	if err != nil {
		t.Fatalf("ImportSnapshot failed: %v", err)
	}

	if snapshot.Metadata.DatabaseName != "test_db" {
		t.Errorf("Expected database name 'test_db', got '%s'", snapshot.Metadata.DatabaseName)
	}
	if snapshot.Metadata.DatabaseEngine != "postgres" {
		t.Errorf("Expected engine 'postgres', got '%s'", snapshot.Metadata.DatabaseEngine)
	}
	if snapshot.Metadata.Version != "1.0" {
		t.Errorf("Expected version '1.0', got '%s'", snapshot.Metadata.Version)
	}
	if snapshot.Metadata.Description != "Test snapshot" {
		t.Errorf("Expected description 'Test snapshot', got '%s'", snapshot.Metadata.Description)
	}
	if snapshot.Metadata.Tags["environment"] != "test" {
		t.Error("Expected tag 'environment' = 'test'")
	}
	if snapshot.Schema.Name != "test_db" {
		t.Error("Expected schema name 'test_db'")
	}
	if len(snapshot.Schema.Schemas) != 1 {
		t.Errorf("Expected 1 schema, got %d", len(snapshot.Schema.Schemas))
	}
	if len(snapshot.Schema.Schemas[0].Tables) != 1 {
		t.Errorf("Expected 1 table, got %d", len(snapshot.Schema.Schemas[0].Tables))
	}
}

func TestImportSnapshotYAML(t *testing.T) {
	yamlData := `metadata:
  timestamp: 2024-01-01T12:00:00Z
  version: "1.0"
  databaseName: test_db
  databaseEngine: postgres
  description: Test snapshot
schema:
  name: test_db
  schemas:
    - name: public
      tables:
        - name: users
          columns:
            - name: id
              position: 1
              type: INTEGER
              nullable: false
`

	buf := bytes.NewBufferString(yamlData)
	snapshot, err := ImportSnapshotYAML(buf)
	if err != nil {
		t.Fatalf("ImportSnapshotYAML failed: %v", err)
	}

	if snapshot.Metadata.DatabaseName != "test_db" {
		t.Errorf("Expected database name 'test_db', got '%s'", snapshot.Metadata.DatabaseName)
	}
	if snapshot.Metadata.DatabaseEngine != "postgres" {
		t.Errorf("Expected engine 'postgres', got '%s'", snapshot.Metadata.DatabaseEngine)
	}
	if snapshot.Schema.Name != "test_db" {
		t.Error("Expected schema name 'test_db'")
	}
}

func TestSnapshotRoundTrip(t *testing.T) {
	// Create original snapshot
	original := &schemaextract.Snapshot{
		Metadata: schemaextract.SnapshotMetadata{
			Timestamp:      time.Date(2024, 1, 1, 12, 0, 0, 0, time.UTC),
			Version:        "1.0",
			DatabaseName:   "test_db",
			DatabaseEngine: "postgres",
			Description:    "Round trip test",
			Tags: map[string]string{
				"test": "roundtrip",
			},
		},
		Schema: &schemaextract.DatabaseSchema{
			Name: "test_db",
			Schemas: []*schemaextract.Schema{
				{
					Name: "public",
					Tables: []*schemaextract.Table{
						{
							Name: "users",
							Columns: []*schemaextract.Column{
								{Name: "id", Type: "INTEGER", Nullable: false},
								{Name: "email", Type: "VARCHAR(255)", Nullable: true},
							},
						},
					},
				},
			},
		},
	}

	// Export to JSON
	var buf bytes.Buffer
	err := ExportSnapshot(original, &buf)
	if err != nil {
		t.Fatalf("Export failed: %v", err)
	}

	// Import back
	restored, err := ImportSnapshot(&buf)
	if err != nil {
		t.Fatalf("Import failed: %v", err)
	}

	// Verify metadata
	if restored.Metadata.DatabaseName != original.Metadata.DatabaseName {
		t.Error("Database name mismatch after round trip")
	}
	if restored.Metadata.DatabaseEngine != original.Metadata.DatabaseEngine {
		t.Error("Database engine mismatch after round trip")
	}
	if restored.Metadata.Version != original.Metadata.Version {
		t.Error("Version mismatch after round trip")
	}
	if restored.Metadata.Description != original.Metadata.Description {
		t.Error("Description mismatch after round trip")
	}
	if restored.Metadata.Tags["test"] != "roundtrip" {
		t.Error("Tags mismatch after round trip")
	}

	// Verify schema
	if restored.Schema.Name != original.Schema.Name {
		t.Error("Schema name mismatch after round trip")
	}
	if len(restored.Schema.Schemas) != len(original.Schema.Schemas) {
		t.Error("Schema count mismatch after round trip")
	}
	if len(restored.Schema.Schemas[0].Tables) != len(original.Schema.Schemas[0].Tables) {
		t.Error("Table count mismatch after round trip")
	}
	if len(restored.Schema.Schemas[0].Tables[0].Columns) != 2 {
		t.Error("Column count mismatch after round trip")
	}
}
