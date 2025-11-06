package mysql

import (
	"regexp"
	"strings"

	"github.com/nsxbet/sql-schema/comparer/engine"
)

// ExpressionComparer implements MySQL-specific expression comparison.
type ExpressionComparer struct {
	engine.ExpressionComparer
}

// NewExpressionComparer creates a new MySQL expression comparer.
func NewExpressionComparer() *ExpressionComparer {
	return &ExpressionComparer{}
}

// CompareExpressionsSemantically compares two MySQL expressions semantically.
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

	// Use parser to compare if available
	return c.compareUsingParser(norm1, norm2)
}

// NormalizeExpression normalizes a MySQL expression.
// MySQL-specific normalizations:
// - Backtick removal
// - Case normalization (MySQL is case-insensitive for most identifiers)
// - Function name normalization
func (c *ExpressionComparer) NormalizeExpression(expr string) string {
	normalized := strings.TrimSpace(expr)

	// Remove backticks (MySQL identifier quotes)
	normalized = strings.ReplaceAll(normalized, "`", "")

	// Normalize whitespace
	normalized = normalizeWhitespace(normalized)

	// Normalize string quotes (single and double quotes are equivalent in MySQL)
	normalized = c.normalizeStringQuotes(normalized)

	return strings.TrimSpace(normalized)
}

// normalizeStringQuotes normalizes MySQL string quotes.
// MySQL allows both single and double quotes for strings.
func (c *ExpressionComparer) normalizeStringQuotes(expr string) string {
	// For simplicity, we keep the quotes as-is but could normalize
	// to a standard form if needed
	return expr
}

// compareUsingParser uses tokenization to compare two expressions for semantic equivalence.
// This is a lightweight alternative to full AST parsing that handles most common MySQL cases.
// It tokenizes SQL expressions and compares token sequences for equivalence.
func (c *ExpressionComparer) compareUsingParser(expr1, expr2 string) (bool, error) {
	// Tokenize both expressions
	tokens1 := tokenizeExpression(expr1)
	tokens2 := tokenizeExpression(expr2)

	// Compare token sequences
	return compareTokenSequences(tokens1, tokens2), nil
}

// tokenizeExpression breaks a MySQL expression into semantic tokens.
// This handles SQL operators, identifiers (with backticks), literals, and keywords.
func tokenizeExpression(expr string) []string {
	var tokens []string
	expr = strings.TrimSpace(expr)

	// Regular expression to match MySQL tokens
	// Matches: strings, numbers, backtick identifiers, identifiers, operators, punctuation
	tokenRe := regexp.MustCompile(
		"`[^`]*`|'[^']*'|\"[^\"]*\"|[0-9]+\\.?[0-9]*|[a-zA-Z_][a-zA-Z0-9_]*|[<>=!]+|[(),]|[+\\-*/]|\\s+",
	)

	matches := tokenRe.FindAllString(expr, -1)
	for _, match := range matches {
		trimmed := strings.TrimSpace(match)
		if trimmed != "" {
			// Normalize keywords to uppercase (but preserve identifier case)
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

	// Case-insensitive comparison for keywords (MySQL is case-insensitive for keywords)
	if isKeyword(token1) || isKeyword(token2) {
		return strings.EqualFold(token1, token2)
	}

	// Operator equivalence
	if isOperator(token1) && isOperator(token2) {
		return normalizeOperator(token1) == normalizeOperator(token2)
	}

	// Backtick identifier comparison (backticks can be omitted)
	if strings.HasPrefix(token1, "`") && strings.HasPrefix(token2, "`") {
		return strings.Trim(token1, "`") == strings.Trim(token2, "`")
	}
	if strings.HasPrefix(token1, "`") && !strings.HasPrefix(token2, "`") {
		return strings.Trim(token1, "`") == token2
	}
	if !strings.HasPrefix(token1, "`") && strings.HasPrefix(token2, "`") {
		return token1 == strings.Trim(token2, "`")
	}

	return false
}

// isKeyword checks if a token is a MySQL keyword.
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
		"IF": true, "IFNULL": true, "NULLIF": true, "COALESCE": true,
		"CONCAT": true, "CURRENT_TIMESTAMP": true, "NOW": true,
	}
	return keywords[strings.ToUpper(token)]
}

// isOperator checks if a token is an operator.
func isOperator(token string) bool {
	operators := []string{"=", "!=", "<>", "<", ">", "<=", ">=", "+", "-", "*", "/", "&&", "||"}
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
