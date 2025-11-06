// Package schemaextract provides database schema extraction for MySQL and PostgreSQL.
package schemaextract

import "time"

// DatabaseSchema represents the complete schema metadata for a database.
type DatabaseSchema struct {
	Name         string    `json:"name"                   yaml:"name"`
	CharacterSet string    `json:"characterSet,omitempty" yaml:"characterSet,omitempty"` // MySQL only
	Collation    string    `json:"collation,omitempty"    yaml:"collation,omitempty"`
	SearchPath   []string  `json:"searchPath,omitempty"   yaml:"searchPath,omitempty"` // PostgreSQL only
	Schemas      []*Schema `json:"schemas"                yaml:"schemas"`
}

// Schema represents a database schema (namespace).
// MySQL has a single unnamed schema, PostgreSQL can have multiple named schemas.
type Schema struct {
	Name       string       `json:"name"                        yaml:"name"`
	Tables     []*Table     `json:"tables,omitempty"            yaml:"tables,omitempty"`
	Views      []*View      `json:"views,omitempty"             yaml:"views,omitempty"`
	Functions  []*Function  `json:"functions,omitempty"         yaml:"functions,omitempty"`
	Procedures []*Procedure `json:"procedures,omitempty"        yaml:"procedures,omitempty"`
	Events     []*Event     `json:"events,omitempty"            yaml:"events,omitempty"` // MySQL only
	// PostgreSQL-specific
	MaterializedViews []*MaterializedView `json:"materializedViews,omitempty" yaml:"materializedViews,omitempty"`
	Sequences         []*Sequence         `json:"sequences,omitempty"         yaml:"sequences,omitempty"`
	Extensions        []*Extension        `json:"extensions,omitempty"        yaml:"extensions,omitempty"`
	EnumTypes         []*EnumType         `json:"enumTypes,omitempty"         yaml:"enumTypes,omitempty"`
	Rules             []*Rule             `json:"rules,omitempty"             yaml:"rules,omitempty"` // PostgreSQL rules (schema-level)
}

// Table represents a database table.
type Table struct {
	Name             string             `json:"name"                       yaml:"name"`
	Columns          []*Column          `json:"columns"                    yaml:"columns"`
	Indexes          []*Index           `json:"indexes,omitempty"          yaml:"indexes,omitempty"`
	ForeignKeys      []*ForeignKey      `json:"foreignKeys,omitempty"      yaml:"foreignKeys,omitempty"`
	CheckConstraints []*CheckConstraint `json:"checkConstraints,omitempty" yaml:"checkConstraints,omitempty"`
	Triggers         []*Trigger         `json:"triggers,omitempty"         yaml:"triggers,omitempty"`
	Partitions       []*Partition       `json:"partitions,omitempty"       yaml:"partitions,omitempty"`
	Rules            []*Rule            `json:"rules,omitempty"            yaml:"rules,omitempty"` // PostgreSQL table-level rules
	Comment          string             `json:"comment,omitempty"          yaml:"comment,omitempty"`
	// MySQL-specific
	Engine        string `json:"engine,omitempty"           yaml:"engine,omitempty"`
	Collation     string `json:"collation,omitempty"        yaml:"collation,omitempty"`
	RowCount      int64  `json:"rowCount,omitempty"         yaml:"rowCount,omitempty"`
	DataSize      int64  `json:"dataSize,omitempty"         yaml:"dataSize,omitempty"`
	IndexSize     int64  `json:"indexSize,omitempty"        yaml:"indexSize,omitempty"`
	DataFree      int64  `json:"dataFree,omitempty"         yaml:"dataFree,omitempty"`
	CreateOptions string `json:"createOptions,omitempty"    yaml:"createOptions,omitempty"`
	Charset       string `json:"charset,omitempty"          yaml:"charset,omitempty"`
}

// Column represents a table column.
type Column struct {
	Name         string      `json:"name"                   yaml:"name"`
	Position     int32       `json:"position"               yaml:"position"`
	Type         string      `json:"type"                   yaml:"type"`
	Nullable     bool        `json:"nullable"               yaml:"nullable"`
	Default      string      `json:"default,omitempty"      yaml:"default,omitempty"`
	Comment      string      `json:"comment,omitempty"      yaml:"comment,omitempty"`
	CharacterSet string      `json:"characterSet,omitempty" yaml:"characterSet,omitempty"`
	Collation    string      `json:"collation,omitempty"    yaml:"collation,omitempty"`
	OnUpdate     string      `json:"onUpdate,omitempty"     yaml:"onUpdate,omitempty"` // MySQL: ON UPDATE CURRENT_TIMESTAMP
	Generation   *Generation `json:"generation,omitempty"   yaml:"generation,omitempty"`
	Identity     *Identity   `json:"identity,omitempty"     yaml:"identity,omitempty"` // PostgreSQL: IDENTITY columns
}

// Generation represents a generated column.
type Generation struct {
	Type       GenerationType `json:"type"       yaml:"type"` // VIRTUAL or STORED
	Expression string         `json:"expression" yaml:"expression"`
}

// GenerationType indicates how a generated column is stored.
type GenerationType string

const (
	GenerationTypeVirtual GenerationType = "VIRTUAL"
	GenerationTypeStored  GenerationType = "STORED"
)

// Identity represents a PostgreSQL IDENTITY column.
type Identity struct {
	Seed      int64 `json:"seed"               yaml:"seed"`      // Starting value
	Increment int64 `json:"increment"          yaml:"increment"` // Increment value
	Always    bool  `json:"always"             yaml:"always"`    // ALWAYS vs BY DEFAULT
	Cycle     bool  `json:"cycle"              yaml:"cycle"`     // Cycle when reaching max/min
	Cache     int64 `json:"cache,omitempty"    yaml:"cache,omitempty"`
	MinValue  int64 `json:"minValue,omitempty" yaml:"minValue,omitempty"`
	MaxValue  int64 `json:"maxValue,omitempty" yaml:"maxValue,omitempty"`
}

// Index represents a table index.
type Index struct {
	Name        string         `json:"name"                  yaml:"name"`
	Type        string         `json:"type"                  yaml:"type"` // BTREE, HASH, FULLTEXT, SPATIAL, GIN, GIST, etc.
	Expressions []string       `json:"expressions"           yaml:"expressions"`
	Primary     bool           `json:"primary"               yaml:"primary"`
	Unique      bool           `json:"unique"                yaml:"unique"`
	Visible     bool           `json:"visible"               yaml:"visible"`
	Comment     string         `json:"comment,omitempty"     yaml:"comment,omitempty"`
	KeyLength   []int64        `json:"keyLength,omitempty"   yaml:"keyLength,omitempty"` // MySQL: prefix length, -1 means full column
	Descending  []bool         `json:"descending,omitempty"  yaml:"descending,omitempty"`
	WhereClause string         `json:"whereClause,omitempty" yaml:"whereClause,omitempty"` // Partial index condition
	Spatial     *SpatialConfig `json:"spatial,omitempty"     yaml:"spatial,omitempty"`     // Spatial index configuration
}

// SpatialConfig represents spatial index configuration.
type SpatialConfig struct {
	BoundingBox  string `json:"boundingBox,omitempty"  yaml:"boundingBox,omitempty"`  // MySQL: bounding box specification
	Tessellation string `json:"tessellation,omitempty" yaml:"tessellation,omitempty"` // MySQL: tessellation scheme
	Storage      string `json:"storage,omitempty"      yaml:"storage,omitempty"`      // Storage parameters
}

// ForeignKey represents a foreign key constraint.
type ForeignKey struct {
	Name              string   `json:"name"                yaml:"name"`
	Columns           []string `json:"columns"             yaml:"columns"`
	ReferencedTable   string   `json:"referencedTable"     yaml:"referencedTable"`
	ReferencedColumns []string `json:"referencedColumns"   yaml:"referencedColumns"`
	OnDelete          string   `json:"onDelete,omitempty"  yaml:"onDelete,omitempty"` // CASCADE, SET NULL, RESTRICT, NO ACTION
	OnUpdate          string   `json:"onUpdate,omitempty"  yaml:"onUpdate,omitempty"`
	MatchType         string   `json:"matchType,omitempty" yaml:"matchType,omitempty"` // MySQL: SIMPLE, PARTIAL, FULL
}

// CheckConstraint represents a CHECK constraint.
type CheckConstraint struct {
	Name       string `json:"name"       yaml:"name"`
	Expression string `json:"expression" yaml:"expression"`
}

// Trigger represents a database trigger.
type Trigger struct {
	Name   string `json:"name"                          yaml:"name"`
	Event  string `json:"event"                         yaml:"event"`  // INSERT, UPDATE, DELETE
	Timing string `json:"timing"                        yaml:"timing"` // BEFORE, AFTER
	Body   string `json:"body"                          yaml:"body"`
	// MySQL-specific
	SQLMode             string `json:"sqlMode,omitempty"             yaml:"sqlMode,omitempty"`
	CharacterSetClient  string `json:"characterSetClient,omitempty"  yaml:"characterSetClient,omitempty"`
	CollationConnection string `json:"collationConnection,omitempty" yaml:"collationConnection,omitempty"`
	Comment             string `json:"comment,omitempty"             yaml:"comment,omitempty"`
}

// Partition represents a table partition.
type Partition struct {
	Name          string        `json:"name"                    yaml:"name"`
	Type          PartitionType `json:"type"                    yaml:"type"`
	Expression    string        `json:"expression,omitempty"    yaml:"expression,omitempty"`
	Value         string        `json:"value,omitempty"         yaml:"value,omitempty"`
	Subpartitions []*Partition  `json:"subpartitions,omitempty" yaml:"subpartitions,omitempty"`
	UseDefault    string        `json:"useDefault,omitempty"    yaml:"useDefault,omitempty"` // Default partition count
}

// PartitionType represents the partition type.
type PartitionType string

const (
	PartitionTypeRange        PartitionType = "RANGE"
	PartitionTypeRangeColumns PartitionType = "RANGE_COLUMNS"
	PartitionTypeList         PartitionType = "LIST"
	PartitionTypeListColumns  PartitionType = "LIST_COLUMNS"
	PartitionTypeHash         PartitionType = "HASH"
	PartitionTypeKey          PartitionType = "KEY"
	PartitionTypeLinearHash   PartitionType = "LINEAR_HASH"
	PartitionTypeLinearKey    PartitionType = "LINEAR_KEY"
)

// View represents a database view.
type View struct {
	Name       string    `json:"name"              yaml:"name"`
	Definition string    `json:"definition"        yaml:"definition"`
	Comment    string    `json:"comment,omitempty" yaml:"comment,omitempty"`
	Columns    []*Column `json:"columns,omitempty" yaml:"columns,omitempty"`
}

// MaterializedView represents a PostgreSQL materialized view.
type MaterializedView struct {
	Name       string    `json:"name"              yaml:"name"`
	Definition string    `json:"definition"        yaml:"definition"`
	Comment    string    `json:"comment,omitempty" yaml:"comment,omitempty"`
	Columns    []*Column `json:"columns,omitempty" yaml:"columns,omitempty"`
}

// Function represents a database function.
type Function struct {
	Name       string `json:"name"                          yaml:"name"`
	Definition string `json:"definition"                    yaml:"definition"`
	Comment    string `json:"comment,omitempty"             yaml:"comment,omitempty"`
	// MySQL-specific
	SQLMode             string `json:"sqlMode,omitempty"             yaml:"sqlMode,omitempty"`
	CharacterSetClient  string `json:"characterSetClient,omitempty"  yaml:"characterSetClient,omitempty"`
	CollationConnection string `json:"collationConnection,omitempty" yaml:"collationConnection,omitempty"`
	DatabaseCollation   string `json:"databaseCollation,omitempty"   yaml:"databaseCollation,omitempty"`
}

// Procedure represents a database stored procedure.
type Procedure struct {
	Name       string `json:"name"                          yaml:"name"`
	Definition string `json:"definition"                    yaml:"definition"`
	Comment    string `json:"comment,omitempty"             yaml:"comment,omitempty"`
	// MySQL-specific
	SQLMode             string `json:"sqlMode,omitempty"             yaml:"sqlMode,omitempty"`
	CharacterSetClient  string `json:"characterSetClient,omitempty"  yaml:"characterSetClient,omitempty"`
	CollationConnection string `json:"collationConnection,omitempty" yaml:"collationConnection,omitempty"`
	DatabaseCollation   string `json:"databaseCollation,omitempty"   yaml:"databaseCollation,omitempty"`
}

// Event represents a MySQL scheduled event.
type Event struct {
	Name                string `json:"name"                          yaml:"name"`
	Definition          string `json:"definition"                    yaml:"definition"`
	TimeZone            string `json:"timeZone,omitempty"            yaml:"timeZone,omitempty"`
	SQLMode             string `json:"sqlMode,omitempty"             yaml:"sqlMode,omitempty"`
	CharacterSetClient  string `json:"characterSetClient,omitempty"  yaml:"characterSetClient,omitempty"`
	CollationConnection string `json:"collationConnection,omitempty" yaml:"collationConnection,omitempty"`
	Comment             string `json:"comment,omitempty"             yaml:"comment,omitempty"`
}

// Sequence represents a PostgreSQL sequence.
type Sequence struct {
	Name      string `json:"name"              yaml:"name"`
	DataType  string `json:"dataType"          yaml:"dataType"`
	Start     int64  `json:"start"             yaml:"start"`
	Increment int64  `json:"increment"         yaml:"increment"`
	Min       int64  `json:"min"               yaml:"min"`
	Max       int64  `json:"max"               yaml:"max"`
	Cache     int64  `json:"cache"             yaml:"cache"`
	Cycle     bool   `json:"cycle"             yaml:"cycle"`
	Comment   string `json:"comment,omitempty" yaml:"comment,omitempty"`
}

// Extension represents a PostgreSQL extension.
type Extension struct {
	Name    string `json:"name"              yaml:"name"`
	Schema  string `json:"schema"            yaml:"schema"`
	Version string `json:"version"           yaml:"version"`
	Comment string `json:"comment,omitempty" yaml:"comment,omitempty"`
}

// EnumType represents a PostgreSQL enum type.
type EnumType struct {
	Name   string   `json:"name"   yaml:"name"`
	Values []string `json:"values" yaml:"values"`
	Schema string   `json:"schema" yaml:"schema"`
}

// Rule represents a PostgreSQL rule.
type Rule struct {
	Name       string `json:"name"              yaml:"name"`
	Event      string `json:"event"             yaml:"event"`  // SELECT, INSERT, UPDATE, DELETE
	Timing     string `json:"timing"            yaml:"timing"` // INSTEAD, ALSO
	Definition string `json:"definition"        yaml:"definition"`
	Comment    string `json:"comment,omitempty" yaml:"comment,omitempty"`
}

// Snapshot wraps a DatabaseSchema with metadata for tracking and management.
// It provides a point-in-time capture of a database schema with additional context.
type Snapshot struct {
	Metadata SnapshotMetadata `json:"metadata" yaml:"metadata"`
	Schema   *DatabaseSchema  `json:"schema"   yaml:"schema"`
}

// SnapshotMetadata contains tracking information for a schema snapshot.
type SnapshotMetadata struct {
	Timestamp      time.Time         `json:"timestamp"                yaml:"timestamp"`
	Version        string            `json:"version"                  yaml:"version"`                  // Snapshot format version
	DatabaseName   string            `json:"databaseName"             yaml:"databaseName"`             // Name of the database
	DatabaseEngine string            `json:"databaseEngine"           yaml:"databaseEngine"`           // "mysql" or "postgres"
	HostInfo       string            `json:"hostInfo,omitempty"       yaml:"hostInfo,omitempty"`       // Optional host connection info
	Tags           map[string]string `json:"tags,omitempty"           yaml:"tags,omitempty"`           // User-defined tags (e.g., "environment": "production")
	Description    string            `json:"description,omitempty"    yaml:"description,omitempty"`    // User-provided description
	ChecksumMD5    string            `json:"checksumMd5,omitempty"    yaml:"checksumMd5,omitempty"`    // MD5 checksum of schema
	ChecksumSHA256 string            `json:"checksumSha256,omitempty" yaml:"checksumSha256,omitempty"` // SHA256 checksum of schema
}
