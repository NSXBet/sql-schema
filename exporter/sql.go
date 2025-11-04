package exporter

import (
	"fmt"
	"io"
	"os"
	"strings"

	schemaextract "github.com/nsxbet/sql-schema"
	"github.com/nsxbet/sql-schema/comparer/engine"
)

// SQLDialect represents the SQL dialect to use for export.
type SQLDialect string

const (
	// DialectMySQL represents MySQL dialect
	DialectMySQL SQLDialect = "mysql"
	// DialectPostgreSQL represents PostgreSQL dialect
	DialectPostgreSQL SQLDialect = "postgres"
)

// ExportSQL exports a database schema as SQL DDL statements.
func ExportSQL(schema *schemaextract.DatabaseSchema, dialect SQLDialect, writer io.Writer) error {
	ddl, err := GenerateDDL(schema, dialect)
	if err != nil {
		return err
	}
	_, err = writer.Write([]byte(ddl))
	return err
}

// ExportSQLFile exports a database schema to a SQL file.
func ExportSQLFile(schema *schemaextract.DatabaseSchema, dialect SQLDialect, filename string) error {
	file, err := os.Create(filename)
	if err != nil {
		return err
	}
	defer file.Close()

	return ExportSQL(schema, dialect, file)
}

// GenerateDDL generates complete DDL for a database schema.
func GenerateDDL(schema *schemaextract.DatabaseSchema, dialect SQLDialect) (string, error) {
	var sb strings.Builder

	// Header comment
	sb.WriteString("-- Generated SQL DDL\n")
	sb.WriteString(fmt.Sprintf("-- Database: %s\n", schema.Name))
	sb.WriteString(fmt.Sprintf("-- Dialect: %s\n\n", dialect))

	// Database-level settings
	if dialect == DialectPostgreSQL && len(schema.SearchPath) > 0 {
		sb.WriteString(fmt.Sprintf("-- Search path: %s\n\n", strings.Join(schema.SearchPath, ", ")))
	}

	for _, schemaObj := range schema.Schemas {
		// Schema creation (PostgreSQL only)
		if dialect == DialectPostgreSQL && schemaObj.Name != "" && schemaObj.Name != "public" {
			sb.WriteString(fmt.Sprintf("CREATE SCHEMA IF NOT EXISTS %s;\n\n", QuoteIdentifier(schemaObj.Name, dialect)))
		}

		// Extensions (PostgreSQL only)
		for _, ext := range schemaObj.Extensions {
			ddl := GenerateExtensionDDL(ext, dialect)
			sb.WriteString(ddl)
			sb.WriteString("\n")
		}

		// Enum types (PostgreSQL only)
		for _, enum := range schemaObj.EnumTypes {
			ddl := GenerateEnumTypeDDL(enum, schemaObj.Name, dialect)
			sb.WriteString(ddl)
			sb.WriteString("\n")
		}

		// Sequences (PostgreSQL only)
		for _, seq := range schemaObj.Sequences {
			ddl := GenerateSequenceDDL(seq, schemaObj.Name, dialect)
			sb.WriteString(ddl)
			sb.WriteString("\n")
		}

		// Tables
		for _, table := range schemaObj.Tables {
			ddl := GenerateTableDDL(table, schemaObj.Name, dialect)
			sb.WriteString(ddl)
			sb.WriteString("\n")
		}

		// Views
		for _, view := range schemaObj.Views {
			ddl := GenerateViewDDL(view, schemaObj.Name, dialect)
			sb.WriteString(ddl)
			sb.WriteString("\n")
		}

		// Materialized views (PostgreSQL only)
		for _, mview := range schemaObj.MaterializedViews {
			ddl := GenerateMaterializedViewDDL(mview, schemaObj.Name, dialect)
			sb.WriteString(ddl)
			sb.WriteString("\n")
		}

		// Functions
		for _, fn := range schemaObj.Functions {
			ddl := GenerateFunctionDDL(fn, schemaObj.Name, dialect)
			sb.WriteString(ddl)
			sb.WriteString("\n")
		}

		// Procedures
		for _, proc := range schemaObj.Procedures {
			ddl := GenerateProcedureDDL(proc, schemaObj.Name, dialect)
			sb.WriteString(ddl)
			sb.WriteString("\n")
		}

		// Events (MySQL only)
		for _, event := range schemaObj.Events {
			ddl := GenerateEventDDL(event, dialect)
			sb.WriteString(ddl)
			sb.WriteString("\n")
		}

		// Rules (PostgreSQL only)
		for _, rule := range schemaObj.Rules {
			ddl := GenerateRuleDDL(rule, schemaObj.Name, dialect)
			sb.WriteString(ddl)
			sb.WriteString("\n")
		}
	}

	return sb.String(), nil
}

// GenerateTableDDL generates DDL for a table.
func GenerateTableDDL(table *schemaextract.Table, schemaName string, dialect SQLDialect) string {
	var sb strings.Builder

	// Table header
	tableName := QualifiedName(schemaName, table.Name, dialect)
	sb.WriteString(fmt.Sprintf("CREATE TABLE %s (\n", tableName))

	// Columns
	for i, col := range table.Columns {
		sb.WriteString("  ")
		sb.WriteString(GenerateColumnDDL(col, dialect))
		if i < len(table.Columns)-1 || len(table.ForeignKeys) > 0 || len(table.CheckConstraints) > 0 {
			sb.WriteString(",")
		}
		sb.WriteString("\n")
	}

	// Check constraints
	for i, check := range table.CheckConstraints {
		sb.WriteString(fmt.Sprintf("  CONSTRAINT %s CHECK (%s)",
			QuoteIdentifier(check.Name, dialect), check.Expression))
		if i < len(table.CheckConstraints)-1 || len(table.ForeignKeys) > 0 {
			sb.WriteString(",")
		}
		sb.WriteString("\n")
	}

	// Foreign keys
	for i, fk := range table.ForeignKeys {
		sb.WriteString("  ")
		sb.WriteString(GenerateForeignKeyDDL(fk, dialect))
		if i < len(table.ForeignKeys)-1 {
			sb.WriteString(",")
		}
		sb.WriteString("\n")
	}

	sb.WriteString(")")

	// Table options (MySQL)
	if dialect == DialectMySQL {
		if table.Engine != "" {
			sb.WriteString(fmt.Sprintf(" ENGINE=%s", table.Engine))
		}
		if table.Charset != "" {
			sb.WriteString(fmt.Sprintf(" CHARACTER SET=%s", table.Charset))
		}
		if table.Collation != "" {
			sb.WriteString(fmt.Sprintf(" COLLATE=%s", table.Collation))
		}
	}

	sb.WriteString(";\n")

	// Table comment
	if table.Comment != "" {
		if dialect == DialectPostgreSQL {
			sb.WriteString(fmt.Sprintf("COMMENT ON TABLE %s IS %s;\n",
				tableName, QuoteString(table.Comment)))
		}
	}

	// Indexes
	for _, idx := range table.Indexes {
		if !idx.Primary { // Primary key is included in column definition
			sb.WriteString(GenerateIndexDDL(idx, tableName, dialect))
			sb.WriteString("\n")
		}
	}

	// Triggers
	for _, trigger := range table.Triggers {
		sb.WriteString(GenerateTriggerDDL(trigger, tableName, dialect))
		sb.WriteString("\n")
	}

	return sb.String()
}

// GenerateColumnDDL generates DDL for a column.
func GenerateColumnDDL(col *schemaextract.Column, dialect SQLDialect) string {
	var sb strings.Builder

	sb.WriteString(QuoteIdentifier(col.Name, dialect))
	sb.WriteString(" ")
	sb.WriteString(col.Type)

	// Identity (PostgreSQL)
	if col.Identity != nil && dialect == DialectPostgreSQL {
		if col.Identity.Always {
			sb.WriteString(" GENERATED ALWAYS AS IDENTITY")
		} else {
			sb.WriteString(" GENERATED BY DEFAULT AS IDENTITY")
		}
	}

	// Generated column
	if col.Generation != nil {
		if dialect == DialectMySQL {
			sb.WriteString(fmt.Sprintf(" GENERATED ALWAYS AS (%s) %s",
				col.Generation.Expression, col.Generation.Type))
		} else if dialect == DialectPostgreSQL {
			sb.WriteString(fmt.Sprintf(" GENERATED ALWAYS AS (%s) STORED",
				col.Generation.Expression))
		}
	}

	// Nullable
	if !col.Nullable {
		sb.WriteString(" NOT NULL")
	}

	// Default
	if col.Default != "" {
		sb.WriteString(fmt.Sprintf(" DEFAULT %s", col.Default))
	}

	// On update (MySQL)
	if col.OnUpdate != "" && dialect == DialectMySQL {
		sb.WriteString(fmt.Sprintf(" ON UPDATE %s", col.OnUpdate))
	}

	// Column comment
	if col.Comment != "" {
		if dialect == DialectMySQL {
			sb.WriteString(fmt.Sprintf(" COMMENT %s", QuoteString(col.Comment)))
		}
	}

	return sb.String()
}

// GenerateForeignKeyDDL generates DDL for a foreign key.
func GenerateForeignKeyDDL(fk *schemaextract.ForeignKey, dialect SQLDialect) string {
	var sb strings.Builder

	sb.WriteString(fmt.Sprintf("CONSTRAINT %s FOREIGN KEY (%s) REFERENCES %s (%s)",
		QuoteIdentifier(fk.Name, dialect),
		QuoteIdentifierList(fk.Columns, dialect),
		QuoteIdentifier(fk.ReferencedTable, dialect),
		QuoteIdentifierList(fk.ReferencedColumns, dialect)))

	if fk.OnDelete != "" {
		sb.WriteString(fmt.Sprintf(" ON DELETE %s", fk.OnDelete))
	}
	if fk.OnUpdate != "" {
		sb.WriteString(fmt.Sprintf(" ON UPDATE %s", fk.OnUpdate))
	}

	return sb.String()
}

// GenerateIndexDDL generates DDL for an index.
func GenerateIndexDDL(idx *schemaextract.Index, tableName string, dialect SQLDialect) string {
	var sb strings.Builder

	if idx.Unique {
		sb.WriteString("CREATE UNIQUE INDEX ")
	} else {
		sb.WriteString("CREATE INDEX ")
	}

	sb.WriteString(QuoteIdentifier(idx.Name, dialect))
	sb.WriteString(" ON ")
	sb.WriteString(tableName)

	// Index type (PostgreSQL)
	if dialect == DialectPostgreSQL && idx.Type != "" && idx.Type != "BTREE" {
		sb.WriteString(fmt.Sprintf(" USING %s", idx.Type))
	}

	// Index columns/expressions
	sb.WriteString(" (")
	for i, expr := range idx.Expressions {
		sb.WriteString(expr)
		if i < len(idx.Expressions)-1 {
			sb.WriteString(", ")
		}
	}
	sb.WriteString(")")

	// Partial index (PostgreSQL)
	if idx.WhereClause != "" && dialect == DialectPostgreSQL {
		sb.WriteString(fmt.Sprintf(" WHERE %s", idx.WhereClause))
	}

	sb.WriteString(";")

	return sb.String()
}

// GenerateViewDDL generates DDL for a view.
func GenerateViewDDL(view *schemaextract.View, schemaName string, dialect SQLDialect) string {
	viewName := QualifiedName(schemaName, view.Name, dialect)
	return fmt.Sprintf("CREATE VIEW %s AS\n%s;\n", viewName, view.Definition)
}

// GenerateMaterializedViewDDL generates DDL for a materialized view (PostgreSQL).
func GenerateMaterializedViewDDL(mview *schemaextract.MaterializedView, schemaName string, dialect SQLDialect) string {
	if dialect != DialectPostgreSQL {
		return ""
	}
	viewName := QualifiedName(schemaName, mview.Name, dialect)
	return fmt.Sprintf("CREATE MATERIALIZED VIEW %s AS\n%s;\n", viewName, mview.Definition)
}

// GenerateFunctionDDL generates DDL for a function.
func GenerateFunctionDDL(fn *schemaextract.Function, schemaName string, dialect SQLDialect) string {
	return fn.Definition + ";\n"
}

// GenerateProcedureDDL generates DDL for a procedure.
func GenerateProcedureDDL(proc *schemaextract.Procedure, schemaName string, dialect SQLDialect) string {
	return proc.Definition + ";\n"
}

// GenerateTriggerDDL generates DDL for a trigger.
func GenerateTriggerDDL(trigger *schemaextract.Trigger, tableName string, dialect SQLDialect) string {
	return trigger.Body + ";\n"
}

// GenerateSequenceDDL generates DDL for a sequence (PostgreSQL).
func GenerateSequenceDDL(seq *schemaextract.Sequence, schemaName string, dialect SQLDialect) string {
	if dialect != DialectPostgreSQL {
		return ""
	}

	seqName := QualifiedName(schemaName, seq.Name, dialect)
	var sb strings.Builder

	sb.WriteString(fmt.Sprintf("CREATE SEQUENCE %s", seqName))

	if seq.DataType != "" && seq.DataType != "bigint" {
		sb.WriteString(fmt.Sprintf(" AS %s", seq.DataType))
	}

	sb.WriteString(fmt.Sprintf(" START WITH %d INCREMENT BY %d", seq.Start, seq.Increment))

	if seq.Min != 0 {
		sb.WriteString(fmt.Sprintf(" MINVALUE %d", seq.Min))
	}
	if seq.Max != 0 {
		sb.WriteString(fmt.Sprintf(" MAXVALUE %d", seq.Max))
	}
	if seq.Cache != 0 && seq.Cache != 1 {
		sb.WriteString(fmt.Sprintf(" CACHE %d", seq.Cache))
	}
	if seq.Cycle {
		sb.WriteString(" CYCLE")
	}

	sb.WriteString(";\n")

	return sb.String()
}

// GenerateExtensionDDL generates DDL for an extension (PostgreSQL).
func GenerateExtensionDDL(ext *schemaextract.Extension, dialect SQLDialect) string {
	if dialect != DialectPostgreSQL {
		return ""
	}

	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("CREATE EXTENSION IF NOT EXISTS %s", QuoteIdentifier(ext.Name, dialect)))

	if ext.Schema != "" && ext.Schema != "public" {
		sb.WriteString(fmt.Sprintf(" SCHEMA %s", QuoteIdentifier(ext.Schema, dialect)))
	}
	if ext.Version != "" {
		sb.WriteString(fmt.Sprintf(" VERSION %s", QuoteString(ext.Version)))
	}

	sb.WriteString(";\n")

	return sb.String()
}

// GenerateEnumTypeDDL generates DDL for an enum type (PostgreSQL).
func GenerateEnumTypeDDL(enum *schemaextract.EnumType, schemaName string, dialect SQLDialect) string {
	if dialect != DialectPostgreSQL {
		return ""
	}

	enumName := QualifiedName(schemaName, enum.Name, dialect)
	var sb strings.Builder

	sb.WriteString(fmt.Sprintf("CREATE TYPE %s AS ENUM (", enumName))

	for i, val := range enum.Values {
		sb.WriteString(QuoteString(val))
		if i < len(enum.Values)-1 {
			sb.WriteString(", ")
		}
	}

	sb.WriteString(");\n")

	return sb.String()
}

// GenerateEventDDL generates DDL for an event (MySQL).
func GenerateEventDDL(event *schemaextract.Event, dialect SQLDialect) string {
	if dialect != DialectMySQL {
		return ""
	}
	return event.Definition + ";\n"
}

// GenerateRuleDDL generates DDL for a rule (PostgreSQL).
func GenerateRuleDDL(rule *schemaextract.Rule, schemaName string, dialect SQLDialect) string {
	if dialect != DialectPostgreSQL {
		return ""
	}
	return rule.Definition + ";\n"
}

// QuoteIdentifier quotes an identifier for the given dialect.
func QuoteIdentifier(identifier string, dialect SQLDialect) string {
	if dialect == DialectMySQL {
		return "`" + strings.ReplaceAll(identifier, "`", "``") + "`"
	}
	// PostgreSQL
	return "\"" + strings.ReplaceAll(identifier, "\"", "\"\"") + "\""
}

// QuoteIdentifierList quotes a list of identifiers.
func QuoteIdentifierList(identifiers []string, dialect SQLDialect) string {
	quoted := make([]string, len(identifiers))
	for i, id := range identifiers {
		quoted[i] = QuoteIdentifier(id, dialect)
	}
	return strings.Join(quoted, ", ")
}

// QuoteString quotes a string value.
func QuoteString(value string) string {
	return "'" + strings.ReplaceAll(value, "'", "''") + "'"
}

// QualifiedName returns a qualified table/view/function name.
func QualifiedName(schemaName, objectName string, dialect SQLDialect) string {
	if schemaName == "" || (dialect == DialectMySQL) {
		return QuoteIdentifier(objectName, dialect)
	}
	// PostgreSQL with schema
	return QuoteIdentifier(schemaName, dialect) + "." + QuoteIdentifier(objectName, dialect)
}

// DialectFromEngine converts an engine type to SQL dialect.
func DialectFromEngine(eng engine.Engine) SQLDialect {
	if eng == engine.MySQL {
		return DialectMySQL
	}
	return DialectPostgreSQL
}
