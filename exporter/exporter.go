package exporter

import (
	"encoding/json"
	"io"
	"os"

	schemaextract "github.com/nsxbet/sql-schema"
	"gopkg.in/yaml.v3"
)

// ExportJSON exports a database schema to JSON format.
func ExportJSON(schema *schemaextract.DatabaseSchema, writer io.Writer) error {
	encoder := json.NewEncoder(writer)
	encoder.SetIndent("", "  ")
	return encoder.Encode(schema)
}

// ExportJSONFile exports a database schema to a JSON file.
func ExportJSONFile(schema *schemaextract.DatabaseSchema, filename string) error {
	file, err := os.Create(filename)
	if err != nil {
		return err
	}
	defer func() { _ = file.Close() }()

	return ExportJSON(schema, file)
}

// ExportJSONCompact exports a database schema to compact JSON format (no indentation).
func ExportJSONCompact(schema *schemaextract.DatabaseSchema, writer io.Writer) error {
	encoder := json.NewEncoder(writer)
	return encoder.Encode(schema)
}

// ExportYAML exports a database schema to YAML format.
func ExportYAML(schema *schemaextract.DatabaseSchema, writer io.Writer) error {
	encoder := yaml.NewEncoder(writer)
	encoder.SetIndent(2)
	defer func() { _ = encoder.Close() }()
	return encoder.Encode(schema)
}

// ExportYAMLFile exports a database schema to a YAML file.
func ExportYAMLFile(schema *schemaextract.DatabaseSchema, filename string) error {
	file, err := os.Create(filename)
	if err != nil {
		return err
	}
	defer func() { _ = file.Close() }()

	return ExportYAML(schema, file)
}

// ImportJSON imports a database schema from JSON format.
func ImportJSON(reader io.Reader) (*schemaextract.DatabaseSchema, error) {
	var schema schemaextract.DatabaseSchema
	decoder := json.NewDecoder(reader)
	if err := decoder.Decode(&schema); err != nil {
		return nil, err
	}
	return &schema, nil
}

// ImportJSONFile imports a database schema from a JSON file.
func ImportJSONFile(filename string) (*schemaextract.DatabaseSchema, error) {
	file, err := os.Open(filename)
	if err != nil {
		return nil, err
	}
	defer func() { _ = file.Close() }()

	return ImportJSON(file)
}

// ImportYAML imports a database schema from YAML format.
func ImportYAML(reader io.Reader) (*schemaextract.DatabaseSchema, error) {
	var schema schemaextract.DatabaseSchema
	decoder := yaml.NewDecoder(reader)
	if err := decoder.Decode(&schema); err != nil {
		return nil, err
	}
	return &schema, nil
}

// ImportYAMLFile imports a database schema from a YAML file.
func ImportYAMLFile(filename string) (*schemaextract.DatabaseSchema, error) {
	file, err := os.Open(filename)
	if err != nil {
		return nil, err
	}
	defer func() { _ = file.Close() }()

	return ImportYAML(file)
}

// ToJSON converts a database schema to a JSON string.
func ToJSON(schema *schemaextract.DatabaseSchema) (string, error) {
	data, err := json.MarshalIndent(schema, "", "  ")
	if err != nil {
		return "", err
	}
	return string(data), nil
}

// ToJSONCompact converts a database schema to a compact JSON string.
func ToJSONCompact(schema *schemaextract.DatabaseSchema) (string, error) {
	data, err := json.Marshal(schema)
	if err != nil {
		return "", err
	}
	return string(data), nil
}

// ToYAML converts a database schema to a YAML string.
func ToYAML(schema *schemaextract.DatabaseSchema) (string, error) {
	data, err := yaml.Marshal(schema)
	if err != nil {
		return "", err
	}
	return string(data), nil
}

// FromJSON parses a JSON string into a database schema.
func FromJSON(jsonStr string) (*schemaextract.DatabaseSchema, error) {
	var schema schemaextract.DatabaseSchema
	if err := json.Unmarshal([]byte(jsonStr), &schema); err != nil {
		return nil, err
	}
	return &schema, nil
}

// FromYAML parses a YAML string into a database schema.
func FromYAML(yamlStr string) (*schemaextract.DatabaseSchema, error) {
	var schema schemaextract.DatabaseSchema
	if err := yaml.Unmarshal([]byte(yamlStr), &schema); err != nil {
		return nil, err
	}
	return &schema, nil
}

// ExportFormat represents the export format type.
type ExportFormat string

const (
	// FormatJSON represents JSON format
	FormatJSON ExportFormat = "json"
	// FormatJSONCompact represents compact JSON format
	FormatJSONCompact ExportFormat = "json-compact"
	// FormatYAML represents YAML format
	FormatYAML ExportFormat = "yaml"
)

// Export exports a database schema in the specified format.
func Export(schema *schemaextract.DatabaseSchema, format ExportFormat, writer io.Writer) error {
	switch format {
	case FormatJSON:
		return ExportJSON(schema, writer)
	case FormatJSONCompact:
		return ExportJSONCompact(schema, writer)
	case FormatYAML:
		return ExportYAML(schema, writer)
	default:
		return ExportJSON(schema, writer)
	}
}

// ExportFile exports a database schema to a file in the specified format.
func ExportFile(schema *schemaextract.DatabaseSchema, format ExportFormat, filename string) error {
	switch format {
	case FormatJSON, FormatJSONCompact:
		return ExportJSONFile(schema, filename)
	case FormatYAML:
		return ExportYAMLFile(schema, filename)
	default:
		return ExportJSONFile(schema, filename)
	}
}

// CreateSnapshot creates a snapshot from a database schema with metadata.
func CreateSnapshot(schema *schemaextract.DatabaseSchema, metadata schemaextract.SnapshotMetadata) *schemaextract.Snapshot {
	return &schemaextract.Snapshot{
		Metadata: metadata,
		Schema:   schema,
	}
}

// ExportSnapshot exports a snapshot to JSON format.
func ExportSnapshot(snapshot *schemaextract.Snapshot, writer io.Writer) error {
	encoder := json.NewEncoder(writer)
	encoder.SetIndent("", "  ")
	return encoder.Encode(snapshot)
}

// ExportSnapshotFile exports a snapshot to a JSON file.
func ExportSnapshotFile(snapshot *schemaextract.Snapshot, filename string) error {
	file, err := os.Create(filename)
	if err != nil {
		return err
	}
	defer func() { _ = file.Close() }()

	return ExportSnapshot(snapshot, file)
}

// ExportSnapshotYAML exports a snapshot to YAML format.
func ExportSnapshotYAML(snapshot *schemaextract.Snapshot, writer io.Writer) error {
	encoder := yaml.NewEncoder(writer)
	encoder.SetIndent(2)
	defer func() { _ = encoder.Close() }()
	return encoder.Encode(snapshot)
}

// ExportSnapshotYAMLFile exports a snapshot to a YAML file.
func ExportSnapshotYAMLFile(snapshot *schemaextract.Snapshot, filename string) error {
	file, err := os.Create(filename)
	if err != nil {
		return err
	}
	defer func() { _ = file.Close() }()

	return ExportSnapshotYAML(snapshot, file)
}

// ImportSnapshot imports a snapshot from JSON format.
func ImportSnapshot(reader io.Reader) (*schemaextract.Snapshot, error) {
	var snapshot schemaextract.Snapshot
	decoder := json.NewDecoder(reader)
	if err := decoder.Decode(&snapshot); err != nil {
		return nil, err
	}
	return &snapshot, nil
}

// ImportSnapshotFile imports a snapshot from a JSON file.
func ImportSnapshotFile(filename string) (*schemaextract.Snapshot, error) {
	file, err := os.Open(filename)
	if err != nil {
		return nil, err
	}
	defer func() { _ = file.Close() }()

	return ImportSnapshot(file)
}

// ImportSnapshotYAML imports a snapshot from YAML format.
func ImportSnapshotYAML(reader io.Reader) (*schemaextract.Snapshot, error) {
	var snapshot schemaextract.Snapshot
	decoder := yaml.NewDecoder(reader)
	if err := decoder.Decode(&snapshot); err != nil {
		return nil, err
	}
	return &snapshot, nil
}

// ImportSnapshotYAMLFile imports a snapshot from a YAML file.
func ImportSnapshotYAMLFile(filename string) (*schemaextract.Snapshot, error) {
	file, err := os.Open(filename)
	if err != nil {
		return nil, err
	}
	defer func() { _ = file.Close() }()

	return ImportSnapshotYAML(file)
}
