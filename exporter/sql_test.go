package exporter

import (
	"bytes"
	"strings"
	"testing"

	schemaextract "github.com/nsxbet/sql-schema"
)

func TestGenerateTableDDL_PostgreSQL(t *testing.T) {
	table := &schemaextract.Table{
		Name: "users",
		Columns: []*schemaextract.Column{
			{Name: "id", Type: "INTEGER", Nullable: false},
			{Name: "name", Type: "VARCHAR(255)", Nullable: false},
			{Name: "email", Type: "VARCHAR(255)", Nullable: true},
		},
		Indexes: []*schemaextract.Index{
			{Name: "idx_email", Type: "BTREE", Expressions: []string{"email"}, Unique: false},
		},
	}

	ddl := GenerateTableDDL(table, "public", DialectPostgreSQL)

	if !strings.Contains(ddl, "CREATE TABLE") {
		t.Error("Expected CREATE TABLE statement")
	}
	if !strings.Contains(ddl, "\"users\"") {
		t.Error("Expected quoted table name")
	}
	if !strings.Contains(ddl, "\"id\"") {
		t.Error("Expected id column")
	}
	if !strings.Contains(ddl, "NOT NULL") {
		t.Error("Expected NOT NULL constraint")
	}
	if !strings.Contains(ddl, "CREATE INDEX") {
		t.Error("Expected index creation")
	}
}

func TestGenerateTableDDL_MySQL(t *testing.T) {
	table := &schemaextract.Table{
		Name:    "users",
		Engine:  "InnoDB",
		Charset: "utf8mb4",
		Columns: []*schemaextract.Column{
			{Name: "id", Type: "INT", Nullable: false},
			{Name: "name", Type: "VARCHAR(255)", Nullable: false},
		},
	}

	ddl := GenerateTableDDL(table, "", DialectMySQL)

	if !strings.Contains(ddl, "CREATE TABLE") {
		t.Error("Expected CREATE TABLE statement")
	}
	if !strings.Contains(ddl, "`users`") {
		t.Error("Expected backtick-quoted table name")
	}
	if !strings.Contains(ddl, "ENGINE=InnoDB") {
		t.Error("Expected ENGINE clause")
	}
	if !strings.Contains(ddl, "CHARACTER SET=utf8mb4") {
		t.Error("Expected CHARACTER SET clause")
	}
}

func TestGenerateColumnDDL(t *testing.T) {
	testCases := []struct {
		name     string
		column   *schemaextract.Column
		dialect  SQLDialect
		expected []string
	}{
		{
			name: "Basic column",
			column: &schemaextract.Column{
				Name:     "id",
				Type:     "INTEGER",
				Nullable: false,
			},
			dialect:  DialectPostgreSQL,
			expected: []string{"\"id\"", "INTEGER", "NOT NULL"},
		},
		{
			name: "Column with default",
			column: &schemaextract.Column{
				Name:     "status",
				Type:     "VARCHAR(50)",
				Nullable: false,
				Default:  "'active'",
			},
			dialect:  DialectPostgreSQL,
			expected: []string{"\"status\"", "VARCHAR(50)", "DEFAULT 'active'"},
		},
		{
			name: "Generated column (MySQL)",
			column: &schemaextract.Column{
				Name:     "full_name",
				Type:     "VARCHAR(511)",
				Nullable: true,
				Generation: &schemaextract.Generation{
					Type:       schemaextract.GenerationTypeVirtual,
					Expression: "CONCAT(first_name, ' ', last_name)",
				},
			},
			dialect:  DialectMySQL,
			expected: []string{"`full_name`", "GENERATED ALWAYS AS", "VIRTUAL"},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			ddl := GenerateColumnDDL(tc.column, tc.dialect)

			for _, exp := range tc.expected {
				if !strings.Contains(ddl, exp) {
					t.Errorf("Expected '%s' in column DDL, got: %s", exp, ddl)
				}
			}
		})
	}
}

func TestGenerateIndexDDL(t *testing.T) {
	idx := &schemaextract.Index{
		Name:        "idx_email",
		Type:        "BTREE",
		Expressions: []string{"email"},
		Unique:      true,
	}

	ddl := GenerateIndexDDL(idx, "\"public\".\"users\"", DialectPostgreSQL)

	if !strings.Contains(ddl, "CREATE UNIQUE INDEX") {
		t.Error("Expected CREATE UNIQUE INDEX")
	}
	if !strings.Contains(ddl, "\"idx_email\"") {
		t.Error("Expected index name")
	}
	if !strings.Contains(ddl, "(email)") {
		t.Error("Expected index expression")
	}
}

func TestGenerateForeignKeyDDL(t *testing.T) {
	fk := &schemaextract.ForeignKey{
		Name:              "fk_user_id",
		Columns:           []string{"user_id"},
		ReferencedTable:   "users",
		ReferencedColumns: []string{"id"},
		OnDelete:          "CASCADE",
		OnUpdate:          "RESTRICT",
	}

	ddl := GenerateForeignKeyDDL(fk, DialectPostgreSQL)

	if !strings.Contains(ddl, "FOREIGN KEY") {
		t.Error("Expected FOREIGN KEY")
	}
	if !strings.Contains(ddl, "REFERENCES") {
		t.Error("Expected REFERENCES")
	}
	if !strings.Contains(ddl, "ON DELETE CASCADE") {
		t.Error("Expected ON DELETE CASCADE")
	}
	if !strings.Contains(ddl, "ON UPDATE RESTRICT") {
		t.Error("Expected ON UPDATE RESTRICT")
	}
}

func TestGenerateSequenceDDL(t *testing.T) {
	seq := &schemaextract.Sequence{
		Name:      "user_id_seq",
		DataType:  "bigint",
		Start:     1,
		Increment: 1,
		Min:       1,
		Max:       9223372036854775807,
		Cache:     20,
		Cycle:     false,
	}

	ddl := GenerateSequenceDDL(seq, "public", DialectPostgreSQL)

	if !strings.Contains(ddl, "CREATE SEQUENCE") {
		t.Error("Expected CREATE SEQUENCE")
	}
	if !strings.Contains(ddl, "\"user_id_seq\"") {
		t.Error("Expected sequence name")
	}
	if !strings.Contains(ddl, "START WITH 1") {
		t.Error("Expected START WITH clause")
	}
	if !strings.Contains(ddl, "INCREMENT BY 1") {
		t.Error("Expected INCREMENT BY clause")
	}
	if !strings.Contains(ddl, "CACHE 20") {
		t.Error("Expected CACHE clause")
	}
}

func TestGenerateEnumTypeDDL(t *testing.T) {
	enum := &schemaextract.EnumType{
		Name:   "status_type",
		Values: []string{"active", "inactive", "pending"},
		Schema: "public",
	}

	ddl := GenerateEnumTypeDDL(enum, "public", DialectPostgreSQL)

	if !strings.Contains(ddl, "CREATE TYPE") {
		t.Error("Expected CREATE TYPE")
	}
	if !strings.Contains(ddl, "AS ENUM") {
		t.Error("Expected AS ENUM")
	}
	if !strings.Contains(ddl, "'active'") {
		t.Error("Expected 'active' value")
	}
	if !strings.Contains(ddl, "'inactive'") {
		t.Error("Expected 'inactive' value")
	}
	if !strings.Contains(ddl, "'pending'") {
		t.Error("Expected 'pending' value")
	}
}

func TestGenerateExtensionDDL(t *testing.T) {
	ext := &schemaextract.Extension{
		Name:    "uuid-ossp",
		Schema:  "public",
		Version: "1.1",
	}

	ddl := GenerateExtensionDDL(ext, DialectPostgreSQL)

	if !strings.Contains(ddl, "CREATE EXTENSION IF NOT EXISTS") {
		t.Error("Expected CREATE EXTENSION IF NOT EXISTS")
	}
	if !strings.Contains(ddl, "\"uuid-ossp\"") {
		t.Error("Expected extension name")
	}
	if !strings.Contains(ddl, "VERSION '1.1'") {
		t.Error("Expected VERSION clause")
	}
}

func TestExportSQL(t *testing.T) {
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

	var buf bytes.Buffer
	err := ExportSQL(schema, DialectPostgreSQL, &buf)
	if err != nil {
		t.Fatalf("ExportSQL failed: %v", err)
	}

	output := buf.String()

	if !strings.Contains(output, "CREATE TABLE") {
		t.Error("Expected CREATE TABLE in output")
	}
	if !strings.Contains(output, "test_db") {
		t.Error("Expected database name in output")
	}
	if !strings.Contains(output, "\"users\"") {
		t.Error("Expected table name in output")
	}
}

func TestQuoteIdentifier(t *testing.T) {
	testCases := []struct {
		identifier string
		dialect    SQLDialect
		expected   string
	}{
		{"users", DialectPostgreSQL, "\"users\""},
		{"users", DialectMySQL, "`users`"},
		{"table-name", DialectPostgreSQL, "\"table-name\""},
		{"table-name", DialectMySQL, "`table-name`"},
		{"my`table", DialectMySQL, "`my``table`"},
		{"my\"table", DialectPostgreSQL, "\"my\"\"table\""},
	}

	for _, tc := range testCases {
		t.Run(tc.identifier+"_"+string(tc.dialect), func(t *testing.T) {
			result := QuoteIdentifier(tc.identifier, tc.dialect)
			if result != tc.expected {
				t.Errorf("Expected %s, got %s", tc.expected, result)
			}
		})
	}
}

func TestQualifiedName(t *testing.T) {
	testCases := []struct {
		schema   string
		object   string
		dialect  SQLDialect
		expected string
	}{
		{"public", "users", DialectPostgreSQL, "\"public\".\"users\""},
		{"", "users", DialectPostgreSQL, "\"users\""},
		{"", "users", DialectMySQL, "`users`"},
		{"myschema", "mytable", DialectPostgreSQL, "\"myschema\".\"mytable\""},
	}

	for _, tc := range testCases {
		t.Run(tc.schema+"_"+tc.object, func(t *testing.T) {
			result := QualifiedName(tc.schema, tc.object, tc.dialect)
			if result != tc.expected {
				t.Errorf("Expected %s, got %s", tc.expected, result)
			}
		})
	}
}

func TestGenerateDDL_CompleteSchema(t *testing.T) {
	schema := &schemaextract.DatabaseSchema{
		Name: "test_db",
		Schemas: []*schemaextract.Schema{
			{
				Name: "public",
				Extensions: []*schemaextract.Extension{
					{Name: "uuid-ossp", Schema: "public"},
				},
				EnumTypes: []*schemaextract.EnumType{
					{Name: "status", Values: []string{"active", "inactive"}},
				},
				Sequences: []*schemaextract.Sequence{
					{Name: "user_id_seq", Start: 1, Increment: 1},
				},
				Tables: []*schemaextract.Table{
					{
						Name: "users",
						Columns: []*schemaextract.Column{
							{Name: "id", Type: "INTEGER", Nullable: false},
							{Name: "name", Type: "VARCHAR(255)", Nullable: false},
						},
						Indexes: []*schemaextract.Index{
							{Name: "idx_name", Expressions: []string{"name"}},
						},
					},
				},
				Views: []*schemaextract.View{
					{Name: "active_users", Definition: "SELECT * FROM users WHERE status = 'active'"},
				},
			},
		},
	}

	ddl, err := GenerateDDL(schema, DialectPostgreSQL)
	if err != nil {
		t.Fatalf("GenerateDDL failed: %v", err)
	}

	// Verify all major components are present
	expectedComponents := []string{
		"CREATE EXTENSION",
		"CREATE TYPE",
		"CREATE SEQUENCE",
		"CREATE TABLE",
		"CREATE INDEX",
		"CREATE VIEW",
	}

	for _, component := range expectedComponents {
		if !strings.Contains(ddl, component) {
			t.Errorf("Expected %s in DDL output", component)
		}
	}
}
