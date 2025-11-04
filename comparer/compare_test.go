package comparer

import (
	"testing"

	schemaextract "github.com/nsxbet/sql-schema"
	"github.com/nsxbet/sql-schema/comparer/engine"
	"github.com/nsxbet/sql-schema/diff"
)

func TestCompareSchemasDetailed_EmptySchemas(t *testing.T) {
	oldSchema := &schemaextract.DatabaseSchema{
		Name:    "test_db",
		Schemas: []*schemaextract.Schema{},
	}

	newSchema := &schemaextract.DatabaseSchema{
		Name:    "test_db",
		Schemas: []*schemaextract.Schema{},
	}

	opts := &CompareOptions{
		Engine: engine.PostgreSQL,
	}

	result, err := CompareSchemasDetailed(oldSchema, newSchema, opts)
	if err != nil {
		t.Fatalf("CompareSchemasDetailed failed: %v", err)
	}

	if result == nil {
		t.Fatal("Expected non-nil result")
	}

	if result.DatabaseName != "test_db" {
		t.Errorf("Expected database name 'test_db', got '%s'", result.DatabaseName)
	}

	if len(result.SchemaChanges) != 0 {
		t.Errorf("Expected 0 schema changes, got %d", len(result.SchemaChanges))
	}
}

func TestCompareSchemasDetailed_NewSchema(t *testing.T) {
	oldSchema := &schemaextract.DatabaseSchema{
		Name:    "test_db",
		Schemas: []*schemaextract.Schema{},
	}

	newSchema := &schemaextract.DatabaseSchema{
		Name: "test_db",
		Schemas: []*schemaextract.Schema{
			{
				Name:   "public",
				Tables: []*schemaextract.Table{},
			},
		},
	}

	opts := &CompareOptions{
		Engine: engine.PostgreSQL,
	}

	result, err := CompareSchemasDetailed(oldSchema, newSchema, opts)
	if err != nil {
		t.Fatalf("CompareSchemasDetailed failed: %v", err)
	}

	// Schema changes are reported twice: once from compareSchemasLevel, once from new schema loop
	// This is expected behavior to show the schema creation and its contents
	if len(result.SchemaChanges) < 1 {
		t.Errorf("Expected at least 1 schema change, got %d", len(result.SchemaChanges))
	}

	// Check that at least one of the changes is CREATE for public schema
	foundCreate := false
	for _, change := range result.SchemaChanges {
		if change.Action == diff.MetadataDiffActionCreate && change.SchemaName == "public" {
			foundCreate = true
			break
		}
	}

	if !foundCreate {
		t.Error("Expected to find CREATE action for schema 'public'")
	}
}

func TestCompareSchemasDetailed_NewTable(t *testing.T) {
	oldSchema := &schemaextract.DatabaseSchema{
		Name: "test_db",
		Schemas: []*schemaextract.Schema{
			{
				Name:   "public",
				Tables: []*schemaextract.Table{},
			},
		},
	}

	newSchema := &schemaextract.DatabaseSchema{
		Name: "test_db",
		Schemas: []*schemaextract.Schema{
			{
				Name: "public",
				Tables: []*schemaextract.Table{
					{
						Name: "users",
						Columns: []*schemaextract.Column{
							{
								Name:     "id",
								Type:     "INTEGER",
								Nullable: false,
							},
							{
								Name:     "name",
								Type:     "VARCHAR(255)",
								Nullable: true,
							},
						},
					},
				},
			},
		},
	}

	opts := &CompareOptions{
		Engine: engine.PostgreSQL,
	}

	result, err := CompareSchemasDetailed(oldSchema, newSchema, opts)
	if err != nil {
		t.Fatalf("CompareSchemasDetailed failed: %v", err)
	}

	if len(result.TableChanges) != 1 {
		t.Errorf("Expected 1 table change, got %d", len(result.TableChanges))
	}

	if result.TableChanges[0].Action != diff.MetadataDiffActionCreate {
		t.Errorf("Expected CREATE action, got %s", result.TableChanges[0].Action)
	}

	if result.TableChanges[0].TableName != "users" {
		t.Errorf("Expected table name 'users', got '%s'", result.TableChanges[0].TableName)
	}
}

func TestCompareSchemasDetailed_ModifiedTable(t *testing.T) {
	oldSchema := &schemaextract.DatabaseSchema{
		Name: "test_db",
		Schemas: []*schemaextract.Schema{
			{
				Name: "public",
				Tables: []*schemaextract.Table{
					{
						Name: "users",
						Columns: []*schemaextract.Column{
							{
								Name:     "id",
								Type:     "INTEGER",
								Nullable: false,
							},
						},
					},
				},
			},
		},
	}

	newSchema := &schemaextract.DatabaseSchema{
		Name: "test_db",
		Schemas: []*schemaextract.Schema{
			{
				Name: "public",
				Tables: []*schemaextract.Table{
					{
						Name: "users",
						Columns: []*schemaextract.Column{
							{
								Name:     "id",
								Type:     "INTEGER",
								Nullable: false,
							},
							{
								Name:     "email",
								Type:     "VARCHAR(255)",
								Nullable: true,
							},
						},
					},
				},
			},
		},
	}

	opts := &CompareOptions{
		Engine: engine.PostgreSQL,
	}

	result, err := CompareSchemasDetailed(oldSchema, newSchema, opts)
	if err != nil {
		t.Fatalf("CompareSchemasDetailed failed: %v", err)
	}

	if len(result.TableChanges) != 1 {
		t.Errorf("Expected 1 table change, got %d", len(result.TableChanges))
	}

	if result.TableChanges[0].Action != diff.MetadataDiffActionAlter {
		t.Errorf("Expected ALTER action, got %s", result.TableChanges[0].Action)
	}

	if len(result.TableChanges[0].ColumnChanges) != 1 {
		t.Errorf("Expected 1 column change, got %d", len(result.TableChanges[0].ColumnChanges))
	}

	if result.TableChanges[0].ColumnChanges[0].Action != diff.MetadataDiffActionCreate {
		t.Errorf("Expected CREATE action for column, got %s", result.TableChanges[0].ColumnChanges[0].Action)
	}
}

func TestCompareSchemasDetailed_DroppedTable(t *testing.T) {
	oldSchema := &schemaextract.DatabaseSchema{
		Name: "test_db",
		Schemas: []*schemaextract.Schema{
			{
				Name: "public",
				Tables: []*schemaextract.Table{
					{
						Name: "users",
						Columns: []*schemaextract.Column{
							{
								Name:     "id",
								Type:     "INTEGER",
								Nullable: false,
							},
						},
					},
				},
			},
		},
	}

	newSchema := &schemaextract.DatabaseSchema{
		Name: "test_db",
		Schemas: []*schemaextract.Schema{
			{
				Name:   "public",
				Tables: []*schemaextract.Table{},
			},
		},
	}

	opts := &CompareOptions{
		Engine: engine.PostgreSQL,
	}

	result, err := CompareSchemasDetailed(oldSchema, newSchema, opts)
	if err != nil {
		t.Fatalf("CompareSchemasDetailed failed: %v", err)
	}

	if len(result.TableChanges) != 1 {
		t.Errorf("Expected 1 table change, got %d", len(result.TableChanges))
	}

	if result.TableChanges[0].Action != diff.MetadataDiffActionDrop {
		t.Errorf("Expected DROP action, got %s", result.TableChanges[0].Action)
	}

	if result.TableChanges[0].TableName != "users" {
		t.Errorf("Expected table name 'users', got '%s'", result.TableChanges[0].TableName)
	}
}
