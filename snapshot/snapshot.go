// Package snapshot provides utilities for managing database schema snapshots.
package snapshot

import (
	"context"
	"crypto/md5"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	schemaextract "github.com/nsxbet/sql-schema"
	"github.com/nsxbet/sql-schema/comparer"
	"github.com/nsxbet/sql-schema/comparer/engine"
	"github.com/nsxbet/sql-schema/diff"
	"github.com/nsxbet/sql-schema/exporter"
	"github.com/nsxbet/sql-schema/extractor/mysql"
	"github.com/nsxbet/sql-schema/extractor/postgres"
)

// SnapshotOptions contains options for creating a snapshot.
type SnapshotOptions struct {
	Version     string            // Snapshot format version (default: "1.0")
	Tags        map[string]string // User-defined tags
	Description string            // Optional description
	HostInfo    string            // Optional host connection info
	ComputeHash bool              // Whether to compute checksums (default: false)
}

// TakeSnapshot extracts a database schema and creates a snapshot with metadata.
func TakeSnapshot(db *sql.DB, dbName string, eng engine.Engine, opts *SnapshotOptions) (*schemaextract.Snapshot, error) {
	if opts == nil {
		opts = &SnapshotOptions{Version: "1.0"}
	}
	if opts.Version == "" {
		opts.Version = "1.0"
	}

	// Extract schema
	var schema *schemaextract.DatabaseSchema
	var err error

	ctx := context.Background()
	switch eng {
	case engine.MySQL:
		extractor := mysql.NewExtractor(db, dbName)
		schema, err = extractor.ExtractSchema(ctx)
	case engine.PostgreSQL:
		extractor := postgres.NewExtractor(db, dbName)
		schema, err = extractor.ExtractSchema(ctx)
	default:
		return nil, fmt.Errorf("unsupported database engine: %v", eng)
	}

	if err != nil {
		return nil, fmt.Errorf("failed to extract schema: %w", err)
	}

	// Create metadata
	metadata := schemaextract.SnapshotMetadata{
		Timestamp:      time.Now(),
		Version:        opts.Version,
		DatabaseName:   dbName,
		DatabaseEngine: string(eng),
		HostInfo:       opts.HostInfo,
		Tags:           opts.Tags,
		Description:    opts.Description,
	}

	// Compute checksums if requested
	if opts.ComputeHash {
		md5Hash, sha256Hash, err := ComputeSchemaChecksums(schema)
		if err != nil {
			return nil, fmt.Errorf("failed to compute checksums: %w", err)
		}
		metadata.ChecksumMD5 = md5Hash
		metadata.ChecksumSHA256 = sha256Hash
	}

	return exporter.CreateSnapshot(schema, metadata), nil
}

// SaveSnapshot saves a snapshot to a file.
func SaveSnapshot(snapshot *schemaextract.Snapshot, filename string) error {
	ext := strings.ToLower(filepath.Ext(filename))
	switch ext {
	case ".json":
		return exporter.ExportSnapshotFile(snapshot, filename)
	case ".yaml", ".yml":
		return exporter.ExportSnapshotYAMLFile(snapshot, filename)
	default:
		// Default to JSON
		return exporter.ExportSnapshotFile(snapshot, filename)
	}
}

// LoadSnapshot loads a snapshot from a file.
func LoadSnapshot(filename string) (*schemaextract.Snapshot, error) {
	ext := strings.ToLower(filepath.Ext(filename))
	switch ext {
	case ".json":
		return exporter.ImportSnapshotFile(filename)
	case ".yaml", ".yml":
		return exporter.ImportSnapshotYAMLFile(filename)
	default:
		// Try JSON first, then YAML
		snapshot, err := exporter.ImportSnapshotFile(filename)
		if err != nil {
			return exporter.ImportSnapshotYAMLFile(filename)
		}
		return snapshot, nil
	}
}

// CompareSnapshots compares two snapshots and returns the differences.
func CompareSnapshots(old, new *schemaextract.Snapshot, opts *comparer.CompareOptions) (*diff.MetadataDiff, error) {
	if opts == nil {
		// Try to determine engine from snapshot metadata
		eng := engine.PostgreSQL
		if old.Metadata.DatabaseEngine == "mysql" || new.Metadata.DatabaseEngine == "mysql" {
			eng = engine.MySQL
		}
		opts = &comparer.CompareOptions{Engine: eng}
	}

	return comparer.CompareSchemasDetailed(old.Schema, new.Schema, opts)
}

// LoadAndCompare is a convenience function that loads two snapshots and compares them.
func LoadAndCompare(oldFile, newFile string, opts *comparer.CompareOptions) (*diff.MetadataDiff, error) {
	oldSnapshot, err := LoadSnapshot(oldFile)
	if err != nil {
		return nil, fmt.Errorf("failed to load old snapshot: %w", err)
	}

	newSnapshot, err := LoadSnapshot(newFile)
	if err != nil {
		return nil, fmt.Errorf("failed to load new snapshot: %w", err)
	}

	return CompareSnapshots(oldSnapshot, newSnapshot, opts)
}

// ListSnapshots lists all snapshot files in a directory.
func ListSnapshots(directory string) ([]SnapshotInfo, error) {
	entries, err := os.ReadDir(directory)
	if err != nil {
		return nil, err
	}

	var snapshots []SnapshotInfo
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		ext := strings.ToLower(filepath.Ext(entry.Name()))
		if ext != ".json" && ext != ".yaml" && ext != ".yml" {
			continue
		}

		fullPath := filepath.Join(directory, entry.Name())
		info, err := GetSnapshotInfo(fullPath)
		if err != nil {
			// Skip files that aren't valid snapshots
			continue
		}

		snapshots = append(snapshots, *info)
	}

	return snapshots, nil
}

// SnapshotInfo contains summary information about a snapshot file.
type SnapshotInfo struct {
	Filename       string
	Timestamp      time.Time
	Version        string
	DatabaseName   string
	DatabaseEngine string
	Tags           map[string]string
	Description    string
	FileSize       int64
}

// GetSnapshotInfo reads metadata from a snapshot file without loading the full schema.
func GetSnapshotInfo(filename string) (*SnapshotInfo, error) {
	snapshot, err := LoadSnapshot(filename)
	if err != nil {
		return nil, err
	}

	fileInfo, err := os.Stat(filename)
	if err != nil {
		return nil, err
	}

	return &SnapshotInfo{
		Filename:       filename,
		Timestamp:      snapshot.Metadata.Timestamp,
		Version:        snapshot.Metadata.Version,
		DatabaseName:   snapshot.Metadata.DatabaseName,
		DatabaseEngine: snapshot.Metadata.DatabaseEngine,
		Tags:           snapshot.Metadata.Tags,
		Description:    snapshot.Metadata.Description,
		FileSize:       fileInfo.Size(),
	}, nil
}

// ValidateSnapshot validates a snapshot file and returns any errors.
func ValidateSnapshot(filename string) error {
	snapshot, err := LoadSnapshot(filename)
	if err != nil {
		return fmt.Errorf("failed to load snapshot: %w", err)
	}

	// Basic validation
	if snapshot.Schema == nil {
		return fmt.Errorf("snapshot has no schema")
	}
	if snapshot.Metadata.DatabaseName == "" {
		return fmt.Errorf("snapshot has no database name")
	}
	if snapshot.Metadata.DatabaseEngine == "" {
		return fmt.Errorf("snapshot has no database engine")
	}
	if snapshot.Metadata.Version == "" {
		return fmt.Errorf("snapshot has no version")
	}

	// Validate checksums if present
	if snapshot.Metadata.ChecksumMD5 != "" || snapshot.Metadata.ChecksumSHA256 != "" {
		md5Hash, sha256Hash, err := ComputeSchemaChecksums(snapshot.Schema)
		if err != nil {
			return fmt.Errorf("failed to compute checksums for validation: %w", err)
		}

		if snapshot.Metadata.ChecksumMD5 != "" && snapshot.Metadata.ChecksumMD5 != md5Hash {
			return fmt.Errorf("MD5 checksum mismatch: expected %s, got %s",
				snapshot.Metadata.ChecksumMD5, md5Hash)
		}

		if snapshot.Metadata.ChecksumSHA256 != "" && snapshot.Metadata.ChecksumSHA256 != sha256Hash {
			return fmt.Errorf("SHA256 checksum mismatch: expected %s, got %s",
				snapshot.Metadata.ChecksumSHA256, sha256Hash)
		}
	}

	return nil
}

// ComputeSchemaChecksums computes MD5 and SHA256 checksums of a schema.
func ComputeSchemaChecksums(schema *schemaextract.DatabaseSchema) (md5Hash, sha256Hash string, err error) {
	// Serialize schema to JSON (deterministic)
	data, err := json.Marshal(schema)
	if err != nil {
		return "", "", err
	}

	// Compute MD5
	md5Hasher := md5.New()
	md5Hasher.Write(data)
	md5Hash = hex.EncodeToString(md5Hasher.Sum(nil))

	// Compute SHA256
	sha256Hasher := sha256.New()
	sha256Hasher.Write(data)
	sha256Hash = hex.EncodeToString(sha256Hasher.Sum(nil))

	return md5Hash, sha256Hash, nil
}

// ExportSnapshotSQL exports a snapshot as SQL DDL.
func ExportSnapshotSQL(snapshot *schemaextract.Snapshot, filename string) error {
	dialect := exporter.DialectPostgreSQL
	if snapshot.Metadata.DatabaseEngine == "mysql" {
		dialect = exporter.DialectMySQL
	}

	return exporter.ExportSQLFile(snapshot.Schema, dialect, filename)
}
