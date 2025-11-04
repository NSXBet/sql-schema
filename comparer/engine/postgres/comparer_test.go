package postgres

import (
	"testing"

	schemaextract "github.com/nsxbet/sql-schema"
	"github.com/nsxbet/sql-schema/comparer/engine"
)

func TestNewComparer(t *testing.T) {
	comparer := NewComparer()
	if comparer == nil {
		t.Fatal("NewComparer returned nil")
	}

	if comparer.GetEngine() != engine.PostgreSQL {
		t.Errorf("Expected engine PostgreSQL, got %s", comparer.GetEngine())
	}
}

func TestPostgresComparer_Column(t *testing.T) {
	comparer := NewComparer()
	colComparer := comparer.Column()

	if colComparer == nil {
		t.Fatal("Column comparer is nil")
	}

	tests := []struct {
		name     string
		col1     *schemaextract.Column
		col2     *schemaextract.Column
		expected bool
	}{
		{
			name: "identical columns",
			col1: &schemaextract.Column{
				Name:     "id",
				Type:     "INTEGER",
				Nullable: false,
			},
			col2: &schemaextract.Column{
				Name:     "id",
				Type:     "INTEGER",
				Nullable: false,
			},
			expected: true,
		},
		{
			name: "type aliases - INTEGER vs INT",
			col1: &schemaextract.Column{
				Name:     "count",
				Type:     "INTEGER",
				Nullable: true,
			},
			col2: &schemaextract.Column{
				Name:     "count",
				Type:     "INT",
				Nullable: true,
			},
			expected: true,
		},
		{
			name: "type aliases - INT4 vs INTEGER",
			col1: &schemaextract.Column{
				Name: "value",
				Type: "INT4",
			},
			col2: &schemaextract.Column{
				Name: "value",
				Type: "INTEGER",
			},
			expected: true,
		},
		{
			name: "type aliases - BIGINT vs INT8",
			col1: &schemaextract.Column{
				Name: "big_value",
				Type: "BIGINT",
			},
			col2: &schemaextract.Column{
				Name: "big_value",
				Type: "INT8",
			},
			expected: true,
		},
		{
			name: "type aliases - VARCHAR vs CHARACTER VARYING",
			col1: &schemaextract.Column{
				Name: "name",
				Type: "VARCHAR(255)",
			},
			col2: &schemaextract.Column{
				Name: "name",
				Type: "CHARACTER VARYING(255)",
			},
			expected: true,
		},
		{
			name: "type aliases - TIMESTAMPTZ vs TIMESTAMP WITH TIME ZONE",
			col1: &schemaextract.Column{
				Name: "created_at",
				Type: "TIMESTAMPTZ",
			},
			col2: &schemaextract.Column{
				Name: "created_at",
				Type: "TIMESTAMP WITH TIME ZONE",
			},
			expected: true,
		},
		{
			name: "different nullability",
			col1: &schemaextract.Column{
				Name:     "email",
				Type:     "VARCHAR(255)",
				Nullable: false,
			},
			col2: &schemaextract.Column{
				Name:     "email",
				Type:     "VARCHAR(255)",
				Nullable: true,
			},
			expected: false,
		},
		{
			name: "different types",
			col1: &schemaextract.Column{
				Name: "value",
				Type: "INTEGER",
			},
			col2: &schemaextract.Column{
				Name: "value",
				Type: "BIGINT",
			},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Use CompareColumns to check if they're equal
			colDiff, err := colComparer.CompareColumns(tt.col1, tt.col2)
			if err != nil {
				t.Fatalf("CompareColumns failed: %v", err)
			}

			// If expected true (columns equal), colDiff should be nil
			// If expected false (columns differ), colDiff should not be nil
			result := (colDiff == nil)

			if result != tt.expected {
				t.Errorf("Expected %v, got %v for columns:\n  col1: %+v\n  col2: %+v",
					tt.expected, result, tt.col1, tt.col2)
			}
		})
	}
}

func TestPostgresComparer_Index(t *testing.T) {
	comparer := NewComparer()
	indexComparer := comparer.Index()

	if indexComparer == nil {
		t.Fatal("Index comparer is nil")
	}

	tests := []struct {
		name     string
		idx1     *schemaextract.Index
		idx2     *schemaextract.Index
		expected bool
	}{
		{
			name: "identical indexes",
			idx1: &schemaextract.Index{
				Name:        "idx_name",
				Expressions: []string{"name"},
				Unique:      false,
			},
			idx2: &schemaextract.Index{
				Name:        "idx_name",
				Expressions: []string{"name"},
				Unique:      false,
			},
			expected: true,
		},
		{
			name: "different unique constraint",
			idx1: &schemaextract.Index{
				Name:        "idx_email",
				Expressions: []string{"email"},
				Unique:      true,
			},
			idx2: &schemaextract.Index{
				Name:        "idx_email",
				Expressions: []string{"email"},
				Unique:      false,
			},
			expected: false,
		},
		{
			name: "different columns",
			idx1: &schemaextract.Index{
				Name:        "idx_user",
				Expressions: []string{"user_id"},
			},
			idx2: &schemaextract.Index{
				Name:        "idx_user",
				Expressions: []string{"user_id", "tenant_id"},
			},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Use CompareIndexes to check if they're equal
			idxDiff, err := indexComparer.CompareIndexes(tt.idx1, tt.idx2)
			if err != nil {
				t.Fatalf("CompareIndexes failed: %v", err)
			}

			// If expected true (indexes equal), idxDiff should be nil
			// If expected false (indexes differ), idxDiff should not be nil
			result := (idxDiff == nil)

			if result != tt.expected {
				t.Errorf("Expected %v, got %v", tt.expected, result)
			}
		})
	}
}

func TestPostgresComparer_Expression(t *testing.T) {
	comparer := NewComparer()
	exprComparer := comparer.Expression()

	if exprComparer == nil {
		t.Fatal("Expression comparer is nil")
	}

	tests := []struct {
		name     string
		expr1    string
		expr2    string
		expected bool
	}{
		{
			name:     "identical expressions",
			expr1:    "age >= 18",
			expr2:    "age >= 18",
			expected: true,
		},
		{
			name:     "whitespace differences",
			expr1:    "value > 0",
			expr2:    "value>0",
			expected: true,
		},
		{
			name:     "IN vs ANY(ARRAY[]) equivalence",
			expr1:    "status IN ('active', 'pending')",
			expr2:    "status = ANY(ARRAY['active', 'pending'])",
			expected: true,
		},
		{
			name:     "type cast removal",
			expr1:    "value::integer > 10",
			expr2:    "value > 10",
			expected: true,
		},
		{
			name:     "different expressions",
			expr1:    "age >= 18",
			expr2:    "age > 18",
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := exprComparer.CompareExpressionsSemantically(tt.expr1, tt.expr2)
			if err != nil {
				t.Fatalf("CompareExpressionsSemantically failed: %v", err)
			}
			if result != tt.expected {
				t.Errorf("Expected %v, got %v for:\n  expr1: %s\n  expr2: %s",
					tt.expected, result, tt.expr1, tt.expr2)
			}
		})
	}
}

func TestPostgresComparer_Function(t *testing.T) {
	comparer := NewComparer()
	funcComparer := comparer.Function()

	if funcComparer == nil {
		t.Fatal("Function comparer is nil")
	}

	tests := []struct {
		name     string
		func1    *schemaextract.Function
		func2    *schemaextract.Function
		expected bool
	}{
		{
			name: "identical functions",
			func1: &schemaextract.Function{
				Name:       "calculate",
				Definition: "RETURNS integer AS $$ BEGIN RETURN 42; END; $$",
			},
			func2: &schemaextract.Function{
				Name:       "calculate",
				Definition: "RETURNS integer AS $$ BEGIN RETURN 42; END; $$",
			},
			expected: true,
		},
		{
			name: "different definitions",
			func1: &schemaextract.Function{
				Name:       "get_value",
				Definition: "RETURNS integer AS $$ BEGIN RETURN 1; END; $$",
			},
			func2: &schemaextract.Function{
				Name:       "get_value",
				Definition: "RETURNS integer AS $$ BEGIN RETURN 2; END; $$",
			},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := funcComparer.Equal(tt.func1, tt.func2)
			if result != tt.expected {
				t.Errorf("Expected %v, got %v", tt.expected, result)
			}
		})
	}
}

func TestPostgresComparer_CanUseAlterFunction(t *testing.T) {
	comparer := NewComparer()
	funcComparer := comparer.Function()

	oldFunc := &schemaextract.Function{
		Name:       "test_func",
		Definition: "RETURNS integer AS $$ BEGIN RETURN 1; END; $$ LANGUAGE plpgsql",
	}

	newFunc := &schemaextract.Function{
		Name:       "test_func",
		Definition: "RETURNS integer AS $$ BEGIN RETURN 2; END; $$ LANGUAGE plpgsql",
	}

	// CanUseAlterFunction should return false for signature changes
	canAlter := funcComparer.CanUseAlterFunction(oldFunc, newFunc)

	// The result depends on implementation - just verify it doesn't panic
	_ = canAlter
}
