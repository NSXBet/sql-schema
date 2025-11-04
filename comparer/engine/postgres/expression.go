package postgres

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/nsxbet/sql-schema/comparer/engine"
)

// ExpressionComparer implements PostgreSQL-specific expression comparison.
type ExpressionComparer struct {
	engine.ExpressionComparer
}

// NewExpressionComparer creates a new PostgreSQL expression comparer.
func NewExpressionComparer() *ExpressionComparer {
	return &ExpressionComparer{}
}

// CompareExpressionsSemantically compares two PostgreSQL expressions semantically.
// It normalizes both expressions and uses the parser to compare their ASTs.
func (c *ExpressionComparer) CompareExpressionsSemantically(expr1, expr2 string) (bool, error) {
	// Quick string comparison first
	if strings.TrimSpace(expr1) == strings.TrimSpace(expr2) {
		return true, nil
	}

	// Normalize both expressions
	norm1 := c.NormalizeExpression(expr1)
	norm2 := c.NormalizeExpression(expr2)

	// Compare normalized versions
	if norm1 == norm2 {
		return true, nil
	}

	// Use parser to compare AST if normalization didn't match
	return c.compareUsingParser(norm1, norm2)
}

// NormalizeExpression normalizes a PostgreSQL expression.
// This handles PostgreSQL-specific quirks:
// - IN (...) to ANY(ARRAY[...]) equivalence
// - Interval syntax variations
// - Type cast removal (::type)
// - Extra parentheses
func (c *ExpressionComparer) NormalizeExpression(expr string) string {
	normalized := strings.TrimSpace(expr)

	// Remove type casts (::type)
	normalized = c.removeTypeCasts(normalized)

	// Normalize IN to ANY(ARRAY[...])
	normalized = c.normalizeINToANY(normalized)

	// Normalize interval syntax
	normalized = c.normalizeInterval(normalized)

	// Remove extra parentheses
	normalized = c.removeExtraParentheses(normalized)

	// Normalize whitespace
	normalized = normalizeWhitespace(normalized)

	return strings.TrimSpace(normalized)
}

// removeTypeCasts removes PostgreSQL type casts (::type) from expressions.
func (c *ExpressionComparer) removeTypeCasts(expr string) string {
	// Remove ::type casts
	typeCastRe := regexp.MustCompile(`::[a-zA-Z_][a-zA-Z0-9_]*(\s*\(.*?\))?`)
	return typeCastRe.ReplaceAllString(expr, "")
}

// normalizeINToANY normalizes PostgreSQL IN (...) to ANY(ARRAY[...]) syntax.
// PostgreSQL treats these as equivalent:
//   - column IN (1, 2, 3)
//   - column = ANY(ARRAY[1, 2, 3])
func (c *ExpressionComparer) normalizeINToANY(expr string) string {
	// Pattern: column IN (values)
	inPattern := regexp.MustCompile(`(\w+)\s+IN\s*\((.*?)\)`)

	return inPattern.ReplaceAllStringFunc(expr, func(match string) string {
		submatches := inPattern.FindStringSubmatch(match)
		if len(submatches) != 3 {
			return match
		}
		column := submatches[1]
		values := submatches[2]

		// Convert to ANY(ARRAY[...])
		return fmt.Sprintf("%s = ANY(ARRAY[%s])", column, values)
	})
}

// normalizeInterval normalizes PostgreSQL interval syntax.
// PostgreSQL supports multiple interval formats:
//   - INTERVAL '1 day'
//   - '1 day'::interval
//   - '1 day'
func (c *ExpressionComparer) normalizeInterval(expr string) string {
	// Pattern 1: '1 day'::interval -> INTERVAL '1 day'
	intervalCastRe := regexp.MustCompile(`'([^']+)'::interval`)
	expr = intervalCastRe.ReplaceAllString(expr, "INTERVAL '$1'")

	// Normalize INTERVAL keyword case
	expr = regexp.MustCompile(`(?i)interval`).ReplaceAllString(expr, "INTERVAL")

	return expr
}

// removeExtraParentheses removes unnecessary nested parentheses.
func (c *ExpressionComparer) removeExtraParentheses(expr string) string {
	// Simple heuristic: remove double parentheses ((...))
	for {
		newExpr := regexp.MustCompile(`\(\s*\((.*?)\)\s*\)`).ReplaceAllString(expr, "($1)")
		if newExpr == expr {
			break
		}
		expr = newExpr
	}
	return expr
}

// compareUsingParser uses tokenization to compare two expressions for semantic equivalence.
// This is a lightweight alternative to full AST parsing that handles most common cases.
// It tokenizes SQL expressions and compares token sequences for equivalence.
func (c *ExpressionComparer) compareUsingParser(expr1, expr2 string) (bool, error) {
	// Tokenize both expressions
	tokens1 := tokenizeExpression(expr1)
	tokens2 := tokenizeExpression(expr2)

	// Compare token sequences
	return compareTokenSequences(tokens1, tokens2), nil
}

// tokenizeExpression breaks an expression into semantic tokens.
// This handles SQL operators, identifiers, literals, and keywords.
func tokenizeExpression(expr string) []string {
	var tokens []string
	expr = strings.TrimSpace(expr)

	// Regular expression to match SQL tokens
	// Matches: strings, numbers, identifiers, operators, punctuation
	tokenRe := regexp.MustCompile(`'[^']*'|"[^"]*"|[0-9]+\.?[0-9]*|[a-zA-Z_][a-zA-Z0-9_]*|[<>=!]+|[(),\[\]]|[+\-*/]|\s+`)

	matches := tokenRe.FindAllString(expr, -1)
	for _, match := range matches {
		trimmed := strings.TrimSpace(match)
		if trimmed != "" {
			// Normalize keywords to uppercase
			upper := strings.ToUpper(trimmed)
			if isKeyword(upper) {
				tokens = append(tokens, upper)
			} else {
				tokens = append(tokens, trimmed)
			}
		}
	}

	return tokens
}

// compareTokenSequences compares two token sequences for semantic equivalence.
// This handles normalized token comparison.
func compareTokenSequences(tokens1, tokens2 []string) bool {
	// Quick length check
	if len(tokens1) != len(tokens2) {
		return false
	}

	// Compare token by token
	for i := 0; i < len(tokens1); i++ {
		if !tokensEqual(tokens1[i], tokens2[i]) {
			return false
		}
	}

	return true
}

// tokensEqual checks if two tokens are semantically equal.
// This handles case-insensitive comparison for keywords and operators.
func tokensEqual(token1, token2 string) bool {
	// Direct comparison
	if token1 == token2 {
		return true
	}

	// Case-insensitive comparison for keywords
	if isKeyword(token1) || isKeyword(token2) {
		return strings.EqualFold(token1, token2)
	}

	// Operator equivalence
	if isOperator(token1) && isOperator(token2) {
		return normalizeOperator(token1) == normalizeOperator(token2)
	}

	return false
}

// isKeyword checks if a token is a SQL keyword.
func isKeyword(token string) bool {
	keywords := map[string]bool{
		"SELECT": true, "FROM": true, "WHERE": true, "AND": true, "OR": true,
		"NOT": true, "IN": true, "EXISTS": true, "BETWEEN": true, "LIKE": true,
		"IS": true, "NULL": true, "TRUE": true, "FALSE": true, "CASE": true,
		"WHEN": true, "THEN": true, "ELSE": true, "END": true, "AS": true,
		"JOIN": true, "LEFT": true, "RIGHT": true, "INNER": true, "OUTER": true,
		"ON": true, "USING": true, "UNION": true, "INTERSECT": true, "EXCEPT": true,
		"ORDER": true, "BY": true, "GROUP": true, "HAVING": true, "LIMIT": true,
		"OFFSET": true, "DISTINCT": true, "ALL": true, "ANY": true, "SOME": true,
		"ARRAY": true, "INTERVAL": true, "CAST": true, "EXTRACT": true,
	}
	return keywords[strings.ToUpper(token)]
}

// isOperator checks if a token is an operator.
func isOperator(token string) bool {
	operators := []string{"=", "!=", "<>", "<", ">", "<=", ">=", "+", "-", "*", "/", "||"}
	for _, op := range operators {
		if token == op {
			return true
		}
	}
	return false
}

// normalizeOperator normalizes operator representations.
func normalizeOperator(op string) string {
	// Normalize not-equal operators
	if op == "!=" || op == "<>" {
		return "<>"
	}
	return op
}

// compareASTNodes compares two AST nodes for semantic equivalence.
// This is used for deep comparison of complex expressions.
// Note: This is a lightweight implementation. For full AST comparison,
// consider integrating a proper SQL parser library.
func compareASTNodes(node1, node2 any) bool {
	// For now, use string representation comparison
	// This could be enhanced with proper AST traversal in the future
	str1 := fmt.Sprintf("%v", node1)
	str2 := fmt.Sprintf("%v", node2)
	return str1 == str2
}

// normalizeWhitespace normalizes whitespace in expressions.
func normalizeWhitespace(expr string) string {
	// Replace multiple spaces with single space
	spaceRe := regexp.MustCompile(`\s+`)
	expr = spaceRe.ReplaceAllString(expr, " ")

	// Remove spaces around operators
	expr = strings.ReplaceAll(expr, " = ", "=")
	expr = strings.ReplaceAll(expr, " < ", "<")
	expr = strings.ReplaceAll(expr, " > ", ">")
	expr = strings.ReplaceAll(expr, " + ", "+")
	expr = strings.ReplaceAll(expr, " - ", "-")
	expr = strings.ReplaceAll(expr, " * ", "*")
	expr = strings.ReplaceAll(expr, " / ", "/")

	return expr
}

// IsEquivalentExpression checks if two expressions are semantically equivalent
// using PostgreSQL-specific rules.
func (c *ExpressionComparer) IsEquivalentExpression(expr1, expr2 string) bool {
	equal, _ := c.CompareExpressionsSemantically(expr1, expr2)
	return equal
}
