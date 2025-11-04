package mysql

import (
	"strings"

	schemaextract "github.com/nsxbet/sql-schema"
	"github.com/nsxbet/sql-schema/comparer/engine"
	defaultcomparer "github.com/nsxbet/sql-schema/comparer/engine/base"
	"github.com/nsxbet/sql-schema/diff"
)

// Comparer implements MySQL-specific schema comparison.
type Comparer struct {
	*defaultcomparer.BaseComparer
	expressionComparer *ExpressionComparer
	columnComparer     *ColumnComparer
	functionComparer   *FunctionComparer
}

// NewComparer creates a new MySQL comparer.
func NewComparer() *Comparer {
	base := defaultcomparer.NewBaseComparer(engine.MySQL)
	return &Comparer{
		BaseComparer:       base,
		expressionComparer: NewExpressionComparer(),
		columnComparer:     NewColumnComparer(),
		functionComparer:   NewFunctionComparer(),
	}
}

// Expression returns the MySQL expression comparer.
func (c *Comparer) Expression() engine.ExpressionComparer {
	return c.expressionComparer
}

// Column returns the MySQL column comparer.
func (c *Comparer) Column() engine.ColumnComparer {
	return c.columnComparer
}

// Function returns the MySQL function comparer.
func (c *Comparer) Function() engine.FunctionComparer {
	return c.functionComparer
}

// ColumnComparer implements MySQL-specific column comparison.
type ColumnComparer struct {
	*defaultcomparer.BaseColumnComparer
}

// NewColumnComparer creates a new MySQL column comparer.
func NewColumnComparer() *ColumnComparer {
	return &ColumnComparer{
		BaseColumnComparer: &defaultcomparer.BaseColumnComparer{},
	}
}

// IsEquivalentType checks if two MySQL types are semantically equivalent.
// MySQL type aliases:
//   - INT = INTEGER
//   - BOOL = BOOLEAN = TINYINT(1)
//   - CHAR BYTE = BINARY
func (c *ColumnComparer) IsEquivalentType(type1, type2 string) bool {
	norm1 := c.NormalizeType(type1)
	norm2 := c.NormalizeType(type2)

	// Check direct equality first
	if norm1 == norm2 {
		return true
	}

	// Check MySQL type aliases
	aliases := map[string][]string{
		"INTEGER":   {"INT"},
		"BOOLEAN":   {"BOOL", "TINYINT(1)"},
		"BINARY":    {"CHAR BYTE"},
		"VARBINARY": {"VARCHAR BYTE"},
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

// NormalizeType normalizes a MySQL column type.
func (c *ColumnComparer) NormalizeType(colType string) string {
	// MySQL is case-insensitive, so uppercase for consistency
	normalized := strings.ToUpper(strings.TrimSpace(colType))

	// Handle unsigned/zerofill attributes
	normalized = strings.ReplaceAll(normalized, " UNSIGNED", "")
	normalized = strings.ReplaceAll(normalized, " ZEROFILL", "")

	return normalized
}

// CompareColumns compares two MySQL columns.
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
	if col1.OnUpdate != col2.OnUpdate {
		return false
	}

	// Check generated columns
	if (col1.Generation == nil) != (col2.Generation == nil) {
		return false
	}
	if col1.Generation != nil && col2.Generation != nil {
		if !generationsEqual(col1.Generation, col2.Generation) {
			return false
		}
	}

	return true
}

func generationsEqual(gen1, gen2 *schemaextract.Generation) bool {
	return gen1.Type == gen2.Type &&
		gen1.Expression == gen2.Expression
}

// FunctionComparer implements MySQL-specific function comparison.
type FunctionComparer struct {
	*defaultcomparer.BaseFunctionComparer
	expressionComparer *ExpressionComparer
}

// NewFunctionComparer creates a new MySQL function comparer.
func NewFunctionComparer() *FunctionComparer {
	return &FunctionComparer{
		BaseFunctionComparer: &defaultcomparer.BaseFunctionComparer{},
		expressionComparer:   NewExpressionComparer(),
	}
}

// Equal checks if two functions are identical.
func (c *FunctionComparer) Equal(oldFunc, newFunc *schemaextract.Function) bool {
	if oldFunc.Name != newFunc.Name {
		return false
	}

	// Compare SQL mode and collation settings (MySQL-specific)
	if oldFunc.SQLMode != newFunc.SQLMode {
		return false
	}
	if oldFunc.CharacterSetClient != newFunc.CharacterSetClient {
		return false
	}
	if oldFunc.CollationConnection != newFunc.CollationConnection {
		return false
	}

	// Use semantic comparison for definitions
	equal, _ := c.expressionComparer.CompareExpressionsSemantically(
		oldFunc.Definition,
		newFunc.Definition,
	)
	return equal
}

// CompareDetailed provides detailed comparison.
func (c *FunctionComparer) CompareDetailed(oldFunc, newFunc *schemaextract.Function) (*diff.FunctionComparisonResult, error) {
	result := &diff.FunctionComparisonResult{
		ChangedAttributes: []string{},
	}

	// Compare definitions
	equal, _ := c.expressionComparer.CompareExpressionsSemantically(
		oldFunc.Definition,
		newFunc.Definition,
	)
	if !equal {
		result.BodyChanged = true
	}

	// Check MySQL-specific attributes
	if oldFunc.SQLMode != newFunc.SQLMode {
		result.AttributesChanged = true
		result.ChangedAttributes = append(result.ChangedAttributes, "sql_mode")
	}
	if oldFunc.CharacterSetClient != newFunc.CharacterSetClient {
		result.AttributesChanged = true
		result.ChangedAttributes = append(result.ChangedAttributes, "character_set_client")
	}
	if oldFunc.CollationConnection != newFunc.CollationConnection {
		result.AttributesChanged = true
		result.ChangedAttributes = append(result.ChangedAttributes, "collation_connection")
	}

	// MySQL typically requires DROP/CREATE for function changes
	result.CanUseAlterFunction = false

	return result, nil
}

func contains(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}
