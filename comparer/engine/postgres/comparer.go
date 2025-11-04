package postgres

import (
	"strings"

	schemaextract "github.com/nsxbet/sql-schema"
	"github.com/nsxbet/sql-schema/comparer/engine"
	defaultcomparer "github.com/nsxbet/sql-schema/comparer/engine/base"
	"github.com/nsxbet/sql-schema/diff"
)

// Comparer implements PostgreSQL-specific schema comparison.
type Comparer struct {
	*defaultcomparer.BaseComparer
	expressionComparer *ExpressionComparer
	functionComparer   *FunctionComparer
	viewComparer       *ViewComparer
	columnComparer     *ColumnComparer
}

// NewComparer creates a new PostgreSQL comparer.
func NewComparer() *Comparer {
	base := defaultcomparer.NewBaseComparer(engine.PostgreSQL)
	return &Comparer{
		BaseComparer:       base,
		expressionComparer: NewExpressionComparer(),
		functionComparer:   NewFunctionComparer(),
		viewComparer:       NewViewComparer(),
		columnComparer:     NewColumnComparer(),
	}
}

// Expression returns the PostgreSQL expression comparer.
func (c *Comparer) Expression() engine.ExpressionComparer {
	return c.expressionComparer
}

// Function returns the PostgreSQL function comparer.
func (c *Comparer) Function() engine.FunctionComparer {
	return c.functionComparer
}

// View returns the PostgreSQL view comparer.
func (c *Comparer) View() engine.ViewComparer {
	return c.viewComparer
}

// Column returns the PostgreSQL column comparer.
func (c *Comparer) Column() engine.ColumnComparer {
	return c.columnComparer
}

// FunctionComparer implements PostgreSQL-specific function comparison.
type FunctionComparer struct {
	*defaultcomparer.BaseFunctionComparer
	expressionComparer *ExpressionComparer
}

// NewFunctionComparer creates a new PostgreSQL function comparer.
func NewFunctionComparer() *FunctionComparer {
	return &FunctionComparer{
		BaseFunctionComparer: &defaultcomparer.BaseFunctionComparer{},
		expressionComparer:   NewExpressionComparer(),
	}
}

// Equal checks if two functions are identical, using semantic comparison for definitions.
func (c *FunctionComparer) Equal(oldFunc, newFunc *schemaextract.Function) bool {
	if oldFunc.Name != newFunc.Name {
		return false
	}

	// Use semantic comparison for definitions
	equal, _ := c.expressionComparer.CompareExpressionsSemantically(
		oldFunc.Definition,
		newFunc.Definition,
	)
	return equal
}

// CompareDetailed provides detailed comparison with PostgreSQL-specific logic.
func (c *FunctionComparer) CompareDetailed(oldFunc, newFunc *schemaextract.Function) (*diff.FunctionComparisonResult, error) {
	result := &diff.FunctionComparisonResult{}

	// Compare definitions semantically
	equal, _ := c.expressionComparer.CompareExpressionsSemantically(
		oldFunc.Definition,
		newFunc.Definition,
	)

	if !equal {
		result.BodyChanged = true
	}

	// PostgreSQL allows ALTER FUNCTION for body changes if signature stays the same
	result.CanUseAlterFunction = c.CanUseAlterFunction(oldFunc, newFunc)

	return result, nil
}

// GetSignature extracts function signature (name + parameter types).
func (c *FunctionComparer) GetSignature(function *schemaextract.Function) string {
	// Extract signature from definition
	// Format: CREATE [OR REPLACE] FUNCTION name(params) ...
	def := function.Definition
	nameStart := strings.Index(strings.ToUpper(def), "FUNCTION")
	if nameStart == -1 {
		return function.Name
	}

	// Find opening parenthesis
	parenStart := strings.Index(def[nameStart:], "(")
	if parenStart == -1 {
		return function.Name
	}

	// Find closing parenthesis
	parenEnd := strings.Index(def[nameStart+parenStart:], ")")
	if parenEnd == -1 {
		return function.Name
	}

	signature := def[nameStart : nameStart+parenStart+parenEnd+1]
	return strings.TrimSpace(signature)
}

// CanUseAlterFunction determines if ALTER FUNCTION can be used.
// PostgreSQL allows ALTER FUNCTION for body changes.
func (c *FunctionComparer) CanUseAlterFunction(oldFunc, newFunc *schemaextract.Function) bool {
	// Check if signature changed
	oldSig := c.GetSignature(oldFunc)
	newSig := c.GetSignature(newFunc)

	// If signature is the same, we can use ALTER FUNCTION
	return oldSig == newSig
}

// ViewComparer implements PostgreSQL-specific view comparison.
type ViewComparer struct {
	*defaultcomparer.BaseViewComparer
	expressionComparer *ExpressionComparer
}

// NewViewComparer creates a new PostgreSQL view comparer.
func NewViewComparer() *ViewComparer {
	return &ViewComparer{
		BaseViewComparer:   &defaultcomparer.BaseViewComparer{},
		expressionComparer: NewExpressionComparer(),
	}
}

// CompareView compares two views with semantic comparison.
func (c *ViewComparer) CompareView(oldView, newView *schemaextract.View) (*diff.ViewComparisonResult, error) {
	result := &diff.ViewComparisonResult{}

	// Compare definitions semantically
	equal, _ := c.expressionComparer.CompareExpressionsSemantically(
		oldView.Definition,
		newView.Definition,
	)

	if !equal {
		result.DefinitionChanged = true
	}

	if oldView.Comment != newView.Comment {
		result.CommentChanged = true
	}

	// PostgreSQL views usually require recreation for definition changes
	result.RequiresRecreation = result.DefinitionChanged

	return result, nil
}

// ColumnComparer implements PostgreSQL-specific column comparison.
type ColumnComparer struct {
	*defaultcomparer.BaseColumnComparer
}

// NewColumnComparer creates a new PostgreSQL column comparer.
func NewColumnComparer() *ColumnComparer {
	return &ColumnComparer{
		BaseColumnComparer: &defaultcomparer.BaseColumnComparer{},
	}
}

// IsEquivalentType checks if two PostgreSQL types are semantically equivalent.
// Examples:
//   - INTEGER = INT = INT4
//   - CHARACTER VARYING = VARCHAR
//   - TIMESTAMP WITH TIME ZONE = TIMESTAMPTZ
func (c *ColumnComparer) IsEquivalentType(type1, type2 string) bool {
	norm1 := c.NormalizeType(type1)
	norm2 := c.NormalizeType(type2)

	// Check direct equality first
	if norm1 == norm2 {
		return true
	}

	// Check PostgreSQL type aliases
	aliases := map[string][]string{
		"INTEGER":                  {"INT", "INT4"},
		"BIGINT":                   {"INT8"},
		"SMALLINT":                 {"INT2"},
		"REAL":                     {"FLOAT4"},
		"DOUBLE PRECISION":         {"FLOAT8"},
		"CHARACTER VARYING":        {"VARCHAR"},
		"CHARACTER":                {"CHAR"},
		"TIMESTAMP WITH TIME ZONE": {"TIMESTAMPTZ"},
		"TIME WITH TIME ZONE":      {"TIMETZ"},
		"BOOLEAN":                  {"BOOL"},
	}

	// Check if types are in the same alias group
	for canonical, typeList := range aliases {
		isType1 := norm1 == canonical || contains(typeList, norm1)
		isType2 := norm2 == canonical || contains(typeList, norm2)
		if isType1 && isType2 {
			return true
		}
	}

	return false
}

// NormalizeType normalizes a PostgreSQL column type.
func (c *ColumnComparer) NormalizeType(colType string) string {
	normalized := strings.ToUpper(strings.TrimSpace(colType))

	// Remove precision/scale for comparison
	// e.g., VARCHAR(255) -> VARCHAR
	if idx := strings.Index(normalized, "("); idx != -1 {
		baseType := normalized[:idx]
		// But keep it for numeric types where precision matters
		if !isNumericType(baseType) {
			return baseType
		}
	}

	return normalized
}

// CompareColumns compares two PostgreSQL columns.
func (c *ColumnComparer) CompareColumns(oldCol, newCol *schemaextract.Column) (*diff.ColumnDiff, error) {
	var action diff.MetadataDiffAction
	if oldCol == nil && newCol != nil {
		action = diff.MetadataDiffActionCreate
	} else if oldCol != nil && newCol == nil {
		action = diff.MetadataDiffActionDrop
	} else if oldCol != nil && newCol != nil {
		if !c.columnsEqual(oldCol, newCol) {
			action = diff.MetadataDiffActionAlter
		} else {
			return nil, nil
		}
	}

	return &diff.ColumnDiff{
		Action:    action,
		OldColumn: oldCol,
		NewColumn: newCol,
	}, nil
}

func (c *ColumnComparer) columnsEqual(col1, col2 *schemaextract.Column) bool {
	if col1.Name != col2.Name {
		return false
	}
	if !c.IsEquivalentType(col1.Type, col2.Type) {
		return false
	}
	if col1.Nullable != col2.Nullable {
		return false
	}
	if col1.Default != col2.Default {
		return false
	}
	if col1.Comment != col2.Comment {
		return false
	}

	// Check PostgreSQL-specific: Identity columns
	if (col1.Identity == nil) != (col2.Identity == nil) {
		return false
	}
	if col1.Identity != nil && col2.Identity != nil {
		if !identitiesEqual(col1.Identity, col2.Identity) {
			return false
		}
	}

	return true
}

func identitiesEqual(id1, id2 *schemaextract.Identity) bool {
	return id1.Seed == id2.Seed &&
		id1.Increment == id2.Increment &&
		id1.Always == id2.Always &&
		id1.Cycle == id2.Cycle
}

func contains(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}

func isNumericType(baseType string) bool {
	numericTypes := []string{"NUMERIC", "DECIMAL", "DEC"}
	for _, nt := range numericTypes {
		if baseType == nt {
			return true
		}
	}
	return false
}
