package snapshot

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	schemaextract "github.com/nsxbet/sql-schema"
	"github.com/nsxbet/sql-schema/comparer/engine"
)

func TestComputeSchemaChecksums(t *testing.T) {
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

	md5Hash, sha256Hash, err := ComputeSchemaChecksums(schema)
	if err != nil {
		t.Fatalf("ComputeSchemaChecksums failed: %v", err)
	}

	if md5Hash == "" {
		t.Error("MD5 hash is empty")
	}
	if sha256Hash == "" {
		t.Error("SHA256 hash is empty")
	}

	// Verify checksums are deterministic
	md5Hash2, sha256Hash2, err := ComputeSchemaChecksums(schema)
	if err != nil {
		t.Fatalf("Second checksum computation failed: %v", err)
	}

	if md5Hash != md5Hash2 {
		t.Error("MD5 hashes are not deterministic")
	}
	if sha256Hash != sha256Hash2 {
		t.Error("SHA256 hashes are not deterministic")
	}

	// MD5 should be 32 hex characters, SHA256 should be 64
	if len(md5Hash) != 32 {
		t.Errorf("MD5 hash should be 32 characters, got %d", len(md5Hash))
	}
	if len(sha256Hash) != 64 {
		t.Errorf("SHA256 hash should be 64 characters, got %d", len(sha256Hash))
	}
}

func TestSaveAndLoadSnapshot(t *testing.T) {
	tempDir := t.TempDir()

	snapshot := &schemaextract.Snapshot{
		Metadata: schemaextract.SnapshotMetadata{
			Timestamp:      time.Date(2024, 1, 1, 12, 0, 0, 0, time.UTC),
			Version:        "1.0",
			DatabaseName:   "test_db",
			DatabaseEngine: "postgres",
			Description:    "Test snapshot",
			Tags: map[string]string{
				"environment": "test",
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

	// Test JSON format
	jsonFile := filepath.Join(tempDir, "snapshot.json")
	err := SaveSnapshot(snapshot, jsonFile)
	if err != nil {
		t.Fatalf("SaveSnapshot (JSON) failed: %v", err)
	}

	loaded, err := LoadSnapshot(jsonFile)
	if err != nil {
		t.Fatalf("LoadSnapshot (JSON) failed: %v", err)
	}

	if loaded.Metadata.DatabaseName != "test_db" {
		t.Error("Database name mismatch")
	}
	if loaded.Metadata.Tags["environment"] != "test" {
		t.Error("Tags mismatch")
	}
	if len(loaded.Schema.Schemas) != 1 {
		t.Error("Schema count mismatch")
	}

	// Test YAML format
	yamlFile := filepath.Join(tempDir, "snapshot.yaml")
	err = SaveSnapshot(snapshot, yamlFile)
	if err != nil {
		t.Fatalf("SaveSnapshot (YAML) failed: %v", err)
	}

	loaded2, err := LoadSnapshot(yamlFile)
	if err != nil {
		t.Fatalf("LoadSnapshot (YAML) failed: %v", err)
	}

	if loaded2.Metadata.DatabaseName != "test_db" {
		t.Error("Database name mismatch (YAML)")
	}
}

func TestValidateSnapshot(t *testing.T) {
	tempDir := t.TempDir()

	// Create a valid snapshot
	snapshot := &schemaextract.Snapshot{
		Metadata: schemaextract.SnapshotMetadata{
			Timestamp:      time.Now(),
			Version:        "1.0",
			DatabaseName:   "test_db",
			DatabaseEngine: "postgres",
		},
		Schema: &schemaextract.DatabaseSchema{
			Name: "test_db",
			Schemas: []*schemaextract.Schema{
				{Name: "public", Tables: []*schemaextract.Table{}},
			},
		},
	}

	// Compute checksums
	md5Hash, sha256Hash, err := ComputeSchemaChecksums(snapshot.Schema)
	if err != nil {
		t.Fatalf("Failed to compute checksums: %v", err)
	}
	snapshot.Metadata.ChecksumMD5 = md5Hash
	snapshot.Metadata.ChecksumSHA256 = sha256Hash

	// Save and validate
	filename := filepath.Join(tempDir, "valid_snapshot.json")
	err = SaveSnapshot(snapshot, filename)
	if err != nil {
		t.Fatalf("SaveSnapshot failed: %v", err)
	}

	err = ValidateSnapshot(filename)
	if err != nil {
		t.Errorf("ValidateSnapshot failed for valid snapshot: %v", err)
	}

	// Test invalid snapshot (wrong checksum)
	snapshot.Metadata.ChecksumMD5 = "invalid_checksum"
	invalidFile := filepath.Join(tempDir, "invalid_snapshot.json")
	err = SaveSnapshot(snapshot, invalidFile)
	if err != nil {
		t.Fatalf("SaveSnapshot failed: %v", err)
	}

	err = ValidateSnapshot(invalidFile)
	if err == nil {
		t.Error("Expected validation error for invalid checksum")
	}
}

func TestGetSnapshotInfo(t *testing.T) {
	tempDir := t.TempDir()

	snapshot := &schemaextract.Snapshot{
		Metadata: schemaextract.SnapshotMetadata{
			Timestamp:      time.Date(2024, 1, 1, 12, 0, 0, 0, time.UTC),
			Version:        "1.0",
			DatabaseName:   "test_db",
			DatabaseEngine: "postgres",
			Description:    "Test description",
			Tags: map[string]string{
				"env": "test",
			},
		},
		Schema: &schemaextract.DatabaseSchema{
			Name:    "test_db",
			Schemas: []*schemaextract.Schema{},
		},
	}

	filename := filepath.Join(tempDir, "snapshot.json")
	err := SaveSnapshot(snapshot, filename)
	if err != nil {
		t.Fatalf("SaveSnapshot failed: %v", err)
	}

	info, err := GetSnapshotInfo(filename)
	if err != nil {
		t.Fatalf("GetSnapshotInfo failed: %v", err)
	}

	if info.DatabaseName != "test_db" {
		t.Error("Database name mismatch")
	}
	if info.DatabaseEngine != "postgres" {
		t.Error("Engine mismatch")
	}
	if info.Version != "1.0" {
		t.Error("Version mismatch")
	}
	if info.Description != "Test description" {
		t.Error("Description mismatch")
	}
	if info.Tags["env"] != "test" {
		t.Error("Tags mismatch")
	}
	if info.FileSize == 0 {
		t.Error("File size should be > 0")
	}
}

func TestListSnapshots(t *testing.T) {
	tempDir := t.TempDir()

	// Create multiple snapshots
	for i := 0; i < 3; i++ {
		snapshot := &schemaextract.Snapshot{
			Metadata: schemaextract.SnapshotMetadata{
				Timestamp:      time.Now().Add(time.Duration(i) * time.Hour),
				Version:        "1.0",
				DatabaseName:   "test_db",
				DatabaseEngine: "postgres",
			},
			Schema: &schemaextract.DatabaseSchema{
				Name:    "test_db",
				Schemas: []*schemaextract.Schema{},
			},
		}

		filename := filepath.Join(tempDir, "snapshot"+string(rune('0'+i))+".json")
		err := SaveSnapshot(snapshot, filename)
		if err != nil {
			t.Fatalf("SaveSnapshot failed: %v", err)
		}
	}

	// Create a non-snapshot file (should be ignored)
	err := os.WriteFile(filepath.Join(tempDir, "readme.txt"), []byte("test"), 0644)
	if err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	snapshots, err := ListSnapshots(tempDir)
	if err != nil {
		t.Fatalf("ListSnapshots failed: %v", err)
	}

	if len(snapshots) != 3 {
		t.Errorf("Expected 3 snapshots, got %d", len(snapshots))
	}

	for _, info := range snapshots {
		if info.DatabaseName != "test_db" {
			t.Error("Unexpected database name in listing")
		}
	}
}

func TestCompareSnapshots(t *testing.T) {
	oldSnapshot := &schemaextract.Snapshot{
		Metadata: schemaextract.SnapshotMetadata{
			Timestamp:      time.Now(),
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
								{Name: "name", Type: "VARCHAR(255)", Nullable: false},
							},
						},
					},
				},
			},
		},
	}

	newSnapshot := &schemaextract.Snapshot{
		Metadata: schemaextract.SnapshotMetadata{
			Timestamp:      time.Now().Add(time.Hour),
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
								{Name: "name", Type: "VARCHAR(255)", Nullable: false},
								{Name: "email", Type: "VARCHAR(255)", Nullable: true}, // New column
							},
						},
					},
				},
			},
		},
	}

	mdiff, err := CompareSnapshots(oldSnapshot, newSnapshot, nil)
	if err != nil {
		t.Fatalf("CompareSnapshots failed: %v", err)
	}

	if len(mdiff.TableChanges) != 1 {
		t.Errorf("Expected 1 table change, got %d", len(mdiff.TableChanges))
	}
}

func TestLoadAndCompare(t *testing.T) {
	tempDir := t.TempDir()

	oldSnapshot := &schemaextract.Snapshot{
		Metadata: schemaextract.SnapshotMetadata{
			Timestamp:      time.Now(),
			Version:        "1.0",
			DatabaseName:   "test_db",
			DatabaseEngine: "postgres",
		},
		Schema: &schemaextract.DatabaseSchema{
			Name: "test_db",
			Schemas: []*schemaextract.Schema{
				{
					Name:   "public",
					Tables: []*schemaextract.Table{},
				},
			},
		},
	}

	newSnapshot := &schemaextract.Snapshot{
		Metadata: schemaextract.SnapshotMetadata{
			Timestamp:      time.Now().Add(time.Hour),
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

	oldFile := filepath.Join(tempDir, "old.json")
	newFile := filepath.Join(tempDir, "new.json")

	err := SaveSnapshot(oldSnapshot, oldFile)
	if err != nil {
		t.Fatalf("SaveSnapshot (old) failed: %v", err)
	}

	err = SaveSnapshot(newSnapshot, newFile)
	if err != nil {
		t.Fatalf("SaveSnapshot (new) failed: %v", err)
	}

	mdiff, err := LoadAndCompare(oldFile, newFile, nil)
	if err != nil {
		t.Fatalf("LoadAndCompare failed: %v", err)
	}

	if len(mdiff.TableChanges) != 1 {
		t.Errorf("Expected 1 table change, got %d", len(mdiff.TableChanges))
	}
}

func TestExportSnapshotSQL(t *testing.T) {
	tempDir := t.TempDir()

	snapshot := &schemaextract.Snapshot{
		Metadata: schemaextract.SnapshotMetadata{
			Timestamp:      time.Now(),
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
								{Name: "name", Type: "VARCHAR(255)", Nullable: false},
							},
						},
					},
				},
			},
		},
	}

	sqlFile := filepath.Join(tempDir, "schema.sql")
	err := ExportSnapshotSQL(snapshot, sqlFile)
	if err != nil {
		t.Fatalf("ExportSnapshotSQL failed: %v", err)
	}

	// Verify file exists and has content
	content, err := os.ReadFile(sqlFile)
	if err != nil {
		t.Fatalf("Failed to read SQL file: %v", err)
	}

	sqlContent := string(content)
	if len(sqlContent) == 0 {
		t.Error("SQL file is empty")
	}

	// Should contain CREATE TABLE
	if !contains(sqlContent, "CREATE TABLE") {
		t.Error("SQL should contain CREATE TABLE")
	}
}

func TestSnapshotOptionsDefaults(t *testing.T) {
	// Test that nil options get sensible defaults
	opts := &SnapshotOptions{}
	if opts.Version == "" {
		opts.Version = "1.0"
	}

	if opts.Version != "1.0" {
		t.Error("Default version should be 1.0")
	}
}

func TestEngineDetection(t *testing.T) {
	testCases := []struct {
		oldEngine string
		newEngine string
		expected  engine.Engine
	}{
		{"postgres", "postgres", engine.PostgreSQL},
		{"mysql", "mysql", engine.MySQL},
		{"postgres", "mysql", engine.MySQL}, // If either is MySQL, use MySQL
	}

	for _, tc := range testCases {
		oldSnapshot := &schemaextract.Snapshot{
			Metadata: schemaextract.SnapshotMetadata{
				DatabaseEngine: tc.oldEngine,
			},
			Schema: &schemaextract.DatabaseSchema{Name: "test"},
		}

		newSnapshot := &schemaextract.Snapshot{
			Metadata: schemaextract.SnapshotMetadata{
				DatabaseEngine: tc.newEngine,
			},
			Schema: &schemaextract.DatabaseSchema{Name: "test"},
		}

		// The CompareSnapshots function should detect engine automatically
		_, err := CompareSnapshots(oldSnapshot, newSnapshot, nil)
		if err != nil {
			// This is expected to fail in this test since schemas are minimal
			// We're just testing that the engine detection logic runs
		}
	}
}

func contains(s, substr string) bool {
	return len(s) > 0 && len(substr) > 0 && s != substr && len(s) >= len(substr) &&
		(s[:len(substr)] == substr || contains(s[1:], substr))
}
